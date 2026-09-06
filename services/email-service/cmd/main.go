// Email Service - gửi email transactional qua SMTP.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "email-service"
	version     = "1.0.0"
)

type sendEmailRequest struct {
	To          []string          `json:"to" validate:"required"`
	Subject     string            `json:"subject" validate:"required"`
	Body        string            `json:"body" validate:"required"`
	BodyType    string            `json:"body_type"` // text or html
	TemplateID  string            `json:"template_id"`
	Variables   map[string]string `json:"variables"`
	Attachments []attachment      `json:"attachments"`
}

type attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"` // base64
}

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	e := echo.New()
	e.HideBanner = true
	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	e.GET("/health", healthHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.POST("/v1/email/send", sendEmailHandler)
	e.POST("/v1/email/template/:template_id", sendTemplateHandler)
	e.GET("/v1/email/status/:message_id", getStatusHandler)

	port := ":" + getEnv("PORT", "8087")
	logger.Info(context.Background(), "starting email service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func sendEmailHandler(c echo.Context) error {
	var req sendEmailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if len(req.To) == 0 || req.Subject == "" || req.Body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing required fields"})
	}

	smtpHost := getEnv("SMTP_HOST", "mailhog")
	smtpPort := getEnv("SMTP_PORT", "1025")
	smtpUser := getEnv("SMTP_USER", "")
	smtpPass := getEnv("SMTP_PASSWORD", "")
	from := getEnv("SMTP_FROM", "noreply@rinco.app")

	bodyType := req.BodyType
	if bodyType == "" {
		bodyType = "text"
	}

	contentType := "text/plain"
	if bodyType == "html" {
		contentType = "text/html"
	}

	msg := buildEmail(from, req.To, req.Subject, req.Body, contentType)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	var auth smtp.Auth
	if smtpUser != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	}

	if err := smtp.SendMail(addr, auth, from, req.To, []byte(msg)); err != nil {
		logger.Error(c.Request().Context(), "send mail failed", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "sent",
		"message_id": uuid.NewV7().String(),
		"sent_at":    time.Now(),
	})
}

func sendTemplateHandler(c echo.Context) error {
	templateID := c.Param("template_id")

	var req struct {
		To        []string          `json:"to"`
		Variables map[string]string `json:"variables"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Load template
	subject, body, err := renderTemplate(templateID, req.Variables)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "template not found"})
	}

	// Send via SMTP (same as sendEmailHandler)
	smtpHost := getEnv("SMTP_HOST", "mailhog")
	smtpPort := getEnv("SMTP_PORT", "1025")
	from := getEnv("SMTP_FROM", "noreply@rinco.app")
	msg := buildEmail(from, req.To, subject, body, "text/html")
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	if err := smtp.SendMail(addr, nil, from, req.To, []byte(msg)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "sent"})
}

func getStatusHandler(c echo.Context) error {
	messageID := c.Param("message_id")
	// In production: query from DB
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message_id": messageID,
		"status":     "delivered",
		"sent_at":    time.Now().Add(-1 * time.Minute),
	})
}

func renderTemplate(templateID string, vars map[string]string) (string, string, error) {
	// Simple templates inline
	templates := map[string]struct{ subject, body string }{
		"welcome": {
			subject: "Chào mừng đến với RINCO!",
			body: `<h1>Xin chào {{.Name}}</h1><p>Cảm ơn bạn đã đăng ký tài khoản RINCO.</p>`,
		},
		"reset_password": {
			subject: "Đặt lại mật khẩu RINCO",
			body: `<p>Bấm vào <a href="{{.ResetURL}}">đây</a> để đặt lại mật khẩu.</p>`,
		},
	}

	t, ok := templates[templateID]
	if !ok {
		return "", "", fmt.Errorf("template not found")
	}

	subject := t.subject
	body := t.body
	for k, v := range vars {
		subject = replaceAll(subject, "{{."+k+"}}", v)
		body = replaceAll(body, "{{."+k+"}}", v)
	}

	return subject, body, nil
}

func buildEmail(from string, to []string, subject, body, contentType string) string {
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = joinStrings(to, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = contentType + "; charset=UTF-8"

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body
	return msg
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for _, v := range s[1:] {
		result += sep + v
	}
	return result
}

func replaceAll(s, old, new string) string {
	for {
		i := indexOf(s, old)
		if i < 0 {
			break
		}
		s = s[:i] + new + s[i+len(old):]
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
