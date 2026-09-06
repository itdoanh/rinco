// Dynamic Model Service - allows tenants to define custom fields + workflows.
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
	serviceName = "dynamic-model-service"
	version     = "1.0.0"
)

type fieldDefRequest struct {
	EntityType   string                 `json:"entity_type" validate:"required"`
	Name         string                 `json:"name" validate:"required"`
	Label        string                 `json:"label"`
	FieldType    string                 `json:"field_type" validate:"required"`
	IsRequired   bool                   `json:"is_required"`
	DefaultValue map[string]interface{} `json:"default_value"`
	Validation   map[string]interface{} `json:"validation"`
	Options      map[string]interface{} `json:"options"`
	DisplayOrder int                    `json:"display_order"`
}

type fieldDefResponse struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	EntityType   string                 `json:"entity_type"`
	Name         string                 `json:"name"`
	Label        string                 `json:"label"`
	FieldType    string                 `json:"field_type"`
	IsRequired   bool                   `json:"is_required"`
	DefaultValue map[string]interface{} `json:"default_value"`
	Validation   map[string]interface{} `json:"validation"`
	Options      map[string]interface{} `json:"options"`
	DisplayOrder int                    `json:"display_order"`
	CreatedAt    time.Time              `json:"created_at"`
}

type workflowRequest struct {
	Name          string                 `json:"name" validate:"required"`
	TriggerEntity string                 `json:"trigger_entity" validate:"required"`
	TriggerEvent  string                 `json:"trigger_event" validate:"required"`
	Actions       []map[string]interface{} `json:"actions"`
	Conditions    []map[string]interface{} `json:"conditions"`
	IsActive      bool                   `json:"is_active"`
}

type workflowResponse struct {
	ID            string                   `json:"id"`
	TenantID      string                   `json:"tenant_id"`
	Name          string                   `json:"name"`
	TriggerEntity string                   `json:"trigger_entity"`
	TriggerEvent  string                   `json:"trigger_event"`
	Actions       []map[string]interface{} `json:"actions"`
	Conditions    []map[string]interface{} `json:"conditions"`
	IsActive      bool                     `json:"is_active"`
	CreatedAt     time.Time                `json:"created_at"`
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

	// Field definitions
	dm := e.Group("/v1/dynamic-model")
	dm.Use(authMiddleware(database))

	dm.POST("/field-defs", createFieldDefHandler(database))
	dm.GET("/field-defs", listFieldDefsHandler(database))
	dm.PATCH("/field-defs/:id", updateFieldDefHandler(database))
	dm.DELETE("/field-defs/:id", deleteFieldDefHandler(database))

	// Workflows
	dm.POST("/workflows", createWorkflowHandler(database))
	dm.GET("/workflows", listWorkflowsHandler(database))
	dm.PATCH("/workflows/:id", updateWorkflowHandler(database))
	dm.DELETE("/workflows/:id", deleteWorkflowHandler(database))

	// Schema generator (returns JSON Schema for tenant)
	dm.GET("/schema/:entity_type", getEntitySchemaHandler(database))

	port := ":" + getEnv("PORT", "8084")
	logger.Info(context.Background(), "starting dynamic-model service", zap.String("port", port))
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
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant or user header"})
			}

			ctx = db.SetTenantContext(ctx, tenantID, userID, isAdmin)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func createFieldDefHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req fieldDefRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		tenantID := logger.TenantIDFromContext(ctx)
		id := uuid.NewV7()

		defaultValueJSON, _ := json.Marshal(req.DefaultValue)
		validationJSON, _ := json.Marshal(req.Validation)
		optionsJSON, _ := json.Marshal(req.Options)

		var createdAt time.Time
		err := database.QueryRowContext(ctx, `
			INSERT INTO workflow.entity_field_defs (
				id, tenant_id, entity_type, name, label, field_type,
				is_required, default_value, validation, options, display_order
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING created_at
		`, id, tenantID, req.EntityType, req.Name, req.Label, req.FieldType,
			req.IsRequired, defaultValueJSON, validationJSON, optionsJSON, req.DisplayOrder).Scan(&createdAt)

		if err != nil {
			logger.Error(ctx, "create field def", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, fieldDefResponse{
			ID:           id.String(),
			TenantID:     tenantID,
			EntityType:   req.EntityType,
			Name:         req.Name,
			Label:        req.Label,
			FieldType:    req.FieldType,
			IsRequired:   req.IsRequired,
			DefaultValue: req.DefaultValue,
			Validation:   req.Validation,
			Options:      req.Options,
			DisplayOrder: req.DisplayOrder,
			CreatedAt:    createdAt,
		})
	}
}

func listFieldDefsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		entityType := c.QueryParam("entity_type")

		query := `SELECT id::text, tenant_id::text, entity_type, name, COALESCE(label, ''),
			field_type, is_required, default_value, validation, options, display_order, created_at
			FROM workflow.entity_field_defs`
		args := []interface{}{}
		if entityType != "" {
			query += ` WHERE entity_type = $1`
			args = append(args, entityType)
		}
		query += ` ORDER BY display_order, created_at`

		rows, err := database.QueryContext(ctx, query, args...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var defs []fieldDefResponse
		for rows.Next() {
			var d fieldDefResponse
			var dv, v, o []byte
			if err := rows.Scan(&d.ID, &d.TenantID, &d.EntityType, &d.Name, &d.Label,
				&d.FieldType, &d.IsRequired, &dv, &v, &o, &d.DisplayOrder, &d.CreatedAt); err != nil {
				continue
			}
			json.Unmarshal(dv, &d.DefaultValue)
			json.Unmarshal(v, &d.Validation)
			json.Unmarshal(o, &d.Options)
			defs = append(defs, d)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"count":     len(defs),
			"field_defs": defs,
		})
	}
}

func updateFieldDefHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")

		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		allowed := []string{"label", "is_required", "default_value", "validation", "options", "display_order"}
		sets := []string{}
		args := []interface{}{}
		idx := 1
		for _, f := range allowed {
			if v, ok := body[f]; ok {
				if v == nil {
					continue
				}
				sets = append(sets, f+" = $"+itoa(idx))
				args = append(args, v)
				idx++
			}
		}

		if len(sets) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no fields to update"})
		}

		args = append(args, id)
		query := "UPDATE workflow.entity_field_defs SET " + joinStrings(sets, ", ") + " WHERE id = $" + itoa(idx)
		_, err := database.ExecContext(ctx, query, args...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

func deleteFieldDefHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		_, err := database.ExecContext(ctx, `DELETE FROM workflow.entity_field_defs WHERE id = $1`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func createWorkflowHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req workflowRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		tenantID := logger.TenantIDFromContext(ctx)
		id := uuid.NewV7()

		actionsJSON, _ := json.Marshal(req.Actions)
		conditionsJSON, _ := json.Marshal(req.Conditions)

		var createdAt time.Time
		err := database.QueryRowContext(ctx, `
			INSERT INTO workflow.workflows (
				id, tenant_id, name, trigger_entity, trigger_event,
				actions, conditions, is_active
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING created_at
		`, id, tenantID, req.Name, req.TriggerEntity, req.TriggerEvent,
			actionsJSON, conditionsJSON, req.IsActive).Scan(&createdAt)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, workflowResponse{
			ID:            id.String(),
			TenantID:      tenantID,
			Name:          req.Name,
			TriggerEntity: req.TriggerEntity,
			TriggerEvent:  req.TriggerEvent,
			Actions:       req.Actions,
			Conditions:    req.Conditions,
			IsActive:      req.IsActive,
			CreatedAt:     createdAt,
		})
	}
}

func listWorkflowsHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		rows, err := database.QueryContext(ctx, `
			SELECT id::text, tenant_id::text, name, trigger_entity, trigger_event,
				actions, conditions, is_active, created_at
			FROM workflow.workflows
			ORDER BY created_at DESC
		`)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var wfs []workflowResponse
		for rows.Next() {
			var w workflowResponse
			var a, c2 []byte
			if err := rows.Scan(&w.ID, &w.TenantID, &w.Name, &w.TriggerEntity, &w.TriggerEvent,
				&a, &c2, &w.IsActive, &w.CreatedAt); err != nil {
				continue
			}
			json.Unmarshal(a, &w.Actions)
			json.Unmarshal(c2, &w.Conditions)
			wfs = append(wfs, w)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"count":     len(wfs),
			"workflows": wfs,
		})
	}
}

func updateWorkflowHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		_, err := database.ExecContext(ctx, `
			UPDATE workflow.workflows SET
				name = COALESCE($1, name),
				actions = COALESCE($2, actions),
				conditions = COALESCE($3, conditions),
				is_active = COALESCE($4, is_active)
			WHERE id = $5
		`, body["name"], body["actions"], body["conditions"], body["is_active"], id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

func deleteWorkflowHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		id := c.Param("id")
		_, err := database.ExecContext(ctx, `DELETE FROM workflow.workflows WHERE id = $1`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func getEntitySchemaHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		entityType := c.Param("entity_type")

		rows, err := database.QueryContext(ctx, `
			SELECT name, label, field_type, is_required, default_value, validation, options
			FROM workflow.entity_field_defs
			WHERE entity_type = $1
			ORDER BY display_order
		`, entityType)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		// Build JSON Schema
		properties := map[string]interface{}{}
		required := []string{}
		for rows.Next() {
			var name, fieldType string
			var label *string
			var isReq bool
			var dv, v, o []byte
			if err := rows.Scan(&name, &label, &fieldType, &isReq, &dv, &v, &o); err != nil {
				continue
			}
			property := map[string]interface{}{}
			switch fieldType {
			case "string", "text":
				property["type"] = "string"
			case "number", "integer":
				property["type"] = "number"
			case "boolean":
				property["type"] = "boolean"
			case "date":
				property["type"] = "string"
				property["format"] = "date"
			case "select":
				property["type"] = "string"
				property["enum"] = []string{}
			case "multiselect":
				property["type"] = "array"
				property["items"] = map[string]interface{}{"type": "string"}
			default:
				property["type"] = "string"
			}
			if label != nil {
				property["title"] = *label
			}
			properties[name] = property
			if isReq {
				required = append(required, name)
			}
		}

		schema := map[string]interface{}{
			"$schema":     "http://json-schema.org/draft-07/schema#",
			"type":        "object",
			"properties":  properties,
			"required":    required,
		}

		return c.JSON(http.StatusOK, schema)
	}
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
