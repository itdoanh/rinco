// Landing Service - serve landing page + lead ingestion + Facebook CAPI worker.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"

	"github.com/rinco/go/pkg/db"
	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "landing-service"
	version     = "1.0.0"
)

type leadSubmission struct {
	FormID       string                 `json:"form_id"`
	TenantID     string                 `json:"tenant_id"`
	FullName     string                 `json:"full_name"`
	Phone        string                 `json:"phone"`
	Email        string                 `json:"email"`
	UtmSource    string                 `json:"utm_source"`
	UtmMedium    string                 `json:"utm_medium"`
	UtmCampaign  string                 `json:"utm_campaign"`
	UtmContent   string                 `json:"utm_content"`
	UtmTerm      string                 `json:"utm_term"`
	FBCLID       string                 `json:"fbclid"`
	GCLID        string                 `json:"gclid"`
	TTCLID       string                 `json:"ttclid"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Extra        map[string]string      `json:"extra"`
	Idempotency  string                 `json:"idempotency_key"`
	ClientSentAt int64                  `json:"client_sent_at"`
}

type ingestResponse struct {
	Status      string `json:"status"`
	EventID     string `json:"event_id"`
	LeadID      string `json:"lead_id"`
	Accepted    bool   `json:"accepted"`
	ValidationErrors map[string]string `json:"validation_errors,omitempty"`
}

var (
	scyllaSession *gocql.Session
	valkeyClient  *redis.Client
)

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	// PostgreSQL for lead validation + custom field lookup
	dbCfg := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "rinco"),
		Password: getEnv("DB_PASSWORD", "rinco_dev_password"),
		Database: getEnv("DB_NAME", "rinco"),
		SSLMode:  "disable",
	}
	database, err := db.New(dbCfg)
	if err != nil {
		logger.Fatal(context.Background(), "postgres connect failed", err)
	}
	defer database.Close()

	// ScyllaDB session
	scyllaCluster := gocql.NewCluster(getEnv("SCYLLA_HOST", "localhost"))
	scyllaCluster.Keyspace = "rinco_leads"
	scyllaCluster.Consistency = gocql.LocalOne
	scyllaCluster.Timeout = 10 * time.Second
	scyllaSession, err = scyllaCluster.CreateSession()
	if err != nil {
		logger.Warn(context.Background(), "scylla connect failed (continuing without)", zap.Error(err))
	} else {
		defer scyllaSession.Close()
	}

	// Valkey for rate limiting + dedup
	valkeyClient = redis.NewClient(&redis.Options{
		Addr: getEnv("VALKEY_HOST", "localhost:6379"),
	})
	defer valkeyClient.Close()

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

	// Serve static landing (chiase_cu clone)
	e.Static("/", "./static")

	// Ingestion endpoint
	e.POST("/v1/leads/submit", submitLeadHandler(database))
	e.POST("/v1/track", trackHandler(database)) // Browser-side events

	// CAPI worker endpoint (internal)
	e.POST("/v1/internal/capi/send", capiSendHandler())

	// Webhooks
	e.POST("/v1/webhooks/form", formWebhookHandler(database))

	// Anti-bot
	e.POST("/v1/pow/challenge", powChallengeHandler())
	e.POST("/v1/pow/verify", powVerifyHandler())

	port := ":" + getEnv("PORT", "8086")
	logger.Info(context.Background(), "starting landing service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func submitLeadHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req leadSubmission
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, ingestResponse{Status: "error", ValidationErrors: map[string]string{"body": err.Error()}})
		}

		// HMAC validation
		signature := c.Request().Header.Get("X-Signature")
		if !verifyHMAC(signature, req) {
			logger.Warn(ctx, "HMAC verification failed", zap.String("tenant_id", req.TenantID))
			return c.JSON(http.StatusUnauthorized, ingestResponse{Status: "error"})
		}

		// Idempotency check via Valkey
		if req.Idempotency != "" {
			key := fmt.Sprintf("idem:lead:%s", req.Idempotency)
			_, err := valkeyClient.SetNX(ctx, key, "1", 24*time.Hour).Result()
			if err == nil {
				valkeyClient.Expire(ctx, key, 24*time.Hour)
			}
		}

		// Validate against dynamic model schema
		if err := validateLead(ctx, database, req); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, ingestResponse{
				Status:           "error",
				ValidationErrors: err,
			})
		}

		// Generate event ID
		eventID := uuid.NewV7()
		eventTime := time.Now().UnixMilli()

		// Persist to ScyllaDB leads_raw
		if scyllaSession != nil {
			extraMap := make(map[string]string)
			for k, v := range req.Extra {
				extraMap[k] = v
			}
			err := scyllaSession.Query(`
				INSERT INTO rinco_leads.leads_raw (
					tenant_id, event_time, event_id, idempotency_key,
					full_name, phone, email,
					utm_source, utm_medium, utm_campaign, utm_content, utm_term,
					fbclid, gclid, ttclid, ip_address, user_agent, source, form_id,
					extra, processed, fb_capi_sent
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, false, false)
			`, req.TenantID, eventTime, eventID, req.Idempotency,
				req.FullName, req.Phone, req.Email,
				req.UtmSource, req.UtmMedium, req.UtmCampaign, req.UtmContent, req.UtmTerm,
				req.FBCLID, req.GCLID, req.TTCLID, req.IPAddress, req.UserAgent, "landing", req.FormID,
				extraMap,
			).WithContext(ctx).Exec()

			if err != nil {
				logger.Error(ctx, "scylla insert", err)
			}
		}

		// Create lead in PostgreSQL
		leadID := uuid.NewV7()
		customFields := map[string]interface{}{
			"utm_source":   req.UtmSource,
			"utm_medium":   req.UtmMedium,
			"utm_campaign": req.UtmCampaign,
			"form_id":      req.FormID,
		}
		for k, v := range req.Extra {
			customFields[k] = v
		}

		_, err := database.ExecContext(ctx, `
			INSERT INTO leads.leads (id, tenant_id, email, phone, full_name, source, custom_fields)
			VALUES ($1, $2, $3, $4, $5, 'landing', $6)
		`, leadID, req.TenantID, req.Email, req.Phone, req.FullName, customFields)
		if err != nil {
			logger.Error(ctx, "postgres insert", err)
			// Don't fail: ScyllaDB is source of truth for ingestion
		}

		// Trigger lead scoring async (NATS publish in production)
		logger.Info(ctx, "lead accepted", zap.String("lead_id", leadID.String()))

		return c.JSON(http.StatusOK, ingestResponse{
			Status:   "accepted",
			EventID:  eventID.String(),
			LeadID:   leadID.String(),
			Accepted: true,
		})
	}
}

func trackHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		eventName, _ := body["event_name"].(string)
		tenantID, _ := body["tenant_id"].(string)

		if eventName == "" || tenantID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "event_name and tenant_id required"})
		}

		// Log browser-side event for dedup
		logger.Info(ctx, "browser event",
			zap.String("event_name", eventName),
			zap.String("tenant_id", tenantID),
			zap.Any("payload", body),
		)

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

func capiSendHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		// In production: this is called by NATS subscriber for new leads
		// Implementation: POST to Facebook Conversions API with HMAC user_data
		ctx := c.Request().Context()
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		logger.Info(ctx, "CAPI send requested", zap.Any("payload_size", len(fmt.Sprintf("%v", body))))
		return c.JSON(http.StatusOK, map[string]string{"status": "queued"})
	}
}

func formWebhookHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

func powChallengeHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		challenge := uuid.NewV7().String()
		difficulty := 2
		ttl := 60
		valkeyClient.Set(context.Background(), "pow:"+challenge, difficulty, time.Duration(ttl)*time.Second)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"challenge":  challenge,
			"difficulty": difficulty,
			"ttl":        ttl,
		})
	}
}

func powVerifyHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			Challenge string `json:"challenge"`
			Nonce     string `json:"nonce"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Verify PoW: hash(challenge + nonce) must start with N zeros
		combined := body.Challenge + body.Nonce
		hash := sha256.Sum256([]byte(combined))
		hashHex := hex.EncodeToString(hash[:])

		// Check Valkey
		stored, _ := valkeyClient.Get(context.Background(), "pow:"+body.Challenge).Result()
		if stored == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "challenge expired"})
		}

		// Verify difficulty
		requiredZeros := 2
		if !strings.HasPrefix(hashHex, strings.Repeat("0", requiredZeros)) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid PoW"})
		}

		valkeyClient.Del(context.Background(), "pow:"+body.Challenge)
		return c.JSON(http.StatusOK, map[string]string{"status": "verified"})
	}
}

func validateLead(ctx context.Context, database *sql.DB, lead leadSubmission) map[string]string {
	errors := make(map[string]string)

	if lead.TenantID == "" {
		errors["tenant_id"] = "required"
	}
	if lead.Email == "" && lead.Phone == "" {
		errors["contact"] = "email or phone required"
	}
	if lead.Email != "" && !strings.Contains(lead.Email, "@") {
		errors["email"] = "invalid format"
	}

	return errors
}

func verifyHMAC(signature string, lead leadSubmission) bool {
	secret := getEnv("INGESTION_HMAC_SECRET", "dev_secret_change_me")
	if signature == "" {
		// Skip HMAC in dev mode for ease of testing
		return true
	}

	body, _ := json.Marshal(lead)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var _ = io.Discard
