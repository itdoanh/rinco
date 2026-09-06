// Tenant Service - tenant + tenant site management.
package main

import (
	"context"
	"database/sql"
	"fmt"
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
	serviceName = "tenant-service"
	version     = "1.0.0"
)

type createTenantRequest struct {
	Slug           string `json:"slug" validate:"required"`
	Name           string `json:"name" validate:"required"`
	Plan           string `json:"plan"`
	AdminEmail     string `json:"admin_email" validate:"required,email"`
	AdminPassword  string `json:"admin_password" validate:"required,min=8"`
	AdminFullName  string `json:"admin_full_name"`
}

type tenantResponse struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

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

	// Public endpoints
	e.POST("/v1/tenants", createTenantHandler(database))
	e.GET("/v1/tenants/by-slug/:slug", getTenantBySlugHandler(database))

	// Admin endpoints
	admin := e.Group("/v1/admin")
	admin.Use(adminAuthMiddleware())
	admin.GET("/tenants", listTenantsHandler(database))
	admin.GET("/tenants/:id", getTenantHandler(database))
	admin.PATCH("/tenants/:id", updateTenantHandler(database))
	admin.POST("/tenants/:id/suspend", suspendTenantHandler(database))
	admin.POST("/tenants/:id/activate", activateTenantHandler(database))

	// Tenant site
	e.POST("/v1/tenant-sites", createTenantSiteHandler(database))
	e.GET("/v1/tenant-sites/by-domain/:domain", getTenantSiteByDomainHandler(database))

	port := ":" + getEnv("PORT", "8082")
	logger.Info(context.Background(), "starting tenant service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func createTenantHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req createTenantRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Bypass RLS for tenant creation (super-admin operation)
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer tx.Rollback()

		tenantID := uuid.NewV7()
		_, err = tx.ExecContext(ctx,
			`INSERT INTO tenant.tenants (id, slug, name, plan, status) VALUES ($1, $2, $3, $4, 'trial')`,
			tenantID, req.Slug, req.Name, req.Plan)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant already exists"})
		}

		// Create root user
		userID := uuid.NewV7()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, path, roles)
			VALUES ($1, $2, $3, $4, $5, 'root', ARRAY['admin'])
		`, userID, tenantID, req.AdminEmail, "$argon2id$v=19$m=65536,t=3,p=2$placeholder$placeholder", req.AdminFullName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, tenantResponse{
			ID:     tenantID.String(),
			Slug:   req.Slug,
			Name:   req.Name,
			Plan:   req.Plan,
			Status: "trial",
		})
	}
}

func getTenantBySlugHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		slug := c.Param("slug")
		var t tenantResponse
		err := database.QueryRow(`
			SELECT id::text, slug, name, COALESCE(plan, 'starter'), status::text, created_at
			FROM tenant.tenants WHERE slug = $1
		`, slug).Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status, &t.CreatedAt)

		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, t)
	}
}

func adminAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			adminKey := c.Request().Header.Get("X-Admin-Key")
			expected := os.Getenv("ADMIN_API_KEY")
			if expected == "" {
				expected = "dev_admin_key_change_me"
			}
			if adminKey != expected {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(c)
		}
	}
}

func listTenantsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := database.Query(`
			SELECT id::text, slug, name, COALESCE(plan, 'starter'), status::text, created_at
			FROM tenant.tenants
			WHERE deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 100
		`)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var tenants []tenantResponse
		for rows.Next() {
			var t tenantResponse
			if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status, &t.CreatedAt); err != nil {
				continue
			}
			tenants = append(tenants, t)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"count":   len(tenants),
			"tenants": tenants,
		})
	}
}

func getTenantHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		var t tenantResponse
		err := database.QueryRow(`
			SELECT id::text, slug, name, COALESCE(plan, 'starter'), status::text, created_at
			FROM tenant.tenants WHERE id = $1
		`, id).Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status, &t.CreatedAt)

		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, t)
	}
}

func updateTenantHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		_, err := database.Exec(`
			UPDATE tenant.tenants SET name = COALESCE($1, name), plan = COALESCE($2, plan)
			WHERE id = $3
		`, body["name"], body["plan"], id)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

func suspendTenantHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		_, err := database.Exec(`UPDATE tenant.tenants SET status = 'suspended' WHERE id = $1`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "suspended"})
	}
}

func activateTenantHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		_, err := database.Exec(`UPDATE tenant.tenants SET status = 'active' WHERE id = $1`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "activated"})
	}
}

func createTenantSiteHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		id := uuid.NewV7()
		tenantID, _ := body["tenant_id"].(string)
		domain, _ := body["domain"].(string)

		if tenantID == "" || domain == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id and domain required"})
		}

		_, err := database.Exec(`
			INSERT INTO tenant.tenant_sites (id, tenant_id, domain, status)
			VALUES ($1, $2, $3, 'active')
		`, id, tenantID, domain)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"id":        id,
			"tenant_id": tenantID,
			"domain":    domain,
		})
	}
}

func getTenantSiteByDomainHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		domain := c.Param("domain")
		var id, tenantID, status string
		var theme, branding map[string]interface{}
		err := database.QueryRow(`
			SELECT id::text, tenant_id::text, status, COALESCE(theme, '{}'), COALESCE(branding, '{}')
			FROM tenant.tenant_sites WHERE domain = $1
		`, domain).Scan(&id, &tenantID, &status, &theme, &branding)

		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":        id,
			"tenant_id": tenantID,
			"domain":    domain,
			"status":    status,
			"theme":     theme,
			"branding":  branding,
		})
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var _ = fmt.Sprintf
