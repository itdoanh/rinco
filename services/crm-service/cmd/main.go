// CRM Service - handles CRM tree (employee hierarchy) + leads.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
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
	serviceName = "crm-service"
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
	e.HidePort = true

	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	e.GET("/health", healthHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Protected routes
	crm := e.Group("/v1/crm")
	crm.Use(authMiddleware(database))
	crm.POST("/users", createUserHandler(database))
	crm.GET("/users/:id", getUserHandler(database))
	crm.GET("/users/:id/subtree", getSubtreeHandler(database))
	crm.POST("/users/:id/promote", promoteUserHandler(database))
	crm.POST("/users/:id/demote", demoteUserHandler(database))
	crm.POST("/users/:id/move", moveUserHandler(database))
	crm.GET("/users/:id/path", getUserPathHandler(database))

	crm.POST("/leads", createLeadHandler(database))
	crm.GET("/leads", listLeadsHandler(database))
	crm.GET("/leads/:id", getLeadHandler(database))
	crm.PATCH("/leads/:id", updateLeadHandler(database))

	port := ":" + getEnv("PORT", "8083")
	logger.Info(context.Background(), "starting crm service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

// ============ Types ============

type createUserRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	FullName    string `json:"full_name"`
	ParentID    string `json:"parent_id"`
	Roles       []string `json:"roles"`
}

type userResponse struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	FullName   string    `json:"full_name"`
	Path       string    `json:"path"`
	Roles      []string  `json:"roles"`
	CreatedAt  time.Time `json:"created_at"`
}

type createLeadRequest struct {
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	FullName   string                 `json:"full_name"`
	Source     string                 `json:"source"`
	CampaignID string                 `json:"campaign_id"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

type leadResponse struct {
	ID          string    `json:"id"`
	OwnerUserID *string   `json:"owner_user_id"`
	Email       *string   `json:"email"`
	Phone       *string   `json:"phone"`
	FullName    *string   `json:"full_name"`
	Score       float64   `json:"score"`
	ScoreBand   string    `json:"score_band"`
	Stage       string    `json:"stage"`
	Source      *string   `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type moveUserRequest struct {
	NewParentID string `json:"new_parent_id"`
}

type promoteDemoteRequest struct {
	NewRoles []string `json:"new_roles"`
}

// ============ Handlers ============

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
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant or user header"})
			}

			// Set RLS context for downstream queries
			ctx = db.SetTenantContext(ctx, tenantID, userID, isAdmin)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func createUserHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req createUserRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		tenantID := logger.TenantIDFromContext(ctx)
		if tenantID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "no tenant"})
		}

		// Compute path based on parent
		var parentPath string
		if req.ParentID != "" {
			err := database.QueryRowContext(ctx,
				`SELECT path::text FROM auth.users WHERE id = $1 AND tenant_id = $2`,
				req.ParentID, tenantID).Scan(&parentPath)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "parent not found"})
			}
		} else {
			parentPath = "root"
		}

		// Generate new UUID for path component
		newID := uuid.NewV7()
		newPath := fmt.Sprintf("%s.%s", parentPath, sanitizePathComponent(newID.String()))

		roles := req.Roles
		if len(roles) == 0 {
			roles = []string{"member"}
		}

		var createdAt time.Time
		err := database.QueryRowContext(ctx, `
			INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, path, roles)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING created_at
		`, newID, tenantID, req.Email, "$argon2id$v=19$m=65536,t=3,p=2$abcdef$1234567890abcdef", req.FullName, newPath, pqArray(roles)).Scan(&createdAt)

		if err != nil {
			logger.Error(ctx, "create user failed", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, userResponse{
			ID:        newID.String(),
			Email:     req.Email,
			FullName:  req.FullName,
			Path:      newPath,
			Roles:     roles,
			CreatedAt: createdAt,
		})
	}
}

func getUserHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var u userResponse
		err := database.QueryRowContext(ctx, `
			SELECT id::text, email, COALESCE(full_name, ''), path::text, roles, created_at
			FROM auth.users WHERE id = $1
		`, id).Scan(&u.ID, &u.Email, &u.FullName, &u.Path, pqArrayPtr(&u.Roles), &u.CreatedAt)

		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, u)
	}
}

func getSubtreeHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		// Get user's path first
		var path string
		err := database.QueryRowContext(ctx,
			`SELECT path::text FROM auth.users WHERE id = $1`, id).Scan(&path)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}

		// Find all descendants
		rows, err := database.QueryContext(ctx, `
			SELECT id::text, email, COALESCE(full_name, ''), path::text, roles, created_at
			FROM auth.users
			WHERE path <@ CAST($1 AS ltree)
			ORDER BY path
		`, path)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var users []userResponse
		for rows.Next() {
			var u userResponse
			var roles []byte
			if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Path, &roles, &u.CreatedAt); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			users = append(users, u)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"root_id": id,
			"path":    path,
			"count":   len(users),
			"users":   users,
		})
	}
}

func promoteUserHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var req promoteDemoteRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		_, err := database.ExecContext(ctx,
			`UPDATE auth.users SET roles = $1 WHERE id = $2`,
			pqArray(req.NewRoles), id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "promoted"})
	}
}

func demoteUserHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var req promoteDemoteRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		_, err := database.ExecContext(ctx,
			`UPDATE auth.users SET roles = $1 WHERE id = $2`,
			pqArray(req.NewRoles), id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "demoted"})
	}
}

func moveUserHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var req moveUserRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Get new parent's path
		var newParentPath string
		err := database.QueryRowContext(ctx,
			`SELECT path::text FROM auth.users WHERE id = $1`, req.NewParentID).Scan(&newParentPath)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "new parent not found"})
		}

		// Move user + descendants atomically
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer tx.Rollback()

		// Get user's old path
		var oldPath string
		err = tx.QueryRowContext(ctx,
			`SELECT path::text FROM auth.users WHERE id = $1 FOR UPDATE`, id).Scan(&oldPath)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}

		// Get last component of old path (the user's own component)
		parts := strings.Split(oldPath, ".")
		ownComponent := parts[len(parts)-1]

		// Compute new path for user
		newUserPath := fmt.Sprintf("%s.%s", newParentPath, ownComponent)

		// Update user
		_, err = tx.ExecContext(ctx,
			`UPDATE auth.users SET path = CAST($1 AS ltree) WHERE id = $2`,
			newUserPath, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Update all descendants (npath starts with oldPath)
		_, err = tx.ExecContext(ctx, `
			UPDATE auth.users
			SET path = CAST($1 AS ltree) || subpath(path, nlevel(CAST($2 AS ltree)))
			WHERE path <@ CAST($2 AS ltree) AND id != $3
		`, newParentPath, oldPath, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status":    "moved",
			"new_path":  newUserPath,
		})
	}
}

func getUserPathHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var path string
		err := database.QueryRowContext(ctx,
			`SELECT path::text FROM auth.users WHERE id = $1`, id).Scan(&path)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}

		// Get ancestors
		rows, err := database.QueryContext(ctx, `
			SELECT id::text, COALESCE(full_name, email), nlevel(path) as depth
			FROM auth.users
			WHERE CAST($1 AS ltree) @> path
			ORDER BY nlevel(path)
		`, path)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var ancestors []map[string]interface{}
		for rows.Next() {
			var aid, name string
			var depth int
			if err := rows.Scan(&aid, &name, &depth); err != nil {
				continue
			}
			ancestors = append(ancestors, map[string]interface{}{
				"id":    aid,
				"name":  name,
				"depth": depth,
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"user_id":   id,
			"path":      path,
			"ancestors": ancestors,
		})
	}
}

func createLeadHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req createLeadRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		tenantID := logger.TenantIDFromContext(ctx)
		userID := logger.UserIDFromContext(ctx)

		newID := uuid.NewV7()
		var l leadResponse
		lid := newID.String()

		err := database.QueryRowContext(ctx, `
			INSERT INTO leads.leads (id, tenant_id, owner_user_id, email, phone, full_name, source, campaign_id, custom_fields)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id::text, owner_user_id::text, email, phone, full_name, score, score_band, stage, source, created_at, updated_at
		`, newID, tenantID, userID, req.Email, req.Phone, req.FullName, req.Source, req.CampaignID, req.CustomFields).
			Scan(&l.ID, &l.OwnerUserID, &l.Email, &l.Phone, &l.FullName, &l.Score, &l.ScoreBand, &l.Stage, &l.Source, &l.CreatedAt, &l.UpdatedAt)

		_ = lid
		if err != nil {
			logger.Error(ctx, "create lead failed", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, l)
	}
}

func listLeadsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		stage := c.QueryParam("stage")
		minScore := c.QueryParam("min_score")

		query := `SELECT id::text, owner_user_id::text, email, phone, full_name, score, score_band, stage, source, created_at, updated_at FROM leads.leads WHERE 1=1`
		args := []interface{}{}
		idx := 1

		if stage != "" {
			query += fmt.Sprintf(" AND stage = $%d", idx)
			args = append(args, stage)
			idx++
		}
		if minScore != "" {
			query += fmt.Sprintf(" AND score >= $%d", idx)
			args = append(args, minScore)
			idx++
		}

		query += " ORDER BY created_at DESC LIMIT 100"

		rows, err := database.QueryContext(ctx, query, args...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var leads []leadResponse
		for rows.Next() {
			var l leadResponse
			if err := rows.Scan(&l.ID, &l.OwnerUserID, &l.Email, &l.Phone, &l.FullName, &l.Score, &l.ScoreBand, &l.Stage, &l.Source, &l.CreatedAt, &l.UpdatedAt); err != nil {
				continue
			}
			leads = append(leads, l)
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

		var l leadResponse
		err := database.QueryRowContext(ctx, `
			SELECT id::text, owner_user_id::text, email, phone, full_name, score, score_band, stage, source, created_at, updated_at
			FROM leads.leads WHERE id = $1
		`, id).Scan(&l.ID, &l.OwnerUserID, &l.Email, &l.Phone, &l.FullName, &l.Score, &l.ScoreBand, &l.Stage, &l.Source, &l.CreatedAt, &l.UpdatedAt)

		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "lead not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, l)
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

		// Whitelist allowed fields
		allowed := []string{"stage", "score", "owner_user_id", "custom_fields"}
		sets := []string{}
		args := []interface{}{}
		idx := 1
		for _, f := range allowed {
			if v, ok := body[f]; ok {
				sets = append(sets, fmt.Sprintf("%s = $%d", f, idx))
				args = append(args, v)
				idx++
			}
		}

		if len(sets) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no fields to update"})
		}

		args = append(args, id)
		query := fmt.Sprintf("UPDATE leads.leads SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
		_, err := database.ExecContext(ctx, query, args...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

// ============ Helpers ============

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func sanitizePathComponent(s string) string {
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}

// pqArray wraps a string slice as a Postgres array literal.
// Using pgx-compatible format.
func pqArray(s []string) interface{} {
	if len(s) == 0 {
		return "{}"
	}
	// pgx supports Go []string directly via the simple protocol
	return s
}

func pqArrayPtr(s *[]string) interface{} {
	if s == nil {
		return []string{}
	}
	return *s
}
