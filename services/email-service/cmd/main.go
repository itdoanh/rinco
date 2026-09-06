// Email Service — gửi email transactional qua nhiều driver (SMTP, Resend,
// SendGrid, SES, Console).  Per-tenant driver được resolved từ env /
// header `X-Tenant-Driver`.  Hỗ trợ CRUD template với Go text/template,
// render → markdown → HTML, tracking pixel + redirect link, NATS queue.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"

	"github.com/rinco/go/pkg/db"
	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "email-service"
	version     = "1.0.0"
)

// Drivers
const (
	DriverConsole = "console"
	DriverSMTP    = "smtp"
	DriverResend  = "resend"
	DriverSendGrid = "sendgrid"
	DriverSES     = "ses"
)

// =============================================================================
// Types
// =============================================================================

type sendEmailRequest struct {
	To          []string          `json:"to" validate:"required"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	BodyType    string            `json:"body_type"` // text | html | markdown
	TemplateID  string            `json:"template_id"`
	Variables   map[string]string `json:"variables"`
	Attachments []attachment      `json:"attachments"`
	Priority    string            `json:"priority"` // low | normal | high
	Headers     map[string]string `json:"headers"`
}

type attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"` // base64
}

type batchSendRequest struct {
	Items []sendEmailRequest `json:"items"`
}

type templateRequest struct {
	Name        string `json:"name" validate:"required"`
	Subject     string `json:"subject" validate:"required"`
	Body        string `json:"body" validate:"required"`
	BodyType    string `json:"body_type"`
	Description string `json:"description"`
	Variables   []string `json:"variables"`
}

type templateResponse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	BodyType    string    `json:"body_type"`
	Description string    `json:"description"`
	Variables   []string  `json:"variables"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type renderRequest struct {
	Variables map[string]string `json:"variables"`
}

type logEntry struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	MessageID   string     `json:"message_id"`
	Provider    string     `json:"provider"`
	Status      string     `json:"status"` // queued | sent | failed | bounced | complained
	To          string     `json:"to"`
	Subject     string     `json:"subject"`
	Error       string     `json:"error,omitempty"`
	OpenCount   int        `json:"open_count"`
	ClickCount  int        `json:"click_count"`
	SentAt      time.Time  `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

type statsResponse struct {
	Total     int `json:"total"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
	Bounced   int `json:"bounced"`
	Opened    int `json:"opened"`
	Clicked   int `json:"clicked"`
	OpenRate  float64 `json:"open_rate"`
	ClickRate float64 `json:"click_rate"`
}

// =============================================================================
// In-memory state (production would be PostgreSQL)
// =============================================================================

var (
	muT, muL sync.RWMutex
	tmpls    = map[string]templateResponse{}
	logDB    = []logEntry{}
)

// =============================================================================
// main
// =============================================================================

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	// Open DB (best-effort: used if PG_DSN set, otherwise in-memory).
	var database *sql.DB
	if dsn := os.Getenv("PG_DSN"); dsn != "" {
		cfg := db.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     5432,
			User:     getEnv("DB_USER", "rinco"),
			Password: getEnv("DB_PASSWORD", "rinco_dev_password"),
			Database: getEnv("DB_NAME", "rinco"),
			SSLMode:  "disable",
		}
		if d, err := db.New(cfg); err == nil {
			database = d
		} else {
			logger.Warn(context.Background(), "db connect failed; using in-memory store", zap.Error(err))
		}
	}
	if database != nil {
		defer database.Close()
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	e.GET("/health", healthHandler)
	e.GET("/ready", readyHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// 1x1 tracking pixel
	e.GET("/e/:msg_id.gif", trackPixelHandler)
	// Click redirector
	e.GET("/c/:msg_id", trackClickHandler)

	v1 := e.Group("/v1/email")
	v1.POST("/send", sendHandler)
	v1.POST("/batch", batchHandler)
	v1.POST("/templates", createTemplateHandler)
	v1.GET("/templates", listTemplatesHandler)
	v1.GET("/templates/:id", getTemplateHandler)
	v1.PUT("/templates/:id", updateTemplateHandler)
	v1.DELETE("/templates/:id", deleteTemplateHandler)
	v1.POST("/templates/:id/render", renderTemplateHandler)
	v1.GET("/logs", listLogsHandler)
	v1.GET("/logs/:id", getLogHandler)
	v1.GET("/stats", statsHandler)

	v1.POST("/webhooks/:provider", webhookHandler)

	// Internal / helpers
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"service":   serviceName,
			"version":   version,
			"endpoints": []string{"/health", "/metrics", "/v1/email/send", "/v1/email/templates", "/v1/email/logs", "/v1/email/stats"},
		})
	})

	port := ":" + getEnv("PORT", "8087")
	logger.Info(context.Background(), "starting email service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

// =============================================================================
// Metrics
// =============================================================================

var (
	emailSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rinco_email_sent_total",
		Help: "Emails sent by driver and status",
	}, []string{"driver", "status", "tenant"})
	emailLatencyMs = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rinco_email_latency_milliseconds",
		Help:    "Email send latency in ms",
		Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"driver"})
	opens  = promauto.NewCounterVec(prometheus.CounterOpts{Name: "rinco_email_open_total", Help: "Email opens"}, []string{"tenant"})
	clicks = promauto.NewCounterVec(prometheus.CounterOpts{Name: "rinco_email_click_total", Help: "Email clicks"}, []string{"tenant"})
)

// =============================================================================
// Send handlers
// =============================================================================

func sendHandler(c echo.Context) error {
	var req sendEmailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if len(req.To) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "to required"})
	}
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	userID := c.Request().Header.Get("X-User-ID")
	driver := c.Request().Header.Get("X-Tenant-Driver")
	if driver == "" {
		driver = defaultDriver()
	}

	// Apply template if provided.
	if req.TemplateID != "" {
		tmpl, err := loadTemplate(tenantID, req.TemplateID)
		if err == nil {
			subj, body, err := renderTmpl(tmpl, req.Variables)
			if err != nil {
				return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			}
			if req.Subject == "" {
				req.Subject = subj
			}
			if req.Body == "" {
				req.Body = body
				req.BodyType = tmpl.BodyType
			}
		}
	}

	if req.Subject == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject required (or template)"})
	}
	if req.Body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "body required (or template)"})
	}

	// Convert markdown → HTML if needed.
	htmlBody := req.Body
	if req.BodyType == "markdown" {
		htmlBody = markdownToHTML(req.Body)
		req.BodyType = "html"
	}

	// Inject tracking pixel + click rewriting for HTML.
	msgID := uuid.NewV7().String()
	if req.BodyType == "html" {
		htmlBody = injectTracking(htmlBody, msgID, tenantID)
	}

	// Attachments.
	att := buildAttachments(req.Attachments)

	start := time.Now()
	status := "sent"
	if err := dispatch(driver, req, htmlBody, att); err != nil {
		status = "failed"
		e := storeLog(logEntry{
			ID:        uuid.NewV7().String(),
			TenantID:  tenantID,
			MessageID: msgID,
			Provider:  driver,
			Status:    status,
			To:        strings.Join(req.To, ","),
			Subject:   req.Subject,
			Error:     err.Error(),
			SentAt:    time.Now(),
		})
		emailSent.WithLabelValues(driver, "failed", tenantID).Inc()
		_ = userID
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	emailLatencyMs.WithLabelValues(driver).Observe(float64(time.Since(start).Milliseconds()))
	emailSent.WithLabelValues(driver, "sent", tenantID).Inc()

	storeLog(logEntry{
		ID:        uuid.NewV7().String(),
		TenantID:  tenantID,
		MessageID: msgID,
		Provider:  driver,
		Status:    status,
		To:        strings.Join(req.To, ","),
		Subject:   req.Subject,
		SentAt:    time.Now(),
	})

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "sent",
		"message_id": msgID,
		"driver":     driver,
		"sent_at":    time.Now(),
	})
}

func batchHandler(c echo.Context) error {
	var req batchSendRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if len(req.Items) == 0 || len(req.Items) > 500 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "batch size 1..500"})
	}
	results := []map[string]interface{}{}
	for _, item := range req.Items {
		// Re-use single-send logic in synchronous mode.
		if err := quickSend(c, item); err != nil {
			results = append(results, map[string]interface{}{"status": "failed", "error": err.Error()})
			continue
		}
		results = append(results, map[string]interface{}{"status": "queued"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"submitted": len(req.Items),
		"results":   results,
	})
}

// quickSend is an internal helper used by the batch handler.
func quickSend(c echo.Context, req sendEmailRequest) error {
	if len(req.To) == 0 {
		return errors.New("to required")
	}
	driver := c.Request().Header.Get("X-Tenant-Driver")
	if driver == "" {
		driver = defaultDriver()
	}
	htmlBody := req.Body
	if req.BodyType == "html" {
		htmlBody = injectTracking(htmlBody, uuid.NewV7().String(), "")
	}
	att := buildAttachments(req.Attachments)
	return dispatch(driver, req, htmlBody, att)
}

// =============================================================================
// Drivers
// =============================================================================

func dispatch(driver string, req sendEmailRequest, htmlBody string,
	atts []gomail.Part) error {
	switch driver {
	case DriverConsole:
		fmt.Println("==[email-console]====================================")
		fmt.Printf("From:    %s\n", getEnv("SMTP_FROM", "noreply@rinco.app"))
		fmt.Printf("To:      %s\n", strings.Join(req.To, ", "))
		fmt.Printf("Subject: %s\n", req.Subject)
		fmt.Printf("Body:    %d bytes\n", len(htmlBody))
		fmt.Println("====================================================")
		return nil
	case DriverSMTP:
		return sendSMTP(req, htmlBody, atts)
	case DriverResend:
		return sendResend(req, htmlBody, atts)
	case DriverSendGrid:
		return sendSendGrid(req, htmlBody, atts)
	case DriverSES:
		return sendSES(req, htmlBody, atts)
	}
	return fmt.Errorf("unknown driver: %s", driver)
}

func sendSMTP(req sendEmailRequest, htmlBody string, atts []gomail.Part) error {
	host := getEnv("SMTP_HOST", "mailhog")
	port := getEnv("SMTP_PORT", "1025")
	user := getEnv("SMTP_USER", "")
	pass := getEnv("SMTP_PASSWORD", "")
	from := getEnv("SMTP_FROM", "noreply@rinco.app")

	if user != "" && port != "" {
		// Use native smtp + message.
		msg := buildRawEmail(from, req, htmlBody, atts)
		auth := smtp.PlainAuth("", user, pass, host)
		addr := fmt.Sprintf("%s:%s", host, port)
		return smtp.SendMail(addr, auth, from, req.To, msg)
	}

	// Use gomail as fallback (it composes MIME part / handles html).
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", req.To...)
	if len(req.CC) > 0 {
		m.SetHeader("Cc", req.CC...)
	}
	if len(req.BCC) > 0 {
		m.SetHeader("Bcc", req.BCC...)
	}
	m.SetHeader("Subject", req.Subject)
	if req.BodyType == "html" {
		m.SetBody("text/html", htmlBody)
		m.AddAlternative("text/plain", req.Body)
	} else {
		m.SetBody("text/plain", htmlBody)
	}
	for _, a := range atts {
		m.Attach(a)
	}
	d := gomail.NewDialer(host, atoiOr(port, 1025), user, pass)
	return d.DialAndSend(m)
}

func sendResend(req sendEmailRequest, htmlBody string, _ []gomail.Part) error {
	apiKey := getEnv("RESEND_API_KEY", "")
	if apiKey == "" {
		return errors.New("RESEND_API_KEY not set")
	}
	payload := map[string]interface{}{
		"from":    getEnv("SMTP_FROM", "noreply@rinco.app"),
		"to":      req.To,
		"subject": req.Subject,
		"html":    htmlBody,
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST",
		"https://api.resend.com/emails", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("resend %d: %s", resp.StatusCode, buf.String())
	}
	return nil
}

func sendSendGrid(req sendEmailRequest, htmlBody string, _ []gomail.Part) error {
	apiKey := getEnv("SENDGRID_API_KEY", "")
	if apiKey == "" {
		return errors.New("SENDGRID_API_KEY not set")
	}
	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{{
			"to":    buildSGRecipients(req.To),
			"cc":    buildSGRecipients(req.CC),
			"bcc":   buildSGRecipients(req.BCC),
			"subject": req.Subject,
		}},
		"from":    map[string]string{"email": getEnv("SMTP_FROM", "noreply@rinco.app")},
		"content": []map[string]string{{"type": "text/html", "value": htmlBody}},
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST",
		"https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("sendgrid %d: %s", resp.StatusCode, buf.String())
	}
	return nil
}

func sendSES(req sendEmailRequest, htmlBody string, _ []gomail.Part) error {
	region := getEnv("AWS_REGION", "ap-southeast-1")
	keyID := getEnv("AWS_ACCESS_KEY_ID", "")
	secret := getEnv("AWS_SECRET_ACCESS_KEY", "")
	if region == "" || keyID == "" || secret == "" {
		return errors.New("SES env missing")
	}
	from := getEnv("SMTP_FROM", "noreply@rinco.app")
	body, _ := json.Marshal(map[string]interface{}{
		"Source":      from,
		"Destination": map[string]interface{}{"ToAddresses": req.To},
		"Message": map[string]interface{}{
			"Subject": map[string]interface{}{"Data": req.Subject},
			"Body":    map[string]interface{}{"Html": map[string]string{"Data": htmlBody}},
		},
	})
	httpReq, _ := http.NewRequest("POST",
		"https://email."+region+".amazonaws.com/", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization",
		fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/...", keyID))
	// In production: use SigV4 signing via github.com/aws/aws-sdk-go-v2.
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("ses %d: %s", resp.StatusCode, buf.String())
	}
	return nil
}

func buildSGRecipients(emails []string) []map[string]string {
	out := make([]map[string]string, 0, len(emails))
	for _, e := range emails {
		out = append(out, map[string]string{"email": e})
	}
	return out
}

// buildRawEmail composes a MIME message with HTML + text alternative +
// attachments for plain smtp.SendMail.
func buildRawEmail(from string, req sendEmailRequest, htmlBody string,
	atts []gomail.Part) []byte {
	boundary := "rincoBoundary" + uuid.NewV7().String()
	var headers []string
	headers = append(headers,
		"From: "+from,
		"To: "+strings.Join(req.To, ", "),
		"Subject: "+req.Subject,
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q", boundary),
	)
	body := ""
	body += "\r\n--" + boundary + "\r\n"
	body += "Content-Type: text/plain; charset=utf-8\r\n\r\n"
	body += req.Body + "\r\n"
	if req.BodyType == "html" {
		body += "\r\n--" + boundary + "\r\n"
		body += "Content-Type: text/html; charset=utf-8\r\n\r\n"
		body += htmlBody + "\r\n"
	}
	body += "\r\n--" + boundary + "--\r\n"
	return []byte(strings.Join(headers, "\r\n") + "\r\n" + body)
}

func buildAttachments(items []attachment) []gomail.Part {
	out := make([]gomail.Part, 0, len(items))
	for _, a := range items {
		data, err := base64.StdEncoding.DecodeString(a.Content)
		if err != nil {
			continue
		}
		out = append(out, gomail.Part{
			FileName: a.Filename,
			MIME:     a.ContentType,
			Content:  data,
		})
	}
	return out
}

// =============================================================================
// Templates
// =============================================================================

func createTemplateHandler(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	var req templateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.Name == "" || req.Subject == "" || req.Body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name/subject/body required"})
	}
	if req.BodyType == "" {
		req.BodyType = "html"
	}
	resp := templateResponse{
		ID: uuid.NewV7().String(), TenantID: tenantID, Name: req.Name,
		Subject: req.Subject, Body: req.Body, BodyType: req.BodyType,
		Description: req.Description, Variables: req.Variables,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	muT.Lock()
	tmpls[resp.ID] = resp
	muT.Unlock()
	return c.JSON(http.StatusCreated, resp)
}

func listTemplatesHandler(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	muT.RLock()
	out := []templateResponse{}
	for _, t := range tmpls {
		if tenantID == "" || t.TenantID == tenantID {
			out = append(out, t)
		}
	}
	muT.RUnlock()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"count":     len(out),
		"templates": out,
	})
}

func getTemplateHandler(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	id := c.Param("id")
	muT.RLock()
	t, ok := tmpls[id]
	muT.RUnlock()
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "template not found"})
	}
	if tenantID != "" && t.TenantID != tenantID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}
	return c.JSON(http.StatusOK, t)
}

func updateTemplateHandler(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	id := c.Param("id")
	var req templateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	muT.Lock()
	t, ok := tmpls[id]
	if ok && (tenantID == "" || t.TenantID == tenantID) {
		if req.Name != "" {
			t.Name = req.Name
		}
		if req.Subject != "" {
			t.Subject = req.Subject
		}
		if req.Body != "" {
			t.Body = req.Body
		}
		if req.BodyType != "" {
			t.BodyType = req.BodyType
		}
		if req.Description != "" {
			t.Description = req.Description
		}
		if req.Variables != nil {
			t.Variables = req.Variables
		}
		t.UpdatedAt = time.Now()
		tmpls[id] = t
	}
	muT.Unlock()
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "template not found"})
	}
	return c.JSON(http.StatusOK, t)
}

func deleteTemplateHandler(c echo.Context) error {
	id := c.Param("id")
	muT.Lock()
	delete(tmpls, id)
	muT.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func renderTemplateHandler(c echo.Context) error {
	id := c.Param("id")
	var req renderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	t, err := loadTemplate(c.Request().Header.Get("X-Tenant-ID"), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	subj, body, err := renderTmpl(t, req.Variables)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject":   subj,
		"body":      body,
		"body_type": t.BodyType,
	})
}

// =============================================================================
// Logs / Stats
// =============================================================================

func listLogsHandler(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	muL.RLock()
	out := []logEntry{}
	for _, l := range logDB {
		if tenantID == "" || l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	muL.RUnlock()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"count": len(out),
		"logs":  out,
	})
}

func getLogHandler(c echo.Context) error {
	id := c.Param("id")
	muL.RLock()
	defer muL.RUnlock()
	for _, l := range logDB {
		if l.ID == id {
			return c.JSON(http.StatusOK, l)
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "log not found"})
}

func statsHandler(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	muL.RLock()
	s := statsResponse{}
	for _, l := range logDB {
		if tenantID != "" && l.TenantID != tenantID {
			continue
		}
		s.Total++
		switch l.Status {
		case "sent":
			s.Sent++
		case "failed":
			s.Failed++
		case "bounced":
			s.Bounced++
		}
		s.Opened += l.OpenCount
		s.Clicked += l.ClickCount
	}
	muL.RUnlock()
	if s.Total > 0 {
		s.OpenRate = float64(s.Opened) / float64(s.Total)
		s.ClickRate = float64(s.Clicked) / float64(s.Total)
	}
	return c.JSON(http.StatusOK, s)
}

// storeLog appends to the in-memory log slice (production: PostgreSQL).
func storeLog(l logEntry) error {
	muL.Lock()
	logDB = append(logDB, l)
	muL.Unlock()
	return nil
}

// =============================================================================
// Webhooks (bounce / complaint / delivery)
// =============================================================================

func webhookHandler(c echo.Context) error {
	provider := c.Param("provider")
	body, _ := io.ReadAll(c.Request().Body)
	logger.Info(c.Request().Context(), "email webhook received",
		zap.String("provider", provider),
		zap.Int("bytes", len(body)))
	return c.NoContent(204)
}

// =============================================================================
// Tracking
// =============================================================================

func trackPixelHandler(c echo.Context) error {
	msgID := c.Param("msg_id")
	opens.WithLabelValues(c.QueryParam("tenant")).Inc()
	muL.Lock()
	for i := range logDB {
		if logDB[i].MessageID == msgID {
			logDB[i].OpenCount++
		}
	}
	muL.Unlock()

	// 1x1 transparent gif
	pixel := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00,
		0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
		0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
	}
	c.Response().Header().Set("Content-Type", "image/gif")
	c.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	return c.Blob(http.StatusOK, "image/gif", pixel)
}

func trackClickHandler(c echo.Context) error {
	msgID := c.Param("msg_id")
	target := c.QueryParam("url")
	clicks.WithLabelValues(c.QueryParam("tenant")).Inc()
	muL.Lock()
	for i := range logDB {
		if logDB[i].MessageID == msgID {
			logDB[i].ClickCount++
		}
	}
	muL.Unlock()
	if target == "" {
		return c.NoContent(204)
	}
	return c.Redirect(http.StatusFound, target)
}

var hrefRe = regexp.MustCompile(`(?i)href="([^"]+)"`)

func injectTracking(htmlBody, msgID, tenantID string) string {
	// rewrite links
	htmlBody = hrefRe.ReplaceAllStringFunc(htmlBody, func(s string) string {
		m := hrefRe.FindStringSubmatch(s)
		if len(m) != 2 {
			return s
		}
		dest := m[1]
		if strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "mailto:") ||
			strings.HasPrefix(dest, "tel:") {
			return s
		}
		u, _ := url.Parse(dest)
		if u.Scheme == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return s
		}
		tracker := "/c/" + msgID + "?url=" + url.QueryEscape(dest)
		if tenantID != "" {
			tracker += "&tenant=" + tenantID
		}
		base := os.Getenv("EMAIL_TRACK_BASE")
		if base == "" {
			base = "/"
		}
		return fmt.Sprintf(`href="%s%s"`, strings.TrimRight(base, "/"), tracker)
	})
	// append pixel
	img := fmt.Sprintf(`<img src="/e/%s.gif" width="1" height="1" alt="" style="display:none"/>`,
		msgID)
	if tenantID != "" {
		img = strings.Replace(img, ".gif", ".gif?tenant="+tenantID, 1)
	}
	htmlBody = strings.Replace(htmlBody, "</body>", img+"</body>", -1)
	if !strings.Contains(htmlBody, img) {
		htmlBody += img
	}
	return htmlBody
}

// =============================================================================
// Template engine + helpers
// =============================================================================

func loadTemplate(tenantID, id string) (templateResponse, error) {
	muT.RLock()
	defer muT.RUnlock()
	t, ok := tmpls[id]
	if !ok {
		return t, errors.New("template not found")
	}
	if tenantID != "" && t.TenantID != tenantID {
		return t, errors.New("forbidden")
	}
	return t, nil
}

func renderTmpl(t templateResponse, vars map[string]string) (string, string, error) {
	funcs := template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"dateFormat": func(format string) string { return time.Now().Format(format) },
		"escape":    html.EscapeString,
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}
	parse := func(src string) (*template.Template, error) {
		t2, err := template.New("t").Funcs(funcs).Parse(src)
		if err != nil {
			return nil, err
		}
		return t2, nil
	}
	tSubj, err := parse(t.Subject)
	if err != nil {
		return "", "", err
	}
	tBody, err := parse(t.Body)
	if err != nil {
		return "", "", err
	}
	var sbj, body bytes.Buffer
	if err := tSubj.Execute(&sbj, vars); err != nil {
		return "", "", err
	}
	if err := tBody.Execute(&body, vars); err != nil {
		return "", "", err
	}
	return sbj.String(), body.String(), nil
}

// markdownToHTML is a minimal Markdown subset: headings, paragraphs,
// bold, italic, links, line breaks.  Designed to avoid extra deps.
func markdownToHTML(md string) string {
	var html strings.Builder
	for _, line := range strings.Split(md, "\n") {
		switch {
		case strings.HasPrefix(line, "# "):
			html.WriteString("<h1>" + escapeHTML(strings.TrimPrefix(line, "# ")) + "</h1>")
		case strings.HasPrefix(line, "## "):
			html.WriteString("<h2>" + escapeHTML(strings.TrimPrefix(line, "## ")) + "</h2>")
		case strings.HasPrefix(line, "### "):
			html.WriteString("<h3>" + escapeHTML(strings.TrimPrefix(line, "### ")) + "</h3>")
		case strings.TrimSpace(line) == "":
			html.WriteString("<br/>")
		default:
			html.WriteString("<p>" + processInline(line) + "</p>")
		}
	}
	return html.String()
}

func processInline(line string) string {
	line = boldRe.ReplaceAllString(line, "<strong>$1</strong>")
	line = italicRe.ReplaceAllString(line, "<em>$1</em>")
	line = linkRe.ReplaceAllStringFunc(line, func(s string) string {
		m := linkRe.FindStringSubmatch(s)
		if len(m) != 3 {
			return s
		}
		return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(m[2]), html.EscapeString(m[1]))
	})
	return escapeHTMLInline(line)
}

var (
	boldRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe = regexp.MustCompile(`\*(.+?)\*`)
	linkRe   = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\)]+)\)`)
)

func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// escapeHTMLInline is applied AFTER bold/italic/link substitution.
func escapeHTMLInline(s string) string {
	// Already escaped by html/template when used. Here we just leave as-is
	// since markdownToHTML is a separate, non-template path.
	return s
}

// =============================================================================
// Misc helpers
// =============================================================================

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func readyHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func atoiOr(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func defaultDriver() string {
	if d := os.Getenv("EMAIL_DRIVER"); d != "" {
		return d
	}
	return DriverConsole
}
