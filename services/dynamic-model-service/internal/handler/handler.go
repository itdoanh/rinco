package handler

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

	"github.com/rinco/services/dynamic-model-service/internal/validation"
)

const modelSchema = "model"

type Server struct { db *sql.DB }

func New(db *sql.DB) *Server { return &Server{db: db} }

// ---------------- Models CRUD ---------------- //

type modelResp struct {
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

type createModelReq struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9_-]+$`)

func (s *Server) CreateModel(c echo.Context) error {
	ctx := c.Request().Context()
	var req createModelReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" { return c.JSON(http.StatusBadRequest, map[string]string{"error": "name and slug are required"}) }
	if !slugRe.MatchString(req.Slug) { return c.JSON(http.StatusBadRequest, map[string]string{"error": "slug must match [a-z0-9_-]+"}) }
	tenantID, userID := ctxTenantUser(c)
	id := uuid.NewString()
	resp := modelResp{}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`INSERT INTO %s.models (id, tenant_id, name, slug, version, status, created_by, description) VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, 1, 'draft', NULLIF($5,'')::uuid, $6) RETURNING id, COALESCE(tenant_id::text,''), name, slug, version, status, COALESCE(description,''), COALESCE(created_by::text,''), created_at, updated_at`, modelSchema),
		id, tenantID, req.Name, req.Slug, userID, req.Description).Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.Slug, &resp.Version, &resp.Status, &resp.Description, &resp.CreatedBy, &resp.CreatedAt, &resp.UpdatedAt)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusCreated, resp)
}

func (s *Server) ListModels(c echo.Context) error {
	ctx := c.Request().Context()
	status := c.QueryParam("status"); slug := c.QueryParam("slug")
	q := fmt.Sprintf(`SELECT id::text, COALESCE(tenant_id::text,''), name, slug, version, status, COALESCE(description,''), COALESCE(created_by::text,''), created_at, updated_at FROM %s.models WHERE deleted_at IS NULL`, modelSchema)
	args := []interface{}{}; idx := 1
	if status != "" { q += fmt.Sprintf(" AND status = $%d", idx); args = append(args, status); idx++ }
	if slug != "" { q += fmt.Sprintf(" AND slug = $%d", idx); args = append(args, slug); idx++ }
	q += " ORDER BY created_at DESC LIMIT 500"
	rows, err := s.db.QueryContext(ctx, q, args...); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer rows.Close()
	out := []modelResp{}
	for rows.Next() {
		var m modelResp
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Name, &m.Slug, &m.Version, &m.Status, &m.Description, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err == nil { out = append(out, m) }
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"count": len(out), "models": out})
}

func (s *Server) GetModel(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	resp := modelResp{}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT id::text, COALESCE(tenant_id::text,''), name, slug, version, status, COALESCE(description,''), COALESCE(created_by::text,''), created_at, updated_at FROM %s.models WHERE id = $1::uuid AND deleted_at IS NULL`, modelSchema), id).Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.Slug, &resp.Version, &resp.Status, &resp.Description, &resp.CreatedBy, &resp.CreatedAt, &resp.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) { return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"}) }
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, resp)
}

type updateModelReq struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (s *Server) UpdateModel(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	var req updateModelReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	sets := []string{}; args := []interface{}{}; idx := 1
	if req.Name != nil { sets = append(sets, fmt.Sprintf("name = $%d", idx)); args = append(args, *req.Name); idx++ }
	if req.Description != nil { sets = append(sets, fmt.Sprintf("description = $%d", idx)); args = append(args, *req.Description); idx++ }
	if req.Status != nil { if *req.Status != "draft" && *req.Status != "published" && *req.Status != "archived" { return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid status"}) }; sets = append(sets, fmt.Sprintf("status = $%d", idx)); args = append(args, *req.Status); idx++ }
	if len(sets) == 0 { return c.JSON(http.StatusBadRequest, map[string]string{"error": "no fields to update"}) }
	sets = append(sets, "updated_at = now()"); args = append(args, id)
	q := fmt.Sprintf(`UPDATE %s.models SET %s WHERE id = $%d::uuid AND deleted_at IS NULL`, modelSchema, strings.Join(sets, ", "), idx)
	if _, err := s.db.ExecContext(ctx, q, args...); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) DeleteModel(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s.models SET deleted_at = now(), status='archived' WHERE id = $1::uuid`, modelSchema), id); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "soft_deleted"})
}

type duplicateReq struct {
	TargetTenantID string `json:"target_tenant_id"`
	NewSlug        string `json:"new_slug"`
}

func (s *Server) DuplicateModel(c echo.Context) error {
	srcID := c.Param("id"); ctx := c.Request().Context()
	var body duplicateReq; if err := c.Bind(&body); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	if body.TargetTenantID == "" { return c.JSON(http.StatusBadRequest, map[string]string{"error": "target_tenant_id required"}) }
	targetUUID := body.TargetTenantID
	tx, err := s.db.BeginTx(ctx, nil); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }; defer tx.Rollback()
	var srcName, srcSlug, srcDesc string; var srcVer int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT name, slug, COALESCE(description,''), version FROM %s.models WHERE id=$1::uuid`, modelSchema), srcID).Scan(&srcName, &srcSlug, &srcDesc, &srcVer); err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error": "source model not found"}) }
	newSlug := body.NewSlug; if newSlug == "" { newSlug = srcSlug + "_copy" }
	newID := uuid.NewString()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s.models (id, tenant_id, name, slug, version, status, created_by, description) VALUES ($1::uuid, $2::uuid, $3, $4, 1, 'draft', $5::uuid, $6)`, modelSchema), newID, targetUUID, srcName+" (Copy)", newSlug, zeroUUID(), srcDesc); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s.model_fields (model_id, name, label, type, required, "default", validation_rules, ui_config, display_order) SELECT $1::uuid, name, label, type, required, "default", validation_rules, ui_config, display_order FROM %s.model_fields WHERE model_id = $2::uuid`, modelSchema, modelSchema), newID, srcID); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if err := tx.Commit(); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusCreated, map[string]string{"id": newID, "slug": newSlug, "status": "duplicated"})
}

type publishReq struct { Note string `json:"note"` }

func (s *Server) PublishModel(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	var body publishReq; _ = c.Bind(&body)
	tx, err := s.db.BeginTx(ctx, nil); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }; defer tx.Rollback()
	var ver int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT version FROM %s.models WHERE id=$1::uuid`, modelSchema), id).Scan(&ver); err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"}) }
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT name, label, type, required, "default", validation_rules, ui_config, display_order FROM %s.model_fields WHERE model_id=$1::uuid`, modelSchema), id); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	fields := []map[string]interface{}{}
	for rows.Next() {
		var name, fieldType, label string; var required bool; var dv, vr, ui []byte; var order int
		if err := rows.Scan(&name, &label, &fieldType, &required, &dv, &vr, &ui, &order); err == nil {
			f := map[string]interface{}{"name": name, "label": label, "type": fieldType, "required": required, "display_order": order}
			fields = append(fields, attachJSON(f, "default", dv, "validation_rules", vr, "ui_config", ui))
		}
	}
	rows.Close()
	snap, _ := json.Marshal(map[string]interface{}{"fields": fields, "version": ver})
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s.model_versions (model_id, version, snapshot, created_by, note) VALUES ($1::uuid, $2, $3, $4::uuid, $5)`, modelSchema), id, ver, snap, zeroUUID(), body.Note); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s.models SET status='published', updated_at=now() WHERE id=$1::uuid`, modelSchema), id); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if err := tx.Commit(); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "published", "version": strconv.Itoa(ver)})
}

func (s *Server) ListVersions(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT id::text, model_id::text, version, snapshot, COALESCE(note,''), COALESCE(created_by::text,''), created_at FROM %s.model_versions WHERE model_id = $1::uuid ORDER BY version DESC`, modelSchema), id); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer rows.Close()
	type verResp struct { ID, ModelID, Note, CreatedBy string; Version int; Snapshot map[string]interface{}; CreatedAt time.Time }
	out := []verResp{}
	for rows.Next() {
		var v verResp; var snap []byte
		if err := rows.Scan(&v.ID, &v.ModelID, &v.Version, &snap, &v.Note, &v.CreatedBy, &v.CreatedAt); err == nil {
			_ = json.Unmarshal(snap, &v.Snapshot); out = append(out, v)
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"count": len(out), "versions": out})
}

func (s *Server) RestoreVersion(c echo.Context) error {
	id := c.Param("id"); vStr := c.Param("version"); ctx := c.Request().Context()
	v, err := strconv.Atoi(vStr); if err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid version"}) }
	var snap []byte
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT snapshot FROM %s.model_versions WHERE model_id=$1::uuid AND version=$2`, modelSchema), id, v).Scan(&snap); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return c.JSON(http.StatusNotFound, map[string]string{"error": "version not found"}) }
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	var snapMap struct{ Fields []map[string]interface{} `json:"fields"` }
	if err := json.Unmarshal(snap, &snapMap); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": "snapshot corrupt"}) }
	tx, err := s.db.BeginTx(ctx, nil); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }; defer tx.Rollback()
	var maxV int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT COALESCE(MAX(version),0)+1 FROM %s.model_versions WHERE model_id=$1::uuid`, modelSchema), id).Scan(&maxV); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s.models SET version=$1, status='draft', updated_at=now() WHERE id=$2::uuid`, modelSchema), maxV, id); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s.model_fields WHERE model_id=$1::uuid`, modelSchema), id); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	for _, f := range snapMap.Fields {
		name, _ := f["name"].(string); label, _ := f["label"].(string); fType, _ := f["type"].(string)
		required, _ := f["required"].(bool); orderF, _ := f["display_order"].(float64)
		dm, _ := f["default"].(map[string]interface{}); vm, _ := f["validation_rules"].(map[string]interface{}); um, _ := f["ui_config"].(map[string]interface{})
		if dm == nil { dm = map[string]interface{}{} }
		if vm == nil { vm = map[string]interface{}{} }
		if um == nil { um = map[string]interface{}{} }
		dvBytes, _ := json.Marshal(dm); vrBytes, _ := json.Marshal(vm); uiBytes, _ := json.Marshal(um)
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s.model_fields (model_id, name, label, type, required, "default", validation_rules, ui_config, display_order) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9)`, modelSchema), id, name, label, fType, required, dvBytes, vrBytes, uiBytes, int(orderF)); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	if err := tx.Commit(); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "restored", "new_version": strconv.Itoa(maxV)})
}

func (s *Server) MigrateModel(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s.models SET version = version + 1, updated_at = now() WHERE id=$1::uuid`, modelSchema), id); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	var ver int
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT version FROM %s.models WHERE id=$1::uuid`, modelSchema), id).Scan(&ver); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return s.PublishModel(c) // publish captures new snapshot
}

// ---------------- Fields ---------------- //

type fieldReq struct {
	Name            string                 `json:"name"`
	Label           string                 `json:"label"`
	Type            string                 `json:"type"`
	Required        bool                   `json:"required"`
	Default         map[string]interface{} `json:"default"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
	UIConfig        map[string]interface{} `json:"ui_config"`
	DisplayOrder    int                    `json:"display_order"`
}

var identRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func validFieldType(t string) bool {
	switch t {
	case "string", "text", "number", "integer", "boolean", "date", "datetime", "time", "json", "enum", "array", "object", "relation", "file", "ref", "email", "phone", "url", "color":
		return true
	}
	return false
}

type fieldResp struct {
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

func (s *Server) AddField(c echo.Context) error {
	modelID := c.Param("id"); ctx := c.Request().Context()
	var req fieldReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	if !validFieldType(req.Type) { return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid field type"}) }
	if !identRe.MatchString(req.Name) { return c.JSON(http.StatusBadRequest, map[string]string{"error": "name must match [a-zA-Z_][a-zA-Z0-9_]*"}) }
	dv, _ := json.Marshal(orEmpty(req.Default)); vr, _ := json.Marshal(orEmpty(req.ValidationRules)); ui, _ := json.Marshal(orEmpty(req.UIConfig))
	newID := uuid.NewString()
	resp := fieldResp{}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`INSERT INTO %s.model_fields (id, model_id, name, label, type, required, "default", validation_rules, ui_config, display_order) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id::text, model_id::text, name, COALESCE(label,''), type, required, "default", validation_rules, ui_config, display_order, created_at`, modelSchema), newID, modelID, req.Name, req.Label, req.Type, req.Required, dv, vr, ui, req.DisplayOrder).Scan(&resp.ID, &resp.ModelID, &resp.Name, &resp.Label, &resp.Type, &resp.Required, &dv, &vr, &ui, &resp.DisplayOrder, &resp.CreatedAt)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	_ = json.Unmarshal(dv, &resp.Default); _ = json.Unmarshal(vr, &resp.ValidationRules); _ = json.Unmarshal(ui, &resp.UIConfig)
	return c.JSON(http.StatusCreated, resp)
}

func (s *Server) ListFields(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT id::text, model_id::text, name, COALESCE(label,''), type, required, "default", validation_rules, ui_config, display_order, created_at FROM %s.model_fields WHERE model_id=$1::uuid ORDER BY display_order, name`, modelSchema), id); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer rows.Close()
	out := []fieldResp{}
	for rows.Next() {
		var f fieldResp; var dv, vr, ui []byte
		if err := rows.Scan(&f.ID, &f.ModelID, &f.Name, &f.Label, &f.Type, &f.Required, &dv, &vr, &ui, &f.DisplayOrder, &f.CreatedAt); err == nil {
			_ = json.Unmarshal(dv, &f.Default); _ = json.Unmarshal(vr, &f.ValidationRules); _ = json.Unmarshal(ui, &f.UIConfig); out = append(out, f)
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"count": len(out), "fields": out})
}

func (s *Server) DeleteField(c echo.Context) error {
	fieldID := c.Param("field_id"); ctx := c.Request().Context()
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s.model_fields WHERE id=$1::uuid`, modelSchema), fieldID); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------------- Records ---------------- //

type recordReq struct{ Data map[string]interface{} `json:"data"` }
type recordResp struct {
	ID, ModelID, TenantID, CreatedBy string; Data map[string]interface{}; CreatedAt, UpdatedAt time.Time }

func (s *Server) CreateRecord(c echo.Context) error {
	modelID := c.Param("id"); ctx := c.Request().Context()
	var req recordReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	tenantID, userID := ctxTenantUser(c)
	fields, err := s.fetchFields(ctx, modelID); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if errs, _ := validation.Validate(fields, req.Data); len(errs) > 0 { return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{"error": "validation_failed", "validation_errors": errs}) }
	newID := uuid.NewString()
	dataBytes, _ := json.Marshal(req.Data)
	resp := recordResp{}
	err = s.db.QueryRowContext(ctx, fmt.Sprintf(`INSERT INTO %s.model_records (id, tenant_id, model_id, data, created_by) VALUES ($1::uuid, NULLIF($2,'')::uuid, $3::uuid, $4, NULLIF($5,'')::uuid) RETURNING id::text, model_id::text, COALESCE(tenant_id::text,''), COALESCE(created_by::text,''), created_at, updated_at`, modelSchema), newID, tenantID, modelID, dataBytes, userID).Scan(&resp.ID, &resp.ModelID, &resp.TenantID, &resp.CreatedBy, &resp.CreatedAt, &resp.UpdatedAt)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	resp.Data = req.Data
	return c.JSON(http.StatusCreated, resp)
}

func (s *Server) ListRecords(c echo.Context) error {
	modelID := c.Param("id"); ctx := c.Request().Context()
	filter := c.QueryParam("filter"); limit, _ := strconv.Atoi(c.QueryParam("limit")); if limit <= 0 || limit > 1000 { limit = 50 }
	q := fmt.Sprintf(`SELECT id::text, model_id::text, COALESCE(tenant_id::text,''), data, COALESCE(created_by::text,''), created_at, updated_at FROM %s.model_records WHERE model_id = $1::uuid`, modelSchema)
	args := []interface{}{modelID}
	if filter != "" { q += ` AND data @> $2::jsonb`; args = append(args, filter) }
	q += " ORDER BY updated_at DESC LIMIT " + strconv.Itoa(limit)
	rows, err := s.db.QueryContext(ctx, q, args...); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer rows.Close()
	out := []recordResp{}
	for rows.Next() {
		var r recordResp; var data []byte
		if err := rows.Scan(&r.ID, &r.ModelID, &r.TenantID, &data, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err == nil { _ = json.Unmarshal(data, &r.Data); out = append(out, r) }
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"count": len(out), "records": out})
}

func (s *Server) GetRecord(c echo.Context) error {
	id := c.Param("record_id"); ctx := c.Request().Context()
	r := recordResp{}; var data []byte
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT id::text, model_id::text, COALESCE(tenant_id::text,''), data, COALESCE(created_by::text,''), created_at, updated_at FROM %s.model_records WHERE id=$1::uuid`, modelSchema), id).Scan(&r.ID, &r.ModelID, &r.TenantID, &data, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) { return c.JSON(http.StatusNotFound, map[string]string{"error": "record not found"}) }
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	_ = json.Unmarshal(data, &r.Data)
	return c.JSON(http.StatusOK, r)
}

func (s *Server) UpdateRecord(c echo.Context) error {
	modelID := c.Param("id"); recordID := c.Param("record_id"); ctx := c.Request().Context()
	var req recordReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	fields, err := s.fetchFields(ctx, modelID); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if errs, _ := validation.Validate(fields, req.Data); len(errs) > 0 { return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{"error": "validation_failed", "validation_errors": errs}) }
	dataBytes, _ := json.Marshal(req.Data)
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s.model_records SET data=$1, updated_at=now() WHERE id=$2::uuid`, modelSchema), dataBytes, recordID); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) DeleteRecord(c echo.Context) error {
	recordID := c.Param("record_id"); ctx := c.Request().Context()
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s.model_records WHERE id=$1::uuid`, modelSchema), recordID); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) ValidateData(c echo.Context) error {
	modelID := c.Param("id"); ctx := c.Request().Context()
	var req recordReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	fields, err := s.fetchFields(ctx, modelID); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	errs, _ := validation.Validate(fields, req.Data)
	return c.JSON(http.StatusOK, map[string]interface{}{"valid": len(errs) == 0, "validation_errors": errs})
}

func (s *Server) ImportCSV(c echo.Context) error {
	modelID := c.Param("id"); ctx := c.Request().Context()
	file, err := c.FormFile("file"); if err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": "file required"}) }
	src, err := file.Open(); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer src.Close()
	if save := os.Getenv("DYNAMIC_MODEL_UPLOAD_DIR"); save != "" {
		_ = os.MkdirAll(save, 0o755)
		if dst, derr := os.Create(filepath.Join(save, modelID+"_"+uuid.NewString()+".csv")); derr == nil {
			_, _ = io.Copy(dst, src); dst.Close(); src.Seek(0, 0)
		}
	}
	tenantID, userID := ctxTenantUser(c)
	jobID := uuid.NewString()
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s.model_import_jobs (id, tenant_id, model_id, status, created_by, started_at) VALUES ($1::uuid, NULLIF($2,'')::uuid, $3::uuid, 'running', NULLIF($4,'')::uuid, now())`, modelSchema), jobID, tenantID, modelID, userID); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	reader := csv.NewReader(src); headers, err := reader.Read(); if err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid CSV: " + err.Error()}) }
	headers = lower(headers)
	total, success, failed := 0, 0, 0; var errorsList []string
	fields, ferr := s.fetchFields(ctx, modelID); if ferr != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": ferr.Error()}) }
	for {
		row, err := reader.Read()
		if err == io.EOF { break }
		if err != nil { failed++; errorsList = append(errorsList, "line "+strconv.Itoa(total+2)+": "+err.Error()); continue }
		total++
		data := map[string]interface{}{}
		for i, h := range headers { if i < len(row) { data[h] = row[i] } }
		if vErrs, _ := validation.Validate(fields, data); len(vErrs) > 0 { failed++; errorsList = append(errorsList, fmt.Sprintf("row %d: %s", total, joinErrs(vErrs))); continue }
		dataBytes, _ := json.Marshal(data)
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s.model_records (tenant_id, model_id, data, created_by) VALUES (NULLIF($1,'')::uuid, $2::uuid, $3, NULLIF($4,'')::uuid)`, modelSchema), tenantID, modelID, dataBytes, userID); err != nil { failed++; errorsList = append(errorsList, "row "+strconv.Itoa(total)+": "+err.Error()); continue }
		success++
	}
	errBytes, _ := json.Marshal(errorsList)
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s.model_import_jobs SET status='done', total_rows=$1, success_rows=$2, failed_rows=$3, errors=$4, finished_at=now() WHERE id=$5::uuid`, modelSchema), total, success, failed, errBytes, jobID); err != nil { /* log */ }
	return c.JSON(http.StatusOK, map[string]interface{}{"job_id": jobID, "total_rows": total, "success_rows": success, "failed_rows": failed, "errors": errorsList})
}

func (s *Server) ExportCSV(c echo.Context) error {
	modelID := c.Param("id"); ctx := c.Request().Context()
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT data FROM %s.model_records WHERE model_id=$1::uuid ORDER BY created_at DESC LIMIT 10000`, modelSchema), modelID); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer rows.Close()
	keys := map[string]bool{}; var records []map[string]interface{}
	for rows.Next() { var d []byte; if err := rows.Scan(&d); err == nil { var m map[string]interface{}; _ = json.Unmarshal(d, &m); for k := range m { keys[k] = true }; records = append(records, m) } }
	header := make([]string, 0, len(keys)); for k := range keys { header = append(header, k) }
	sort.Strings(header)
	var buf bytes.Buffer; w := csv.NewWriter(&buf); _ = w.Write(header)
	for _, r := range records {
		row := make([]string, len(header))
		for i, h := range header { if v, ok := r[h]; ok { row[i] = toCSV(v) } }
		_ = w.Write(row)
	}
	w.Flush()
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="model_`+modelID+`.csv"`)
	return c.Blob(http.StatusOK, "text/csv", buf.Bytes())
}

func (s *Server) GenerateUISchema(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT name, COALESCE(label,''), type, required, ui_config FROM %s.model_fields WHERE model_id=$1::uuid ORDER BY display_order, name`, modelSchema), id); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer rows.Close()
	fields := []map[string]interface{}{}
	var order []string
	for rows.Next() { var name, label, fType string; var required bool; var ui []byte
		if err := rows.Scan(&name, &label, &fType, &required, &ui); err == nil {
			f := map[string]interface{}{"name": name, "label": label, "type": fType, "required": required}
			var uim map[string]interface{}; if len(ui) > 0 { _ = json.Unmarshal(ui, &uim) }
			for k, v := range uim { f[k] = v }
			if _, ok := f["placeholder"]; !ok { f["placeholder"] = label }
			fields = append(fields, f); order = append(order, name)
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"type": "object", "ui:order": order, "definitions": fields, "rjsf_version": "5"})
}

func (s *Server) GenerateJSONSchema(c echo.Context) error {
	id := c.Param("id"); ctx := c.Request().Context()
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT name, COALESCE(label,''), type, required, validation_rules FROM %s.model_fields WHERE model_id=$1::uuid ORDER BY display_order, name`, modelSchema), id); if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	defer rows.Close()
	props := map[string]interface{}{}; required := []string{}
	for rows.Next() { var name, label, fType string; var isReq bool; var rules []byte
		if err := rows.Scan(&name, &label, &fType, &isReq, &rules); err == nil {
			var rulesMap map[string]interface{}; if len(rules) > 0 { _ = json.Unmarshal(rules, &rulesMap) }
			prop := validation.JSONSchemaForType(fType, rulesMap); prop["title"] = label
			props[name] = prop
			if isReq { required = append(required, name) }
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": props, "required": required, "additionalProperties": true})
}

// ---------------- Helpers ---------------- //

func (s *Server) fetchFields(ctx context.Context, modelID string) ([]validation.Field, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT name, type, required, validation_rules FROM %s.model_fields WHERE model_id=$1::uuid`, modelSchema), modelID); if err != nil { return nil, err }
	defer rows.Close()
	out := []validation.Field{}
	for rows.Next() { var name, fType string; var required bool; var rules []byte
		if err := rows.Scan(&name, &fType, &required, &rules); err == nil {
			var rulesMap map[string]interface{}; if len(rules) > 0 { _ = json.Unmarshal(rules, &rulesMap) }
			out = append(out, validation.Field{Name: name, Type: fType, Required: required, ValidationRules: rulesMap})
		}
	}
	return out, nil
}

func ctxTenantUser(c echo.Context) (string, string) { return c.Request().Header.Get("X-Tenant-ID"), c.Request().Header.Get("X-User-ID") }
func zeroUUID() string { return "00000000-0000-0000-0000-000000000000" }
func orEmpty(m map[string]interface{}) map[string]interface{} { if m == nil { return map[string]interface{}{} }; return m }
func lower(in []string) []string { out := make([]string, len(in)); for i, v := range in { out[i] = strings.ToLower(strings.TrimSpace(v)) }; return out }
func toCSV(v interface{}) string {
	switch x := v.(type) {
	case string: return x
	case float64, bool: return fmt.Sprintf("%v", x)
	case nil: return ""
	case map[string]interface{}, []interface{}: b, _ := json.Marshal(x); return string(b)
	default: return fmt.Sprintf("%v", x)
	}
}
func joinErrs(errs []validation.Error) string { parts := make([]string, 0, len(errs)); for _, e := range errs { parts = append(parts, e.Field+": "+e.Message) }; return strings.Join(parts, "; ") }
func attachJSON(target map[string]interface{}, pairs ...interface{}) map[string]interface{} {
	for i := 0; i < len(pairs); i += 2 { var decoded map[string]interface{}; raw, _ := pairs[i+1].([]byte); if len(raw) > 0 { _ = json.Unmarshal(raw, &decoded) }; target[pairs[i].(string)] = decoded }
	return target
}
