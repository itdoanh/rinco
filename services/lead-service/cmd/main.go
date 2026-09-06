// Lead Service - facade orchestrating lead ingestion, scoring, enrichment.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/db"
	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "lead-service"
	version     = "1.0.0"
)

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

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
		logger.Fatal(context.Background(), "db connect failed", err)
	}
	defer database.Close()

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

	lead := e.Group("/v1/leads")
	lead.Use(authMiddleware(database))
	lead.POST("", createLeadHandler(database))
	lead.GET("", listLeadsHandler(database))
	lead.GET("/:id", getLeadHandler(database))
	lead.PATCH("/:id", updateLeadHandler(database))
	lead.POST("/:id/score", scoreLeadHandler(database))

	port := ":" + getEnv("PORT", "8085")
	logger.Info(context.Background(), "starting lead service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func authMiddleware(database *sql.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			tenantID := c.Request().Header.Get("X-Tenant-ID")
			userID := c.Request().Header.Get("X-User-ID")
			isAdmin := c.Request().Header.Get("X-Is-Super-Admin") == "true"

			if tenantID == "" || userID == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing headers"})
			}

			ctx = db.SetTenantContext(ctx, tenantID, userID, isAdmin)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func createLeadHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		tenantID := logger.TenantIDFromContext(ctx)
		userID := logger.UserIDFromContext(ctx)

		newID := uuid.NewV7()
		customFields := map[string]interface{}{}
		if v, ok := body["custom_fields"].(map[string]interface{}); ok {
			customFields = v
		}

		_, err := database.ExecContext(ctx, `
			INSERT INTO leads.leads (id, tenant_id, owner_user_id, email, phone, full_name, source, custom_fields)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, newID, tenantID, userID, body["email"], body["phone"], body["full_name"], body["source"], customFields)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Async: trigger lead scoring via NATS publish (in production)
		go triggerLeadScoring(newID.String(), body)

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"id":         newID,
			"status":     "created",
			"scoring":    "queued",
		})
	}
}

func listLeadsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		rows, err := database.QueryContext(ctx, `
			SELECT id::text, email, phone, full_name, score, score_band, stage, source, created_at
			FROM leads.leads
			ORDER BY created_at DESC LIMIT 100
		`)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var leads []map[string]interface{}
		for rows.Next() {
			var id string
			var email, phone, fullName, scoreBand, stage, source *string
			var score float64
			var createdAt time.Time
			if err := rows.Scan(&id, &email, &phone, &fullName, &score, &scoreBand, &stage, &source, &createdAt); err != nil {
				continue
			}
			leads = append(leads, map[string]interface{}{
				"id":         id,
				"email":      email,
				"phone":      phone,
				"full_name":  fullName,
				"score":      score,
				"score_band": scoreBand,
				"stage":      stage,
				"source":     source,
				"created_at": createdAt,
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"count": len(leads),
			"leads": leads,
		})
	}
}

func getLeadHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var lead map[string]interface{}
		var id2 string
		var email, phone, fullName, scoreBand, stage, source *string
		var score float64
		var customFields []byte
		var createdAt, updatedAt time.Time
		err := database.QueryRowContext(ctx, `
			SELECT id::text, email, phone, full_name, score, score_band, stage, source, custom_fields, created_at, updated_at
			FROM leads.leads WHERE id = $1
		`, id).Scan(&id2, &email, &phone, &fullName, &score, &scoreBand, &stage, &source, &customFields, &createdAt, &updatedAt)

		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		var cf map[string]interface{}
		_ = json.Unmarshal(customFields, &cf)

		lead = map[string]interface{}{
			"id":            id2,
			"email":         email,
			"phone":         phone,
			"full_name":     fullName,
			"score":         score,
			"score_band":    scoreBand,
			"stage":         stage,
			"source":        source,
			"custom_fields": cf,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		}
		return c.JSON(http.StatusOK, lead)
	}
}

func updateLeadHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		allowed := []string{"stage", "score", "custom_fields"}
		sets := []string{}
		args := []interface{}{}
		idx := 1
		for _, f := range allowed {
			if v, ok := body[f]; ok {
				sets = append(sets, f+" = $"+itoa(idx))
				args = append(args, v)
				idx++
			}
		}
		args = append(args, id)
		query := "UPDATE leads.leads SET " + joinStrings(sets, ", ") + " WHERE id = $" + itoa(idx)
		_, err := database.ExecContext(ctx, query, args...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

func scoreLeadHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		// Fetch lead
		var customFields []byte
		var email, phone, fullName, source *string
		err := database.QueryRowContext(ctx,
			`SELECT email, phone, full_name, source, custom_fields FROM leads.leads WHERE id = $1`,
			id).Scan(&email, &phone, &fullName, &source, &customFields)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "lead not found"})
		}

		var cf map[string]interface{}
		_ = json.Unmarshal(customFields, &cf)

		// Call lead-scoring service
		scoreResult, err := callLeadScoring(ctx, email, phone, fullName, source, cf)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Update lead with new score
		_, _ = database.ExecContext(ctx, `UPDATE leads.leads SET score = $1 WHERE id = $2`, scoreResult.Score, id)

		return c.JSON(http.StatusOK, scoreResult)
	}
}

type ScoreResult struct {
	Score       float64                `json:"score"`
	ScoreBand   string                 `json:"score_band"`
	Explanation string                 `json:"explanation"`
}

func callLeadScoring(ctx context.Context, email, phone, fullName, source *string, customFields map[string]interface{}) (*ScoreResult, error) {
	// In production: HTTP call to lead-scoring service
	// Here: simple rule-based fallback
	score := 0.5
	if email != nil && *email != "" {
		score += 0.1
	}
	if phone != nil && *phone != "" {
		score += 0.2
	}
	if source != nil && *source != "" {
		score += 0.1
	}
	if v, ok := customFields["page_views"].(float64); ok && v > 0 {
		score += 0.1
	}

	band := "cold"
	if score >= 0.8 {
		band = "hot"
	} else if score >= 0.5 {
		band = "warm"
	}

	return &ScoreResult{Score: score, ScoreBand: band, Explanation: "rule-based fallback"}, nil
}

func triggerLeadScoring(leadID string, body map[string]interface{}) {
	logger.Info(context.Background(), "lead scoring triggered", zap.String("lead_id", leadID))
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
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
