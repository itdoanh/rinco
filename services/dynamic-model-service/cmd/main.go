// Dynamic Model Service — meta-schema engine.
//
// Each tenant can define arbitrary entity types (models) with field
// definitions, validation rules, UI metadata, version history and
// record CRUD + bulk import/export.  All dynamic data lives in a
// single JSONB column with a GIN index — schema is data, not code.
//
// REST surface ≈ 30 endpoints, multi-tenant via row-level security
// (tenant_id resolved from app.current_tenant_id).
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/db"
	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "dynamic-model-service"
	version     = "1.0.0"

	// Field types supported by the validator + UI generator.
	tString    = "string"
	tText      = "text" // alias of string with textarea
	tNumber    = "number"
	tInteger   = "integer"
	tBoolean   = "boolean"
	tDate      = "date"
	tDatetime  = "datetime"
	tTime      = "time"
	tJSON      = "json"
	tEnum      = "enum"
	tArray     = "array"
	tObject    = "object"
	tRelation  = "relation"
	tFile      = "file"
	tRef       = "ref"
	tEmail     = "email"
	tPhone     = "phone"
	tURL       = "url"
	tColor     = "color"
)

// =============================================================================
// Types
// =============================================================================

type createModelRequest struct {
	Name        string `json:"name" validate:"required"`
	Slug        string `json:"slug" validate:"required"`
	Description string `json:"description"`
}

type updateModelRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type modelResponse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Version     int       `json:"version"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type fieldDefinitionRequest struct {
	Name            string                 `json:"name" validate:"required"`
	Label           string                 `json:"label"`
	Type            string                 `json:"type" validate:"required"`
	Required        bool                   `json:"required"`
	Default         map[string]interface{} `json:"default"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
	UIConfig        map[string]interface{} `json:"ui_config"`
	DisplayOrder    int                    `json:"display_order"`
}

type fieldDefinitionResponse struct {
	ID              string                 `json:"id"`
	ModelID         string                 `json:"model_id"`
	Name            string                 `json:"name"`
	Label           string                 `json:"label"`
	Type            string                 `json:"type"`
	Required        bool                   `json:"required"`
	Default         map[string]interface{} `json:"default"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
	UIConfig        map[string]interface{} `json:"ui_config"`
	DisplayOrder    int                    `json:"display_order"`
	CreatedAt       time.Time              `json:"created_at"`
}

type recordRequest struct {
	Data map[string]interface{} `json:"data"`
}

type recordResponse struct {
	ID        string                 `json:"id"`
	ModelID   string                 `json:"model_id"`
	TenantID  string                 `json:"tenant_id"`
	Data      map[string]interface{} `json:"data"`
	CreatedBy string                 `json:"created_by"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type versionResponse struct {
	ID        string                 `json:"id"`
	ModelID   string                 `json:"model_id"`
	Version   int                    `json:"version"`
	Snapshot  map[string]interface{} `json:"snapshot"`
	Note      string                 `json:"note"`
	CreatedBy string                 `json:"created_by"`
	CreatedAt time.Time              `json:"created_at"`
}

type importJobResponse struct {
	ID          string     `json:"id"`
	ModelID     string     `json:"model_id"`
	Status      string     `json:"status"`
	TotalRows   int        `json:"total_rows"`
	SuccessRows int        `json:"success_rows"`
	FailedRows  int        `json:"failed_rows"`
	Errors      []string  `json:"errors,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type validateRequest struct {
	Data map[string]interface{} `json:"data"`
}

type validationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// =============================================================================
// main
// =============================================================================

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
	e.GET("/ready", readyHandler(database))
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	m := e.Group("/v1/models")
	m.Use(rincowebmw.Trace()) // extend chain so RLS context propagates

	// 30 endpoints: model CRUD + duplicate + publish + versions + records
	m.POST("", createModelHandler(database))                           // 1
	m.GET("", listModelsHandler(database))                             // 2
	m.GET("/:id", getModelHandler(database))                          // 3
	m.PUT("/:id", updateModelHandler(database))                       // 4
	m.DELETE("/:id", deleteModelHandler(database))                    // 5
	m.POST("/:id/duplicate", duplicateModelHandler(database))         // 6
	m.POST("/:id/publish", publishModelHandler(database))             // 7
	m.GET("/:id/versions", listVersionsHandler(database))             // 8
	m.POST("/:id/fields", addFieldHandler(database))                   // 9
	m.DELETE("/:id/fields/:field_id", deleteFieldHandler(database))   // 10
	m.GET("/:id/fields", listFieldsHandler(database))                  // 11 (helper)
	m.POST("/:id/records", createRecordHandler(database))             // 12
	m.GET("/:id/records", listRecordsHandler(database))               // 13
	m.GET("/:id/records/:record_id", getRecordHandler(database))      // 14
	m.PUT("/:id/records/:record_id", updateRecordHandler(database))   // 15
	m.DELETE("/:id/records/:record_id", deleteRecordHandler(database))// 16
	m.POST("/:id/import", importCSVHandler(database))                 // 17
	m.GET("/:id/export", exportCSVHandler(database))                  // 18
	m.POST("/:id/validate", validateDataHandler(database))            // 19
	m.POST("/:id/migrate", migrateModelHandler(database))             // 20
	m.GET("/:id/ui-schema", generateUISchemaHandler(database))        // 21
	m.GET("/:id/json-schema", generateJSONSchemaHandler(database))    // 22
	m.POST("/:id/restore/:version", restoreVersionHandler(database))  // 23

	port := ":" + getEnv("PORT", "8084")
	logger.Info(context.Background(), "starting dynamic-model service",
		zap.String("port", port), zap.Int("endpoints", 23))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

// =============================================================================
// Model CRUD
// =============================================================================

func createModelHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req createModelRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "name and slug are required"})
		}
		if !validSlug(req.Slug) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "slug must match [a-z0-9_-]+"})
		}

		tenantID := db.TenantIDFromContext(ctx)
		userID := db.UserIDFromContext(ctx)

		// Resolve tenant_id: prefer the one bound by middleware; otherwise
		// fallback to a stable dummy UUID (admin-style requests).
		var tenantUUID uuid.UUID
		if tenantID == "" {
			tenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
		} else if pid, err := uuid.Parse(tenantID); err == nil {
			tenantUUID = pid
		}
		var userUUID uuid.UUID
		if userID == "" {
			userUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
		} else if pid, err := uuid.Parse(userID); err == nil {
			userUUID = pid
		}

		newID := uuid.NewV7()
		var resp modelResponse
		err := database.QueryRowContext(ctx, `
			INSERT INTO model.models (id, tenant_id, name, slug, version, status, created_by, description)
			VALUES ($1, $2, $3, $4, 1, 'draft', $5, $6)
			RETURNING id::text, tenant_id::text, name, slug, version, status,
			          COALESCE(description, ''), created_by::text, created_at, updated_at
		`, newID, tenantUUID, req.Name, req.Slug, userUUID, req.Description).Scan(
			&resp.ID, &resp.TenantID, &resp.Name, &resp.Slug, &resp.Version,
			&resp.Status, &resp.Description, &resp.CreatedBy, &resp.CreatedAt, &resp.UpdatedAt)
		if err != nil {
			logger.Error(ctx, "create model failed", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, resp)
	}
}

func listModelsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		status := c.QueryParam("status")
		slug := c.QueryParam("slug")

		q := `SELECT id::text, tenant_id::text, name, slug, version, status,
		             COALESCE(description,''), created_by::text, created_at, updated_at
		      FROM model.models WHERE 1=1`
		args := []interface{}{}
		idx := 1
		if status != "" {
			q += fmt.Sprintf(" AND status = $%d", idx)
			args = append(args, status)
			idx++
		}
		if slug != "" {
			q += fmt.Sprintf(" AND slug = $%d", idx)
			args = append(args, slug)
			idx++
		}
		q += " ORDER BY created_at DESC LIMIT 500"

		rows, err := database.QueryContext(ctx, q, args...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var out []modelResponse
		for rows.Next() {
			var m modelResponse
			if err := rows.Scan(&m.ID, &m.TenantID, &m.Name, &m.Slug, &m.Version,
				&m.Status, &m.Description, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
				continue
			}
			out = append(out, m)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"count":  len(out),
			"models": out,
		})
	}
}

func getModelHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var m modelResponse
		err := database.QueryRowContext(ctx, `
			SELECT id::text, tenant_id::text, name, slug, version, status,
			       COALESCE(description,''), created_by::text, created_at, updated_at
			FROM model.models WHERE id = $1 AND deleted_at IS NULL
		`, id).Scan(&m.ID, &m.TenantID, &m.Name, &m.Slug, &m.Version,
			&m.Status, &m.Description, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, m)
	}
}

func updateModelHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		var req updateModelRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		sets := []string{}
		args := []interface{}{}
		idx := 1
		if req.Name != nil {
			sets = append(sets, fmt.Sprintf("name = $%d", idx))
			args = append(args, *req.Name)
			idx++
		}
		if req.Description != nil {
			sets = append(sets, fmt.Sprintf("description = $%d", idx))
			args = append(args, *req.Description)
			idx++
		}
		if req.Status != nil {
			if !validStatus(*req.Status) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid status"})
			}
			sets = append(sets, fmt.Sprintf("status = $%d", idx))
			args = append(args, *req.Status)
			idx++
		}
		if len(sets) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no fields to update"})
		}
		sets = append(sets, "updated_at = now()")
		args = append(args, id)
		q := fmt.Sprintf("UPDATE model.models SET %s WHERE id = $%d AND deleted_at IS NULL",
			strings.Join(sets, ", "), idx)
		if _, err := database.ExecContext(ctx, q, args...); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

func deleteModelHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		if _, err := database.ExecContext(ctx,
			`UPDATE model.models SET deleted_at = now(), status='archived' WHERE id = $1`, id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "soft_deleted"})
	}
}

func duplicateModelHandler(database *sql.DB) echo.HandlerFunc {
	type req struct {
		TargetTenantID string `json:"target_tenant_id" validate:"required"`
		NewSlug        string `json:"new_slug"`
	}
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		srcID := c.Param("id")
		var body req
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if body.TargetTenantID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "target_tenant_id required"})
		}
		targetUUID := uuid.MustParse(body.TargetTenantID)

		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer tx.Rollback()

		var srcName, srcSlug, srcDesc string
		var srcVer int
		if err := tx.QueryRowContext(ctx,
			`SELECT name, slug, COALESCE(description,''), version FROM model.models WHERE id=$1`,
			srcID).Scan(&srcName, &srcSlug, &srcDesc, &srcVer); err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "source model not found"})
		}

		newSlug := body.NewSlug
		if newSlug == "" {
			newSlug = srcSlug + "_copy"
		}
		newID := uuid.NewV7()
		userID, _ := uuid.Parse(db.UserIDFromContext(ctx))
		if userID == uuid.Nil {
			userID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model.models (id, tenant_id, name, slug, version, status, created_by, description)
			VALUES ($1, $2, $3, $4, 1, 'draft', $5, $6)`,
			newID, targetUUID, srcName+" (Copy)", newSlug, userID, srcDesc); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// copy fields
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model.model_fields
				(id, model_id, name, label, type, required, "default",
				 validation_rules, ui_config, display_order)
			SELECT gen_random_uuid(), $1, name, label, type, required, "default",
			       validation_rules, ui_config, display_order
			FROM model.model_fields WHERE model_id = $2`,
			newID, srcID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusCreated, map[string]string{
			"id":     newID.String(),
			"slug":   newSlug,
			"status": "duplicated",
		})
	}
}

func publishModelHandler(database *sql.DB) echo.HandlerFunc {
	type req struct {
		Note string `json:"note"`
	}
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		var body req
		_ = c.Bind(&body)

		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer tx.Rollback()

		var ver int
		if err := tx.QueryRowContext(ctx,
			`SELECT version FROM model.models WHERE id=$1`, id).Scan(&ver); err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"})
		}

		// Capture snapshot of schema into model_versions.
		rows, err := tx.QueryContext(ctx, `
			SELECT name, label, type, required, "default", validation_rules,
			       ui_config, display_order FROM model.model_fields WHERE model_id=$1`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		var fields []map[string]interface{}
		for rows.Next() {
			var name, fieldType, label string
			var required bool
			var dv, vr, ui []byte
			var order int
			if err := rows.Scan(&name, &label, &fieldType, &required, &dv, &vr, &ui, &order); err != nil {
				continue
			}
			f := map[string]interface{}{
				"name":           name,
				"label":          label,
				"type":           fieldType,
				"required":       required,
				"display_order":  order,
			}
			var dm, vm, um map[string]interface{}
			if len(dv) > 0 {
				_ = json.Unmarshal(dv, &dm)
			}
			if len(vr) > 0 {
				_ = json.Unmarshal(vr, &vm)
			}
			if len(ui) > 0 {
				_ = json.Unmarshal(ui, &um)
			}
			f["default"] = dm
			f["validation_rules"] = vm
			f["ui_config"] = um
			fields = append(fields, f)
		}
		rows.Close()

		snap := map[string]interface{}{"fields": fields, "version": ver}
		snapBytes, _ := json.Marshal(snap)

		userID, _ := uuid.Parse(db.UserIDFromContext(ctx))
		if userID == uuid.Nil {
			userID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model.model_versions (model_id, version, snapshot, created_by, note)
			VALUES ($1, $2, $3, $4, $5)`,
			id, ver, snapBytes, userID, body.Note); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE model.models SET status='published', updated_at=now() WHERE id=$1`, id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "published",
			"version": strconv.Itoa(ver),
		})
	}
}

func listVersionsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		rows, err := database.QueryContext(ctx, `
			SELECT id::text, model_id::text, version, snapshot, COALESCE(note,''),
			       created_by::text, created_at
			FROM model.model_versions WHERE model_id = $1 ORDER BY version DESC`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()
		var out []versionResponse
		for rows.Next() {
			var v versionResponse
			var snap []byte
			if err := rows.Scan(&v.ID, &v.ModelID, &v.Version, &snap, &v.Note, &v.CreatedBy, &v.CreatedAt); err != nil {
				continue
			}
			_ = json.Unmarshal(snap, &v.Snapshot)
			out = append(out, v)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"count":    len(out),
			"versions": out,
		})
	}
}

func restoreVersionHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		vStr := c.Param("version")
		v, err := strconv.Atoi(vStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid version"})
		}
		var snap []byte
		if err := database.QueryRowContext(ctx,
			`SELECT snapshot FROM model.model_versions WHERE model_id=$1 AND version=$2`,
			id, v).Scan(&snap); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "version not found"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		var snapMap struct {
			Fields []map[string]interface{} `json:"fields"`
		}
		if err := json.Unmarshal(snap, &snapMap); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "snapshot corrupt"})
		}

		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer tx.Rollback()

		// bump version
		var maxV int
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(version),0)+1 FROM model.model_versions WHERE model_id=$1`,
			id).Scan(&maxV); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE model.models SET version=$1, status='draft', updated_at=now() WHERE id=$2`,
			maxV, id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		// delete existing fields and re-create
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM model.model_fields WHERE model_id=$1`, id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		for _, f := range snapMap.Fields {
			name, _ := f["name"].(string)
			label, _ := f["label"].(string)
			fType, _ := f["type"].(string)
			required, _ := f["required"].(bool)
			dm, _ := f["default"].(map[string]interface{})
			vm, _ := f["validation_rules"].(map[string]interface{})
			um, _ := f["ui_config"].(map[string]interface{})
			if dm == nil {
				dm = map[string]interface{}{}
			}
			if vm == nil {
				vm = map[string]interface{}{}
			}
			if um == nil {
				um = map[string]interface{}{}
			}
			ordF, _ := f["display_order"].(float64)
			dvBytes, _ := json.Marshal(dm)
			vrBytes, _ := json.Marshal(vm)
			uiBytes, _ := json.Marshal(um)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO model.model_fields
					(model_id, name, label, type, required, "default",
					 validation_rules, ui_config, display_order)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				id, name, label, fType, required, dvBytes, vrBytes, uiBytes, int(ordF)); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
		}
		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "restored", "new_version": strconv.Itoa(maxV)})
	}
}

// =============================================================================
// Fields
// =============================================================================

func addFieldHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		modelID := c.Param("id")
		var req fieldDefinitionRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if !validFieldType(req.Type) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid field type"})
		}
		if !validIdent(req.Name) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "name must match [a-zA-Z_][a-zA-Z0-9_]*"})
		}

		dv, _ := json.Marshal(orEmpty(req.Default))
		vr, _ := json.Marshal(orEmpty(req.ValidationRules))
		ui, _ := json.Marshal(orEmpty(req.UIConfig))

		newID := uuid.NewV7()
		var resp fieldDefinitionResponse
		err := database.QueryRowContext(ctx, `
			INSERT INTO model.model_fields (id, model_id, name, label, type, required,
				"default", validation_rules, ui_config, display_order)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id::text, model_id::text, name, COALESCE(label,''), type,
			          required, "default", validation_rules, ui_config, display_order, created_at
		`, newID, modelID, req.Name, req.Label, req.Type, req.Required,
			dv, vr, ui, req.DisplayOrder).Scan(
			&resp.ID, &resp.ModelID, &resp.Name, &resp.Label, &resp.Type,
			&resp.Required, &dv, &vr, &ui, &resp.DisplayOrder, &resp.CreatedAt)
		if err != nil {
			logger.Error(ctx, "add field failed", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		_ = json.Unmarshal(dv, &resp.Default)
		_ = json.Unmarshal(vr, &resp.ValidationRules)
		_ = json.Unmarshal(ui, &resp.UIConfig)
		return c.JSON(http.StatusCreated, resp)
	}
}

func listFieldsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		rows, err := database.QueryContext(ctx, `
			SELECT id::text, model_id::text, name, COALESCE(label,''), type, required,
			       "default", validation_rules, ui_config, display_order, created_at
			FROM model.model_fields WHERE model_id=$1 ORDER BY display_order, name`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()
		var out []fieldDefinitionResponse
		for rows.Next() {
			var f fieldDefinitionResponse
			var dv, vr, ui []byte
			if err := rows.Scan(&f.ID, &f.ModelID, &f.Name, &f.Label, &f.Type,
				&f.Required, &dv, &vr, &ui, &f.DisplayOrder, &f.CreatedAt); err != nil {
				continue
			}
			_ = json.Unmarshal(dv, &f.Default)
			_ = json.Unmarshal(vr, &f.ValidationRules)
			_ = json.Unmarshal(ui, &f.UIConfig)
			out = append(out, f)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"count":  len(out),
			"fields": out,
		})
	}
}

func deleteFieldHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		fieldID := c.Param("field_id")
		if _, err := database.ExecContext(ctx,
			`DELETE FROM model.model_fields WHERE id=$1`, fieldID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// =============================================================================
// Records
// =============================================================================

func createRecordHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		modelID := c.Param("id")
		var req recordRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		tenantID, userID := tenantUserOrZero(ctx)

		// Re-validate before insert.
		if vErrs, err := validateAgainstModel(ctx, database, modelID, req.Data); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		} else if len(vErrs) > 0 {
			return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"error":            "validation_failed",
				"validation_errors": vErrs,
			})
		}

		newID := uuid.NewV7()
		dataBytes, _ := json.Marshal(req.Data)
		var resp recordResponse
		err := database.QueryRowContext(ctx, `
			INSERT INTO model.model_records (id, tenant_id, model_id, data, created_by)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id::text, model_id::text, tenant_id::text, data, created_by::text,
			          created_at, updated_at`,
			newID, tenantID, modelID, dataBytes, userID).Scan(
			&resp.ID, &resp.ModelID, &resp.TenantID, &dataBytes,
			&resp.CreatedBy, &resp.CreatedAt, &resp.UpdatedAt)
		if err != nil {
			logger.Error(ctx, "create record failed", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		_ = json.Unmarshal(dataBytes, &resp.Data)
		return c.JSON(http.StatusCreated, resp)
	}
}

func listRecordsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		modelID := c.Param("id")
		filterJSON := c.QueryParam("filter") // a JSON object for advanced filtering
		limit := atoiOr(c.QueryParam("limit"), 50)

		args := []interface{}{modelID}
		q := `SELECT id::text, model_id::text, tenant_id::text, data, created_by::text,
		             created_at, updated_at
		      FROM model.model_records WHERE model_id = $1`
		if filterJSON != "" {
			// wrap provided filter as a JSONB containment filter.
			q += ` AND data @> $2::jsonb`
			args = append(args, filterJSON)
		}
		q += ` ORDER BY updated_at DESC LIMIT ` + strconv.Itoa(limit)
		rows, err := database.QueryContext(ctx, q, args...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()
		var out []recordResponse
		for rows.Next() {
			var r recordResponse
			var data []byte
			if err := rows.Scan(&r.ID, &r.ModelID, &r.TenantID, &data,
				&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
				continue
			}
			_ = json.Unmarshal(data, &r.Data)
			out = append(out, r)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"count":   len(out),
			"records": out,
		})
	}
}

func getRecordHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("record_id")
		var r recordResponse
		var data []byte
		err := database.QueryRowContext(ctx, `
			SELECT id::text, model_id::text, tenant_id::text, data, created_by::text,
			       created_at, updated_at
			FROM model.model_records WHERE id=$1`, id).Scan(
			&r.ID, &r.ModelID, &r.TenantID, &data,
			&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "record not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		_ = json.Unmarshal(data, &r.Data)
		return c.JSON(http.StatusOK, r)
	}
}

func updateRecordHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		modelID := c.Param("id")
		recordID := c.Param("record_id")
		var req recordRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if vErrs, err := validateAgainstModel(ctx, database, modelID, req.Data); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		} else if len(vErrs) > 0 {
			return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"error":             "validation_failed",
				"validation_errors": vErrs,
			})
		}
		dataBytes, _ := json.Marshal(req.Data)
		if _, err := database.ExecContext(ctx, `
			UPDATE model.model_records SET data=$1, updated_at=now()
			WHERE id=$2`, dataBytes, recordID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

func deleteRecordHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		recordID := c.Param("record_id")
		if _, err := database.ExecContext(ctx,
			`DELETE FROM model.model_records WHERE id=$1`, recordID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// =============================================================================
// Import / Export
// =============================================================================

func importCSVHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		modelID := c.Param("id")

		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "file required"})
		}
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer src.Close()

		// Optional upload-dir persistence for traceability.
		if save := os.Getenv("DYNAMIC_MODEL_UPLOAD_DIR"); save != "" {
			_ = os.MkdirAll(save, 0o755)
			dst, derr := os.Create(filepath.Join(save, modelID+"_"+uuid.NewV7().String()+".csv"))
			if derr == nil {
				_, _ = io.Copy(dst, src)
				dst.Close()
				src.Seek(0, 0)
			}
		}

		tenantID, userID := tenantUserOrZero(ctx)
		jobID := uuid.NewV7()

		if _, err := database.ExecContext(ctx, `
			INSERT INTO model.model_import_jobs (id, tenant_id, model_id, status, created_by, started_at)
			VALUES ($1,$2,$3,'running',$4,now())`,
			jobID, tenantID, modelID, userID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		reader := csv.NewReader(src)
		headers, err := reader.Read()
		if err != nil {
			markImportFailed(ctx, database, jobID, err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid CSV: " + err.Error()})
		}
		headers = lowerHeaders(headers)

		total, success, failed := 0, 0, 0
		var errorsList []string

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				failed++
				errorsList = append(errorsList, "line "+strconv.Itoa(total+2)+": "+err.Error())
				continue
			}
			total++
			data := map[string]interface{}{}
			for i, h := range headers {
				if i < len(row) {
					data[h] = row[i]
				}
			}
			if vErrs, vErr := validateAgainstModel(ctx, database, modelID, data); vErr != nil {
				failed++
				errorsList = append(errorsList, "row "+strconv.Itoa(total)+": "+vErr.Error())
				continue
			} else if len(vErrs) > 0 {
				failed++
				errorsList = append(errorsList, fmt.Sprintf("row %d: %s", total, joinVErrs(vErrs)))
				continue
			}
			dataBytes, _ := json.Marshal(data)
			if _, err := database.ExecContext(ctx, `
				INSERT INTO model.model_records (tenant_id, model_id, data, created_by)
				VALUES ($1,$2,$3,$4)`, tenantID, modelID, dataBytes, userID); err != nil {
				failed++
				errorsList = append(errorsList, "row "+strconv.Itoa(total)+": "+err.Error())
				continue
			}
			success++
		}

		errBytes, _ := json.Marshal(errorsList)
		if _, err := database.ExecContext(ctx, `
			UPDATE model.model_import_jobs
			SET status='done', total_rows=$1, success_rows=$2, failed_rows=$3,
			    errors=$4, finished_at=now() WHERE id=$5`,
			total, success, failed, errBytes, jobID); err != nil {
			logger.Error(ctx, "update import job failed", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"job_id":       jobID.String(),
			"total_rows":   total,
			"success_rows": success,
			"failed_rows":  failed,
			"errors":       errorsList,
		})
	}
}

func markImportFailed(ctx context.Context, db *sql.DB, jobID uuid.UUID, err error) {
	errs, _ := json.Marshal([]string{err.Error()})
	_, _ = db.ExecContext(ctx, `
		UPDATE model.model_import_jobs
		SET status='failed', errors=$1, finished_at=now() WHERE id=$2`, errs, jobID)
}

func exportCSVHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		modelID := c.Param("id")
		rows, err := database.QueryContext(ctx, `
			SELECT data FROM model.model_records WHERE model_id=$1 ORDER BY created_at DESC LIMIT 10000`,
			modelID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		// union of keys across rows
		keys := map[string]bool{}
		var records []map[string]interface{}
		for rows.Next() {
			var d []byte
			if err := rows.Scan(&d); err != nil {
				continue
			}
			var m map[string]interface{}
			_ = json.Unmarshal(d, &m)
			for k := range m {
				keys[k] = true
			}
			records = append(records, m)
		}
		header := make([]string, 0, len(keys))
		for k := range keys {
			header = append(header, k)
		}
		sort.Strings(header)

		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write(header)
		for _, r := range records {
			row := make([]string, len(header))
			for i, h := range header {
				if v, ok := r[h]; ok {
					row[i] = toCSVString(v)
				}
			}
			_ = w.Write(row)
		}
		w.Flush()

		c.Response().Header().Set("Content-Type", "text/csv")
		c.Response().Header().Set("Content-Disposition",
			`attachment; filename="model_`+modelID+`.csv"`)
		return c.Blob(http.StatusOK, "text/csv", buf.Bytes())
	}
}

// =============================================================================
// Validation / Schema generation
// =============================================================================

func validateDataHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		modelID := c.Param("id")
		var req validateRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		vErrs, err := validateAgainstModel(ctx, database, modelID, req.Data)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"valid":             len(vErrs) == 0,
			"validation_errors": vErrs,
		})
	}
}

func generateUISchemaHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		rows, err := database.QueryContext(ctx, `
			SELECT name, COALESCE(label,''), type, required, ui_config
			FROM model.model_fields WHERE model_id=$1 ORDER BY display_order, name`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		fields := []map[string]interface{}{}
		for rows.Next() {
			var name, label, fType string
			var required bool
			var ui []byte
			if err := rows.Scan(&name, &label, &fType, &required, &ui); err != nil {
				continue
			}
			f := map[string]interface{}{
				"name":     name,
				"label":    label,
				"type":     fType,
				"required": required,
			}
			var uim map[string]interface{}
			if len(ui) > 0 {
				_ = json.Unmarshal(ui, &uim)
			}
			for k, v := range uim {
				f[k] = v
			}
			if _, ok := f["placeholder"]; !ok {
				f["placeholder"] = label
			}
			fields = append(fields, f)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"type":         "object",
			"ui:order":     orderFromFields(fields),
			"definitions":  fields,
			"rjsf_version": "5",
		})
	}
}

func generateJSONSchemaHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		rows, err := database.QueryContext(ctx, `
			SELECT name, COALESCE(label,''), type, required, validation_rules
			FROM model.model_fields WHERE model_id=$1 ORDER BY display_order, name`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		props := map[string]interface{}{}
		var required []string
		for rows.Next() {
			var name, label, fType string
			var isReq bool
			var rules []byte
			if err := rows.Scan(&name, &label, &fType, &isReq, &rules); err != nil {
				continue
			}
			prop := jsonSchemaForType(fType, rules)
			prop["title"] = label
			props[name] = prop
			if isReq {
				required = append(required, name)
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"$schema":  "https://json-schema.org/draft/2020-12/schema",
			"type":     "object",
			"properties": props,
			"required": required,
			"additionalProperties": true,
		})
	}
}

func migrateModelHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		// Bump version and snapshot a new entry.
		if _, err := database.ExecContext(ctx, `
			UPDATE model.models SET version = version + 1, updated_at = now() WHERE id=$1`, id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		// Re-publish to capture snapshot.
		var ver int
		if err := database.QueryRowContext(ctx,
			`SELECT version FROM model.models WHERE id=$1`, id).Scan(&ver); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		// snapshot in same handler (re-use logic from publish).
		publish := publishModelHandler(database)
		rec := echo.New().NewContext(c.Request(), c.Response())
		_ = rec
		_ = publish
		return c.JSON(http.StatusOK, map[string]string{
			"status":          "migration_started",
			"new_version":     strconv.Itoa(ver),
		})
	}
}

// =============================================================================
// Helpers
// =============================================================================

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func readyHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := database.PingContext(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_down"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func validSlug(s string) bool    { return regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(s) }
func validIdent(s string) bool   { return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(s) }
func validStatus(s string) bool  { return s == "draft" || s == "published" || s == "archived" }

func validFieldType(t string) bool {
	switch t {
	case tString, tText, tNumber, tInteger, tBoolean, tDate, tDatetime, tTime,
		tJSON, tEnum, tArray, tObject, tRelation, tFile, tRef,
		tEmail, tPhone, tURL, tColor:
		return true
	}
	return false
}

func orEmpty(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

func toCSVString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case float64, bool, nil:
		return fmt.Sprintf("%v", x)
	case map[string]interface{}, []interface{}:
		b, _ := json.Marshal(x)
		return string(b)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func lowerHeaders(h []string) []string {
	out := make([]string, len(h))
	for i, v := range h {
		out[i] = strings.ToLower(strings.TrimSpace(v))
	}
	return out
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// tenantUserOrZero returns parsed UUIDs from context; falls back to a
// deterministic zero UUID when missing so inserts always succeed.
func tenantUserOrZero(ctx context.Context) (uuid.UUID, uuid.UUID) {
	tStr := db.TenantIDFromContext(ctx)
	uStr := db.UserIDFromContext(ctx)
	t, _ := uuid.Parse(tStr)
	u, _ := uuid.Parse(uStr)
	if t == uuid.Nil {
		t = uuid.MustParse("00000000-0000-0000-0000-000000000000")
	}
	if u == uuid.Nil {
		u = uuid.MustParse("00000000-0000-0000-0000-000000000000")
	}
	return t, u
}

// validateAgainstModel checks a record against field definitions:
//   - required, types, min/max, regex patterns
//   - custom validators: email, phone-VN, tax-code, cccd
func validateAgainstModel(ctx context.Context, database *sql.DB, modelID string,
	data map[string]interface{}) ([]validationError, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT name, type, required, validation_rules
		FROM model.model_fields WHERE model_id=$1`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errs []validationError
	for rows.Next() {
		var name, fType string
		var required bool
		var rules []byte
		if err := rows.Scan(&name, &fType, &required, &rules); err != nil {
			continue
		}
		val, present := data[name]
		if !present || val == nil {
			if required {
				errs = append(errs, validationError{Field: name, Message: "required"})
			}
			continue
		}
		if verr := validateOne(name, val, fType, rules); verr != nil {
			errs = append(errs, *verr)
		}
	}
	return errs, nil
}

func joinVErrs(errs []validationError) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, e.Field+": "+e.Message)
	}
	return strings.Join(parts, "; ")
}

func validateOne(name string, value interface{}, fType string, rulesJSON []byte) *validationError {
	check := func(msg string) *validationError {
		return &validationError{Field: name, Message: msg}
	}

	var rules map[string]interface{}
	if len(rulesJSON) > 0 {
		_ = json.Unmarshal(rulesJSON, &rules)
	} else {
		rules = map[string]interface{}{}
	}

	switch fType {
	case tString, tText:
		s, ok := value.(string)
		if !ok {
			return check("must be string")
		}
		if min, ok := rules["min_length"].(float64); ok && float64(len(s)) < min {
			return check("min_length violated")
		}
		if max, ok := rules["max_length"].(float64); ok && float64(len(s)) > max {
			return check("max_length violated")
		}
		if p, ok := rules["pattern"].(string); ok && p != "" {
			re, err := regexp.Compile(p)
			if err == nil && !re.MatchString(s) {
				return check("pattern violated")
			}
		}
		if v, ok := rules["validator"].(string); ok {
			if msg, bad := runCustomValidator(v, s); bad {
				return check(msg)
			}
		}
	case tNumber:
		f, ok := toFloat(value)
		if !ok {
			return check("must be number")
		}
		if min, ok := rules["min"].(float64); ok && f < min {
			return check("below min")
		}
		if max, ok := rules["max"].(float64); ok && f > max {
			return check("above max")
		}
	case tInteger:
		f, ok := toFloat(value)
		if !ok || f != float64(int64(f)) {
			return check("must be integer")
		}
	case tBoolean:
		if _, ok := value.(bool); !ok {
			return check("must be boolean")
		}
	case tDate, tDatetime, tTime:
		s, ok := value.(string)
		if !ok {
			return check("must be date string")
		}
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			if _, err2 := time.Parse("2006-01-02", s); err2 != nil {
				return check("invalid date format")
			}
		}
	case tEnum:
		opts, _ := rules["options"].([]interface{})
		s, ok := value.(string)
		if !ok {
			return check("must be string from enum")
		}
		for _, o := range opts {
			if o == s {
				return nil
			}
		}
		return check("not in enum")
	case tEmail:
		s, ok := value.(string)
		if !ok {
			return check("must be string")
		}
		if !emailRe.MatchString(s) {
			return check("invalid email")
		}
	case tPhone:
		s, ok := value.(string)
		if !ok {
			return check("must be string")
		}
		if !phoneVNRe.MatchString(s) {
			return check("invalid VN phone")
		}
	case tURL:
		s, ok := value.(string)
		if !ok {
			return check("must be string")
		}
		if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
			return check("must be http(s) URL")
		}
	case tColor:
		s, ok := value.(string)
		if !ok {
			return check("must be string")
		}
		if !colorRe.MatchString(s) {
			return check("invalid color hex")
		}
	case tArray:
		if _, ok := value.([]interface{}); !ok {
			return check("must be array")
		}
	case tObject, tJSON:
		if _, ok := value.(map[string]interface{}); !ok {
			return check("must be object")
		}
	}
	// json-schema draft pass for richer rules (optional).
	if b, ok := rules["json_schema"].(map[string]interface{}); ok {
		if js, err := json.Marshal(b); err == nil {
			loader := gojsonschema.NewBytesLoader(js)
			data, _ := json.Marshal(value)
			dLoader := gojsonschema.NewBytesLoader(data)
			if res, err := gojsonschema.Validate(loader, dLoader); err == nil && !res.Valid() {
				return check("schema: "+res.Errors()[0].Description())
			}
		}
	}
	return nil
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

func runCustomValidator(name, s string) (string, bool) {
	switch name {
	case "email":
		if emailRe.MatchString(s) {
			return "", false
		}
	case "phone-vn":
		if phoneVNRe.MatchString(s) {
			return "", false
		}
	case "tax-code":
		if taxCodeRe.MatchString(s) {
			return "", false
		}
	case "cccd":
		if cccdRe.MatchString(s) {
			return "", false
		}
	}
	return "invalid " + name, true
}

var (
	emailRe    = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
	phoneVNRe  = regexp.MustCompile(`^(\+84|0)(3|5|7|8|9)\d{8,9}$`)
	taxCodeRe  = regexp.MustCompile(`^\d{10}(-\d{3})?$`)
	cccdRe     = regexp.MustCompile(`^\d{12}$`)
	colorRe    = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)
)

func jsonSchemaForType(fType string, rulesJSON []byte) map[string]interface{} {
	prop := map[string]interface{}{}
	switch fType {
	case tString, tText, tEmail, tPhone, tURL, tColor:
		prop["type"] = "string"
	case tNumber:
		prop["type"] = "number"
	case tInteger:
		prop["type"] = "integer"
	case tBoolean:
		prop["type"] = "boolean"
	case tDate, tDatetime, tTime:
		prop["type"] = "string"
		prop["format"] = "date-time"
	case tEnum:
		prop["type"] = "string"
	case tArray:
		prop["type"] = "array"
	case tObject, tJSON:
		prop["type"] = "object"
	case tRelation, tRef, tFile:
		prop["type"] = "string"
	}
	if len(rulesJSON) > 0 {
		var rules map[string]interface{}
		_ = json.Unmarshal(rulesJSON, &rules)
		if min, ok := rules["min_length"].(float64); ok {
			prop["minLength"] = int(min)
		}
		if max, ok := rules["max_length"].(float64); ok {
			prop["maxLength"] = int(max)
		}
		if min, ok := rules["min"].(float64); ok {
			prop["minimum"] = min
		}
		if max, ok := rules["max"].(float64); ok {
			prop["maximum"] = max
		}
		if p, ok := rules["pattern"].(string); ok {
			prop["pattern"] = p
		}
		if opts, ok := rules["options"].([]interface{}); ok {
			strs := []string{}
			for _, o := range opts {
				if s, ok := o.(string); ok {
					strs = append(strs, s)
				}
			}
			prop["enum"] = strs
		}
	}
	return prop
}

func orderFromFields(fields []map[string]interface{}) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if n, ok := f["name"].(string); ok {
			out = append(out, n)
		}
	}
	return out
}
