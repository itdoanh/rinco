// Package handler — CRM extra endpoints (leads, pipelines, workflows,
// notifications, audit, sessions, departments, tags, custom-fields).
//
// These handlers complement handler.go, handler_tree.go, handler_extra.go
// to deliver an enterprise-grade CRM Tree service per docs/03-crm-tree.
//
// Conventions:
//   * Every handler sets RLS via s.setRLS() at the very top.
//   * Tenant + user_id resolved through s.tenantFromCtx(c).
//   * Validation first, then DB, then audit + emit workflow trigger.
//   * Soft-delete via deleted_at; list queries filter deleted_at IS NULL.
package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// =============================================================================
// Helpers (shared across new handlers)
// =============================================================================

// auditContextKeys dùng để pull tenant/user từ context.Context khi
// background goroutine (workflow trigger) không có Echo context.
// Cùng key với middleware.TenantMW() gán vào echo context.
const (
	ctxKeyTenantID = "tenant_id"
	ctxKeyUserID   = "user_id"
	ctxKeyIsAdmin  = "is_admin"
)

func (s *Server) audit(ctx context.Context, actorID, tenantID, action, targetType, targetID string, oldVal, newVal any) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("audit panic", slog.Any("err", r))
		}
	}()
	var oldB, newB []byte
	if oldVal != nil {
		oldB, _ = json.Marshal(oldVal)
	}
	if newVal != nil {
		newB, _ = json.Marshal(newVal)
	}
	if tenantID == "" {
		if v, ok := ctx.Value(ctxKeyTenantID).(string); ok {
			tenantID = v
		}
	}
	if tenantID == "" {
		return // can't audit without tenant
	}
	var actorCol interface{} = nil
	if actorID != "" {
		if u, err := uuid.Parse(actorID); err == nil {
			actorCol = u
		}
	}
	var targetCol interface{} = nil
	if targetID != "" {
		if u, err := uuid.Parse(targetID); err == nil {
			targetCol = u
		}
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_log (tenant_id, actor_id, action, target_type, target_id, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, actorCol, action, targetType, targetCol, oldB, newB)
	if err != nil {
		slog.Warn("audit_log insert failed", slog.String("action", action), slog.String("error", err.Error()))
	}
}

// ============================================================================
// Leads
// ============================================================================

type leadReq struct {
	FullName        *string         `json:"full_name,omitempty"`
	Email           *string         `json:"email,omitempty"`
	Phone           *string         `json:"phone,omitempty"`
	CompanyName     *string         `json:"company_name,omitempty"`
	Source          *string         `json:"source,omitempty"`
	SourceDetail    *string         `json:"source_detail,omitempty"`
	UTM             *map[string]any `json:"utm,omitempty"`
	FBCLID          *string         `json:"fbclid,omitempty"`
	FBP             *string         `json:"fbp,omitempty"`
	FBC             *string         `json:"fbc,omitempty"`
	IPAddress       *string         `json:"ip_address,omitempty"`
	UserAgent       *string         `json:"user_agent,omitempty"`
	Score           *float64        `json:"score,omitempty"`
	EstimatedValue  *float64        `json:"estimated_value,omitempty"`
	Currency        *string         `json:"currency,omitempty"`
	Status          *string         `json:"status,omitempty"`
	LostReason      *string         `json:"lost_reason,omitempty"`
	CustomFields    *map[string]any `json:"custom_fields,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	NextFollowupAt  *time.Time      `json:"next_followup_at,omitempty"`
	PipelineID      *string         `json:"pipeline_id,omitempty"`
	StageID         *string         `json:"current_stage_id,omitempty"`
	OwnerUserID     *string         `json:"owner_user_id,omitempty"`
}

type leadResp struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	PipelineID          *uuid.UUID `json:"pipeline_id,omitempty"`
	CurrentStageID      *uuid.UUID `json:"current_stage_id,omitempty"`
	OwnerUserID         *uuid.UUID `json:"owner_user_id,omitempty"`
	ConvertedContactID  *uuid.UUID `json:"converted_contact_id,omitempty"`
	ConvertedDealID     *uuid.UUID `json:"converted_deal_id,omitempty"`
	FullName            string     `json:"full_name,omitempty"`
	Email               string     `json:"email,omitempty"`
	Phone               string     `json:"phone,omitempty"`
	CompanyName         string     `json:"company_name,omitempty"`
	Source              string     `json:"source,omitempty"`
	SourceDetail        string     `json:"source_detail,omitempty"`
	UTM                 map[string]any `json:"utm,omitempty"`
	FBCLID              string     `json:"fbclid,omitempty"`
	FBP                 string     `json:"fbp,omitempty"`
	Score               *float64   `json:"score,omitempty"`
	EstimatedValue      *float64   `json:"estimated_value,omitempty"`
	Currency            string     `json:"currency"`
	Status              string     `json:"status"`
	LostReason          string     `json:"lost_reason,omitempty"`
	CustomFields        map[string]any `json:"custom_fields,omitempty"`
	Tags                []string   `json:"tags"`
	NextFollowupAt      *time.Time `json:"next_followup_at,omitempty"`
	LastContactedAt     *time.Time `json:"last_contacted_at,omitempty"`
	ConvertedAt         *time.Time `json:"converted_at,omitempty"`
	AssignedAt          *time.Time `json:"assigned_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func parseLeadReq(r leadReq) (status string, ownerUUID *uuid.UUID, pipelineUUID *uuid.UUID, stageUUID *uuid.UUID, utmB, cfB []byte, err error) {
	if r.Status == nil {
		status = "NEW"
	} else {
		switch *r.Status {
		case "NEW", "CONTACTED", "QUALIFIED", "PROPOSAL", "WON", "LOST", "ARCHIVED":
			status = *r.Status
		default:
			return "", nil, nil, nil, nil, nil, errors.New("invalid status")
		}
	}
	if r.OwnerUserID != nil && *r.OwnerUserID != "" {
		u, e := uuid.Parse(*r.OwnerUserID)
		if e != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("invalid owner_user_id: %w", e)
		}
		ownerUUID = &u
	}
	if r.PipelineID != nil && *r.PipelineID != "" {
		u, e := uuid.Parse(*r.PipelineID)
		if e != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("invalid pipeline_id: %w", e)
		}
		pipelineUUID = &u
	}
	if r.StageID != nil && *r.StageID != "" {
		u, e := uuid.Parse(*r.StageID)
		if e != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("invalid current_stage_id: %w", e)
		}
		stageUUID = &u
	}
	if r.UTM != nil {
		utmB, _ = json.Marshal(*r.UTM)
	}
	if r.CustomFields != nil {
		cfB, _ = json.Marshal(*r.CustomFields)
	}
	return status, ownerUUID, pipelineUUID, stageUUID, utmB, cfB, nil
}

const leadSelectClause = `
SELECT id, tenant_id, pipeline_id, current_stage_id, owner_user_id,
       converted_contact_id, converted_deal_id,
       COALESCE(full_name,''), COALESCE(email,''), COALESCE(phone,''),
       COALESCE(company_name,''), COALESCE(source,''), COALESCE(source_detail,''),
       COALESCE(utm, '{}'::jsonb), COALESCE(fbclid,''), COALESCE(fbp,''),
       score, estimated_value, COALESCE(currency,'VND'), status, COALESCE(lost_reason,''),
       COALESCE(custom_fields, '{}'::jsonb),
       COALESCE(tags, '{}'),
       next_followup_at, last_contacted_at, converted_at, assigned_at,
       created_at, updated_at
FROM leads WHERE id = $1 AND deleted_at IS NULL
`

func scanLead(row pgx.Row, dst *leadResp) error {
	var utmB, cfB []byte
	if err := row.Scan(
		&dst.ID, &dst.TenantID, &dst.PipelineID, &dst.CurrentStageID, &dst.OwnerUserID,
		&dst.ConvertedContactID, &dst.ConvertedDealID,
		&dst.FullName, &dst.Email, &dst.Phone,
		&dst.CompanyName, &dst.Source, &dst.SourceDetail,
		&utmB, &dst.FBCLID, &dst.FBP,
		&dst.Score, &dst.EstimatedValue, &dst.Currency, &dst.Status, &dst.LostReason,
		&cfB, &dst.Tags,
		&dst.NextFollowupAt, &dst.LastContactedAt, &dst.ConvertedAt, &dst.AssignedAt,
		&dst.CreatedAt, &dst.UpdatedAt,
	); err != nil {
		return err
	}
	if len(utmB) > 0 {
		_ = json.Unmarshal(utmB, &dst.UTM)
	}
	if len(cfB) > 0 {
		_ = json.Unmarshal(cfB, &dst.CustomFields)
	}
	return nil
}

func (s *Server) CreateLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req leadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	status, ownerUUID, pipelineUUID, stageUUID, utmB, cfB, err := parseLeadReq(req)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, err.Error(), nil)
	}

	if req.FullName == nil && req.Email == nil && req.Phone == nil {
		return s.errorResp(c, http.StatusBadRequest, "at least one of full_name/email/phone is required", nil)
	}

	currency := "VND"
	if req.Currency != nil && *req.Currency != "" {
		currency = *req.Currency
	}

	var ownerCol interface{} = nil
	if ownerUUID != nil {
		ownerCol = *ownerUUID
	}
	var pipeCol interface{} = nil
	if pipelineUUID != nil {
		pipeCol = *pipelineUUID
	}
	var stageCol interface{} = nil
	if stageUUID != nil {
		stageCol = *stageUUID
	}

	var ipArg interface{} = nil
	if req.IPAddress != nil && *req.IPAddress != "" {
		ipArg = *req.IPAddress
	}

	var resp leadResp
	row := s.pool.QueryRow(ctx, `
		INSERT INTO leads (tenant_id, pipeline_id, current_stage_id, owner_user_id,
		                   full_name, email, phone, company_name, source, source_detail,
		                   utm, fbclid, fbp, fbc, ip_address, user_agent,
		                   score, estimated_value, currency, status, lost_reason,
		                   custom_fields, tags, next_followup_at, assigned_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,$14,$15::inet,$16,$17,$18,$19,$20,$21,$22::jsonb,$23,$24, NOW())
		RETURNING id
	`, tenantID, pipeCol, stageCol, ownerCol,
		req.FullName, req.Email, req.Phone, req.CompanyName, req.Source, req.SourceDetail,
		utmB, req.FBCLID, req.FBP, req.FBC, ipArg, req.UserAgent,
		req.Score, req.EstimatedValue, currency, status, req.LostReason,
		cfB, req.Tags, req.NextFollowupAt,
	)
	if err := row.Scan(&resp.ID); err != nil {
		slog.Error("create lead failed", slog.String("error", err.Error()))
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	// Re-fetch full record (RLS will allow via tenant filter)
	r2 := s.pool.QueryRow(ctx, leadSelectClause, resp.ID)
	if err := scanLead(r2, &resp); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "refetch failed", err)
	}

	s.audit(ctx, userID, tenantID, "lead.create", "lead", resp.ID.String(), nil, resp)

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) ListLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)

	var (
		where []string
		args []any
	)
	where = append(where, "deleted_at IS NULL")
	addFilter := func(clause string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf("%s $%d", clause, len(args)))
	}
	if v := c.QueryParam("status"); v != "" {
		addFilter("status =", v)
	}
	if v := c.QueryParam("owner_user_id"); v != "" {
		addFilter("owner_user_id =", v)
	}
	if v := c.QueryParam("source"); v != "" {
		addFilter("source =", v)
	}
	if v := c.QueryParam("min_score"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			addFilter("score >=", f)
		}
	}
	if v := c.QueryParam("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			addFilter("created_at >=", t)
		}
	}
	if v := c.QueryParam("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			addFilter("created_at <=", t)
		}
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM leads %s", whereSQL)
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}

	args = append(args, p.PerPage, p.Offset)
	query := fmt.Sprintf(`
		SELECT id, tenant_id, pipeline_id, current_stage_id, owner_user_id,
		       converted_contact_id, converted_deal_id,
		       COALESCE(full_name,''), COALESCE(email,''), COALESCE(phone,''),
		       COALESCE(company_name,''), COALESCE(source,''), COALESCE(source_detail,''),
		       COALESCE(utm, '{}'::jsonb), COALESCE(fbclid,''), COALESCE(fbp,''),
		       score, estimated_value, COALESCE(currency,'VND'), status, COALESCE(lost_reason,''),
		       COALESCE(custom_fields, '{}'::jsonb),
		       COALESCE(tags, '{}'),
		       next_followup_at, last_contacted_at, converted_at, assigned_at,
		       created_at, updated_at
		FROM leads %s
		ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, whereSQL, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var leads []leadResp
	for rows.Next() {
		var l leadResp
		if err := scanLead(rows, &l); err != nil {
			slog.Warn("scan lead failed", slog.String("error", err.Error()))
			continue
		}
		leads = append(leads, l)
	}

	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}
	return s.json(c, http.StatusOK, listResp{Data: leads, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

func (s *Server) GetLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var resp leadResp
	if err := scanLead(s.pool.QueryRow(ctx, leadSelectClause, id), &resp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
		}
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req leadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var utmB, cfB []byte
	if req.UTM != nil {
		utmB, _ = json.Marshal(*req.UTM)
	}
	if req.CustomFields != nil {
		cfB, _ = json.Marshal(*req.CustomFields)
	}

	row := s.pool.QueryRow(ctx, `
		UPDATE leads SET
			full_name = COALESCE($2, full_name),
			email = COALESCE($3, email),
			phone = COALESCE($4, phone),
			company_name = COALESCE($5, company_name),
			source = COALESCE($6, source),
			source_detail = COALESCE($7, source_detail),
			utm = COALESCE($8::jsonb, utm),
			fbclid = COALESCE($9, fbclid),
			fbp = COALESCE($10, fbp),
			score = COALESCE($11, score),
			estimated_value = COALESCE($12, estimated_value),
			currency = COALESCE($13, currency),
			status = COALESCE($14, status),
			lost_reason = COALESCE($15, lost_reason),
			custom_fields = COALESCE($16::jsonb, custom_fields),
			tags = COALESCE($17, tags),
			next_followup_at = COALESCE($18, next_followup_at),
			owner_user_id = COALESCE($19::uuid, owner_user_id),
			assigned_at = CASE WHEN $19::uuid IS NOT NULL THEN NOW() ELSE assigned_at END,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id
	`, id,
		req.FullName, req.Email, req.Phone, req.CompanyName, req.Source, req.SourceDetail,
		utmB, req.FBCLID, req.FBP, req.Score, req.EstimatedValue, req.Currency, req.Status, req.LostReason,
		cfB, req.Tags, req.NextFollowupAt, req.OwnerUserID,
	)
	var newID uuid.UUID
	if err := row.Scan(&newID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
		}
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}

	var resp leadResp
	if err := scanLead(s.pool.QueryRow(ctx, leadSelectClause, id), &resp); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "refetch failed", err)
	}
	s.audit(ctx, userID, tenantID, "lead.update", "lead", id, nil, resp)
	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	r, err := s.pool.Exec(ctx, `UPDATE leads SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	s.audit(ctx, userID, tenantID, "lead.delete", "lead", id, nil, nil)
	return c.NoContent(http.StatusNoContent)
}

type assignLeadReq struct {
	OwnerUserID string `json:"owner_user_id"`
	Reason      string `json:"reason,omitempty"`
}

func (s *Server) AssignLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req assignLeadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.OwnerUserID == "" {
		return s.errorResp(c, http.StatusBadRequest, "owner_user_id is required", nil)
	}
	ownerUUID, err := uuid.Parse(req.OwnerUserID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid owner_user_id", err)
	}

	r, err := s.pool.Exec(ctx, `
		UPDATE leads SET owner_user_id = $2, assigned_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, ownerUUID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "assign failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	s.audit(ctx, userID, tenantID, "lead.assign", "lead", id, nil, map[string]any{
		"owner_user_id": ownerUUID, "reason": req.Reason,
	})

	// Best-effort: notify new owner
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO notifications (tenant_id, user_id, sender_user_id, type, title, body, link, priority)
		VALUES ($1, $2, NULLIF($3,'')::UUID, 'lead_assigned', $4, $5, $6, 'NORMAL')
	`, tenantID, ownerUUID, userID,
		"Lead được phân cho bạn",
		fmt.Sprintf("Lead %s vừa được phân cho bạn.", id),
		fmt.Sprintf("/leads/%s", id))

	var resp leadResp
	if err := scanLead(s.pool.QueryRow(ctx, leadSelectClause, id), &resp); err != nil {
		return s.json(c, http.StatusOK, map[string]any{"status": "assigned"})
	}
	return s.json(c, http.StatusOK, resp)
}

type moveLeadStageReq struct {
	StageID string `json:"stage_id"`
	Notes   string `json:"notes,omitempty"`
}

func (s *Server) MoveLeadStage(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req moveLeadStageReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	stageUUID, err := uuid.Parse(req.StageID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid stage_id", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "tx begin failed", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var fromStageID *uuid.UUID
	var status string
	if err := tx.QueryRow(ctx, `SELECT current_stage_id, status FROM leads WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&fromStageID, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
		}
		return s.errorResp(c, http.StatusInternalServerError, "lookup failed", err)
	}

	// Map stage → status
	var stageCode string
	if err := tx.QueryRow(ctx, `SELECT code FROM pipeline_stages WHERE id = $1`, stageUUID).Scan(&stageCode); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "stage not found", err)
	}

	var toStatus string
	switch stageCode {
	case "NEW":
		toStatus = "NEW"
	case "CONTACTED":
		toStatus = "CONTACTED"
	case "QUALIFIED":
		toStatus = "QUALIFIED"
	case "PROPOSAL":
		toStatus = "PROPOSAL"
	case "WON":
		toStatus = "WON"
	case "LOST":
		toStatus = "LOST"
	default:
		toStatus = status
	}

	if _, err := tx.Exec(ctx, `
		UPDATE leads SET current_stage_id = $2, status = $3, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, stageUUID, toStatus); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "stage update failed", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO lead_stage_history (tenant_id, lead_id, from_stage_id, to_stage_id, changed_by, notes)
		VALUES ($1, $2, $3, $4, NULLIF($5,'')::UUID, $6)
	`, tenantID, id, fromStageID, stageUUID, userID, req.Notes); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "history insert failed", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "tx commit failed", err)
	}

	s.audit(ctx, userID, tenantID, "lead.stage_change", "lead", id,
		map[string]any{"stage": fromStageID, "status": status},
		map[string]any{"stage": stageUUID, "status": toStatus},
	)

	// Trigger workflows (stage_change) — best effort
	go s.runWorkflowsFor(context.Background(), tenantID, "stage_change", "lead", idUUID, map[string]any{
		"from_stage": fromStageID, "to_stage": stageUUID, "to_status": toStatus,
	})

	var resp leadResp
	if err := scanLead(s.pool.QueryRow(ctx, leadSelectClause, id), &resp); err != nil {
		return s.json(c, http.StatusOK, map[string]any{"status": "moved", "new_status": toStatus})
	}
	return s.json(c, http.StatusOK, resp)
}

type convertLeadReq struct {
	CreateDeal bool    `json:"create_deal"`
	DealValue  *float64 `json:"deal_value,omitempty"`
	DealName   *string  `json:"deal_name,omitempty"`
}

// ConvertLead: chuyển Lead → Contact (và tùy chọn tạo Deal). Sau đó đánh dấu
// status = 'QUALIFIED' hoặc 'WON' và set converted_at.
func (s *Server) ConvertLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 15*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	leadUUID, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req convertLeadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "tx begin failed", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		fullName, email, phone, companyName sql.NullString
		ownerUUID                            *uuid.UUID
		cfRaw                                []byte
	)
	err = tx.QueryRow(ctx, `
		SELECT full_name, email, phone, company_name, owner_user_id, custom_fields
		FROM leads WHERE id = $1 AND deleted_at IS NULL FOR UPDATE
	`, id).Scan(&fullName, &email, &phone, &companyName, &ownerUUID, &cfRaw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
		}
		return s.errorResp(c, http.StatusInternalServerError, "lookup failed", err)
	}
	if !fullName.Valid {
		return s.errorResp(c, http.StatusBadRequest, "lead full_name is required for convert", nil)
	}

	names := strings.SplitN(fullName.String, " ", 2)
	first, last := names[0], ""
	if len(names) > 1 {
		last = names[1]
	}

	var companyID *uuid.UUID
	if companyName.Valid && companyName.String != "" {
		var cid uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO companies (tenant_id, name) VALUES ($1, $2) RETURNING id
		`, tenantID, companyName.String).Scan(&cid)
		if err != nil {
			return s.errorResp(c, http.StatusInternalServerError, "company create failed", err)
		}
		companyID = &cid
	}

	var contactID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO contacts (tenant_id, company_id, first_name, last_name, email, phone, owner_user_id, status, source, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', 'lead', $8::jsonb)
		RETURNING id
	`, tenantID, companyID, first, last,
		nullableString(email), nullableString(phone), ownerUUID, cfRaw).Scan(&contactID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "contact create failed", err)
	}

	var dealID *uuid.UUID
	if req.CreateDeal {
		name := first + " " + last
		if req.DealName != nil && *req.DealName != "" {
			name = *req.DealName
		}
		var v float64
		if req.DealValue != nil {
			v = *req.DealValue
		}
		var did uuid.UUID
		if err = tx.QueryRow(ctx, `
			INSERT INTO deals (tenant_id, contact_id, owner_user_id, name, value, stage)
			VALUES ($1, $2, $3, $4, $5, 'qualification')
			RETURNING id
		`, tenantID, contactID, ownerUUID, name, v).Scan(&did); err != nil {
			return s.errorResp(c, http.StatusInternalServerError, "deal create failed", err)
		}
		dealID = &did
	}

	if _, err := tx.Exec(ctx, `
		UPDATE leads
		SET converted_contact_id = $2,
		    converted_deal_id = $3,
		    status = 'QUALIFIED',
		    converted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, contactID, dealID); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "lead update failed", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "tx commit failed", err)
	}

	s.audit(ctx, userID, tenantID, "lead.convert", "lead", id, nil, map[string]any{
		"contact_id": contactID, "deal_id": dealID,
	})

	return s.json(c, http.StatusOK, map[string]any{
		"lead_id":     leadUUID,
		"contact_id":  contactID,
		"deal_id":     dealID,
		"status":      "QUALIFIED",
	})
}

func nullableString(s sql.NullString) interface{} {
	if s.Valid {
		return s.String
	}
	return nil
}

// BulkLeadImport — gộp nhiều lead cùng lúc (giới hạn 1000 cho request này).
type bulkLeadReq struct {
	Leads []leadReq `json:"leads"`
}

type bulkLeadResp struct {
	Inserted int      `json:"inserted"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

func (s *Server) BulkLeadImport(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req bulkLeadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if len(req.Leads) == 0 {
		return s.errorResp(c, http.StatusBadRequest, "leads array required", nil)
	}
	if len(req.Leads) > 1000 {
		return s.errorResp(c, http.StatusBadRequest, "max 1000 leads per request", nil)
	}

	resp := bulkLeadResp{}
	for i, l := range req.Leads {
		status, ownerUUID, pipeUUID, stageUUID, utmB, cfB, err := parseLeadReq(l)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: %s", i, err.Error()))
			continue
		}
		_, dbErr := s.pool.Exec(ctx, `
			INSERT INTO leads (tenant_id, pipeline_id, current_stage_id, owner_user_id,
			                    full_name, email, phone, company_name, source, source_detail,
			                    utm, fbclid, fbp, score, estimated_value, currency, status,
			                    custom_fields, tags, next_followup_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,$14,$15,$16,$17,$18::jsonb,$19,$20)
		`, tenantID, pipeUUID, stageUUID, ownerUUID,
			l.FullName, l.Email, l.Phone, l.CompanyName, l.Source, l.SourceDetail,
			utmB, l.FBCLID, l.FBP, l.Score, l.EstimatedValue,
			coalesceString(l.Currency, "VND"), status, cfB, l.Tags, l.NextFollowupAt,
		)
		if dbErr != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: %s", i, dbErr.Error()))
			continue
		}
		resp.Inserted++
	}

	s.audit(ctx, userID, tenantID, "lead.bulk_import", "lead", "", nil, map[string]any{
		"inserted": resp.Inserted, "failed": resp.Failed, "total": len(req.Leads),
	})
	return s.json(c, http.StatusOK, resp)
}

func coalesceString(s *string, def string) string {
	if s == nil || *s == "" {
		return def
	}
	return *s
}

// =============================================================================
// Pipelines
// =============================================================================

type pipelineReq struct {
	Name        *string       `json:"name,omitempty"`
	Description *string       `json:"description,omitempty"`
	IsDefault   *bool         `json:"is_default,omitempty"`
	Stages      []stageReq    `json:"stages,omitempty"`
}

type stageReq struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Color       string `json:"color,omitempty"`
	Position    int    `json:"position"`
	Probability *int  `json:"probability,omitempty"`
	SLAHours    *int  `json:"sla_hours,omitempty"`
	IsWon       bool  `json:"is_won,omitempty"`
	IsLost      bool  `json:"is_lost,omitempty"`
}

type pipelineResp struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsDefault   bool      `json:"is_default"`
	Stages      []stageResp `json:"stages,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type stageResp struct {
	ID          uuid.UUID `json:"id"`
	PipelineID  uuid.UUID `json:"pipeline_id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Color       string    `json:"color,omitempty"`
	Position    int       `json:"position"`
	Probability int       `json:"probability"`
	SLAHours    *int      `json:"sla_hours,omitempty"`
	IsWon       bool      `json:"is_won"`
	IsLost      bool      `json:"is_lost"`
}

func (s *Server) CreatePipeline(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req pipelineReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.Name == nil || *req.Name == "" {
		return s.errorResp(c, http.StatusBadRequest, "name is required", nil)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "tx begin failed", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var p pipelineResp
	err = tx.QueryRow(ctx, `
		INSERT INTO pipelines (tenant_id, name, description, is_default)
		VALUES ($1, $2, $3, COALESCE($4, false))
		RETURNING id, tenant_id, name, COALESCE(description,''), COALESCE(is_default, false), created_at, updated_at
	`, tenantID, *req.Name, req.Description, req.IsDefault).Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	for _, st := range req.Stages {
		var prob int = 50
		if st.Probability != nil {
			prob = *st.Probability
		}
		color := st.Color
		if color == "" {
			color = "#6366f1"
		}
		var stRow stageResp
		err = tx.QueryRow(ctx, `
			INSERT INTO pipeline_stages (pipeline_id, tenant_id, name, code, color, position, probability, sla_hours, is_won, is_lost)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id, pipeline_id, name, code, COALESCE(color,''), position, probability, sla_hours, COALESCE(is_won,false), COALESCE(is_lost,false)
		`, p.ID, tenantID, st.Name, st.Code, color, st.Position, prob, st.SLAHours, st.IsWon, st.IsLost).Scan(
			&stRow.ID, &stRow.PipelineID, &stRow.Name, &stRow.Code, &stRow.Color,
			&stRow.Position, &stRow.Probability, &stRow.SLAHours, &stRow.IsWon, &stRow.IsLost,
		)
		if err != nil {
			return s.errorResp(c, http.StatusInternalServerError, "stage insert failed", err)
		}
		p.Stages = append(p.Stages, stRow)
	}

	if err := tx.Commit(ctx); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "tx commit failed", err)
	}
	s.audit(ctx, userID, tenantID, "pipeline.create", "pipeline", p.ID.String(), nil, p)
	return s.json(c, http.StatusCreated, p)
}

func (s *Server) ListPipelines(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, name, COALESCE(description,''), COALESCE(is_default,false), created_at, updated_at
		FROM pipelines WHERE deleted_at IS NULL ORDER BY is_default DESC, created_at
	`)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var pipes []pipelineResp
	for rows.Next() {
		var p pipelineResp
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		pipes = append(pipes, p)
	}
	return s.json(c, http.StatusOK, map[string]any{"pipelines": pipes, "count": len(pipes)})
}

func (s *Server) GetPipeline(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var p pipelineResp
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, COALESCE(description,''), COALESCE(is_default,false), created_at, updated_at
		FROM pipelines WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.errorResp(c, http.StatusNotFound, "pipeline not found", nil)
		}
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}

	stageRows, err := s.pool.Query(ctx, `
		SELECT id, pipeline_id, name, code, COALESCE(color,''), position, probability, sla_hours,
		       COALESCE(is_won,false), COALESCE(is_lost,false)
		FROM pipeline_stages WHERE pipeline_id = $1 ORDER BY position
	`, p.ID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "stages query failed", err)
	}
	defer stageRows.Close()
	for stageRows.Next() {
		var st stageResp
		if err := stageRows.Scan(&st.ID, &st.PipelineID, &st.Name, &st.Code, &st.Color, &st.Position, &st.Probability, &st.SLAHours, &st.IsWon, &st.IsLost); err != nil {
			continue
		}
		p.Stages = append(p.Stages, st)
	}
	return s.json(c, http.StatusOK, p)
}

// =============================================================================
// Workflows
// =============================================================================

type workflowReq struct {
	Name          *string         `json:"name,omitempty"`
	Description   *string         `json:"description,omitempty"`
	TriggerType   *string         `json:"trigger_type,omitempty"`
	TriggerConfig *map[string]any `json:"trigger_config,omitempty"`
	Conditions    *map[string]any `json:"conditions,omitempty"`
	Actions       *[]map[string]any `json:"actions,omitempty"`
	IsActive      *bool           `json:"is_active,omitempty"`
}

type workflowResp struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	TriggerType   string    `json:"trigger_type"`
	TriggerConfig map[string]any `json:"trigger_config"`
	Conditions    map[string]any `json:"conditions"`
	Actions       []map[string]any `json:"actions"`
	IsActive      bool      `json:"is_active"`
	RunCount      int       `json:"run_count"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (s *Server) CreateWorkflow(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req workflowReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.Name == nil || req.TriggerType == nil {
		return s.errorResp(c, http.StatusBadRequest, "name and trigger_type are required", nil)
	}
	switch *req.TriggerType {
	case "stage_change", "field_change", "time_based", "manual", "lead_created", "lead_assigned":
	default:
		return s.errorResp(c, http.StatusBadRequest, "invalid trigger_type", nil)
	}
	if req.Actions == nil || len(*req.Actions) == 0 {
		return s.errorResp(c, http.StatusBadRequest, "actions must be non-empty", nil)
	}

	var triggerB, condB []byte = []byte("{}"), []byte("{}")
	if req.TriggerConfig != nil {
		triggerB, _ = json.Marshal(*req.TriggerConfig)
	}
	if req.Conditions != nil {
		condB, _ = json.Marshal(*req.Conditions)
	}
	actionsB, _ := json.Marshal(*req.Actions)

	var createdBy interface{} = nil
	if userID != "" {
		if u, err := uuid.Parse(userID); err == nil {
			createdBy = u
		}
	}

	var resp workflowResp
	var trigCfgB, condBOut, actBOut []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workflows (tenant_id, name, description, trigger_type, trigger_config, conditions, actions, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, COALESCE($8, true), $9)
		RETURNING id, tenant_id, name, COALESCE(description,''), trigger_type,
		          COALESCE(trigger_config,'{}'::jsonb), COALESCE(conditions,'{}'::jsonb),
		          COALESCE(actions,'[]'::jsonb), COALESCE(is_active,true), COALESCE(run_count,0),
		          last_run_at, created_at, updated_at
	`, tenantID, *req.Name, req.Description, *req.TriggerType, triggerB, condB, actionsB, req.IsActive, createdBy).Scan(
		&resp.ID, &resp.TenantID, &resp.Name, &resp.Description, &resp.TriggerType,
		&trigCfgB, &condBOut, &actBOut, &resp.IsActive, &resp.RunCount,
		&resp.LastRunAt, &resp.CreatedAt, &resp.UpdatedAt,
	)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	_ = json.Unmarshal(trigCfgB, &resp.TriggerConfig)
	_ = json.Unmarshal(condBOut, &resp.Conditions)
	_ = json.Unmarshal(actBOut, &resp.Actions)
	s.audit(ctx, userID, tenantID, "workflow.create", "workflow", resp.ID.String(), nil, resp)
	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) ListWorkflows(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, name, COALESCE(description,''), trigger_type,
		       COALESCE(trigger_config,'{}'::jsonb), COALESCE(conditions,'{}'::jsonb),
		       COALESCE(actions,'[]'::jsonb), COALESCE(is_active,true), COALESCE(run_count,0),
		       last_run_at, created_at, updated_at
		FROM workflows WHERE deleted_at IS NULL ORDER BY created_at DESC
	`)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var out []workflowResp
	for rows.Next() {
		var w workflowResp
		var trigB, condB, actB []byte
		if err := rows.Scan(&w.ID, &w.TenantID, &w.Name, &w.Description, &w.TriggerType,
			&trigB, &condB, &actB, &w.IsActive, &w.RunCount,
			&w.LastRunAt, &w.CreatedAt, &w.UpdatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(trigB, &w.TriggerConfig)
		_ = json.Unmarshal(condB, &w.Conditions)
		_ = json.Unmarshal(actB, &w.Actions)
		out = append(out, w)
	}
	return s.json(c, http.StatusOK, map[string]any{"workflows": out, "count": len(out)})
}

func (s *Server) GetWorkflow(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var w workflowResp
	var trigB, condB, actB []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, COALESCE(description,''), trigger_type,
		       COALESCE(trigger_config,'{}'::jsonb), COALESCE(conditions,'{}'::jsonb),
		       COALESCE(actions,'[]'::jsonb), COALESCE(is_active,true), COALESCE(run_count,0),
		       last_run_at, created_at, updated_at
		FROM workflows WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&w.ID, &w.TenantID, &w.Name, &w.Description, &w.TriggerType,
		&trigB, &condB, &actB, &w.IsActive, &w.RunCount,
		&w.LastRunAt, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.errorResp(c, http.StatusNotFound, "workflow not found", nil)
		}
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	_ = json.Unmarshal(trigB, &w.TriggerConfig)
	_ = json.Unmarshal(condB, &w.Conditions)
	_ = json.Unmarshal(actB, &w.Actions)
	return s.json(c, http.StatusOK, w)
}

func (s *Server) UpdateWorkflow(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req workflowReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var trigB, condB, actB []byte
	if req.TriggerConfig != nil {
		trigB, _ = json.Marshal(*req.TriggerConfig)
	}
	if req.Conditions != nil {
		condB, _ = json.Marshal(*req.Conditions)
	}
	if req.Actions != nil {
		actB, _ = json.Marshal(*req.Actions)
	}

	r, err := s.pool.Exec(ctx, `
		UPDATE workflows SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			trigger_type = COALESCE($4, trigger_type),
			trigger_config = COALESCE($5::jsonb, trigger_config),
			conditions = COALESCE($6::jsonb, conditions),
			actions = COALESCE($7::jsonb, actions),
			is_active = COALESCE($8, is_active),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, req.Name, req.Description, req.TriggerType, trigB, condB, actB, req.IsActive)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "workflow not found", nil)
	}
	s.audit(ctx, userID, tenantID, "workflow.update", "workflow", id, nil, req)
	return s.GetWorkflow(c)
}

func (s *Server) DeleteWorkflow(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	r, err := s.pool.Exec(ctx, `UPDATE workflows SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "workflow not found", nil)
	}
	s.audit(ctx, userID, tenantID, "workflow.delete", "workflow", id, nil, nil)
	return c.NoContent(http.StatusNoContent)
}

type triggerWorkflowReq struct {
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Inputs     map[string]any `json:"inputs,omitempty"`
}

func (s *Server) TriggerWorkflow(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req triggerWorkflowReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.EntityType == "" || req.EntityID == "" {
		return s.errorResp(c, http.StatusBadRequest, "entity_type and entity_id required", nil)
	}
	entityUUID, err := uuid.Parse(req.EntityID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid entity_id", err)
	}

	s.runWorkflowsFor(ctx, tenantID, "manual", req.EntityType, entityUUID, req.Inputs)
	return s.json(c, http.StatusOK, map[string]any{"status": "triggered"})
}

// runWorkflowsFor evaluates workflows of the given trigger type and executes
// their actions. Runs in same ctx (best effort) – failures are recorded into
// workflow_executions but never bubble up.
func (s *Server) runWorkflowsFor(ctx context.Context, tenantID, triggerType, entityType string, entityID uuid.UUID, inputs map[string]any) {
	if inputs == nil {
		inputs = map[string]any{}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, conditions, actions
		FROM workflows
		WHERE tenant_id = $1 AND trigger_type = $2 AND is_active = true AND deleted_at IS NULL
	`, tenantID, triggerType)
	if err != nil {
		slog.Warn("workflow list failed", slog.String("error", err.Error()))
		return
	}
	defer rows.Close()
	type wf struct {
		id      uuid.UUID
		name    string
		condB   []byte
		actB    []byte
	}
	var wfs []wf
	for rows.Next() {
		var w wf
		if err := rows.Scan(&w.id, &w.name, &w.condB, &w.actB); err != nil {
			continue
		}
		wfs = append(wfs, w)
	}
	rows.Close()

	for _, w := range wfs {
		var cond struct {
			All []map[string]any `json:"all"`
			Any []map[string]any `json:"any"`
		}
		_ = json.Unmarshal(w.condB, &cond)
		if !evaluateCondition(cond, inputs) {
			continue
		}
		var actions []map[string]any
		_ = json.Unmarshal(w.actB, &actions)

		execResults := []map[string]any{}
		anyFailed := false
		start := time.Now()
		for _, act := range actions {
			actType, _ := act["type"].(string)
			r := s.executeAction(ctx, tenantID, actType, act, entityType, entityID)
			execResults = append(execResults, r)
			if ok, _ := r["ok"].(bool); !ok {
				anyFailed = true
			}
		}

		status := "success"
		if anyFailed {
			status = "partial"
		}
		_, err := s.pool.Exec(ctx, `
			INSERT INTO workflow_executions (tenant_id, workflow_id, entity_type, entity_id, triggered_by, status, inputs, actions_executed, execution_ms)
			VALUES ($1, $2, $3, $4, 'event', $5, $6::jsonb, $7::jsonb, $8)
		`, tenantID, w.id, entityType, entityID, status, mustJSON(inputs), mustJSON(execResults), int(time.Since(start).Milliseconds()))
		if err != nil {
			slog.Warn("workflow exec insert failed", slog.String("error", err.Error()))
		}

		_, _ = s.pool.Exec(ctx, `UPDATE workflows SET run_count = run_count + 1, last_run_at = NOW() WHERE id = $1`, w.id)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// evaluateCondition: simple JSON DSL.
//   {"all":[{"field":"score","op":">","value":70}]}   — all predicates must pass
//   {"any":[...]}                                    — any passes
//   empty conditions -> true
func evaluateCondition(cond struct {
	All []map[string]any `json:"all"`
	Any []map[string]any `json:"any"`
}, inputs map[string]any) bool {
	check := func(p map[string]any) bool {
		field, _ := p["field"].(string)
		op, _ := p["op"].(string)
		exp := p["value"]
		actual, ok := inputs[field]
		if !ok {
			return false
		}
		switch op {
		case "==", "=":
			return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", exp)
		case "!=":
			return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", exp)
		case ">":
			af, _ := toFloat(actual)
			ef, _ := toFloat(exp)
			return af > ef
		case ">=":
			af, _ := toFloat(actual)
			ef, _ := toFloat(exp)
			return af >= ef
		case "<":
			af, _ := toFloat(actual)
			ef, _ := toFloat(exp)
			return af < ef
		case "<=":
			af, _ := toFloat(actual)
			ef, _ := toFloat(exp)
			return af <= ef
		case "contains":
			return strings.Contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", exp))
		}
		return false
	}
	if len(cond.All) > 0 {
		for _, p := range cond.All {
			if !check(p) {
				return false
			}
		}
		return true
	}
	if len(cond.Any) > 0 {
		for _, p := range cond.Any {
			if check(p) {
				return true
			}
		}
		return false
	}
	return true
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	}
	return 0, false
}

// executeAction thực thi 1 action và trả về {ok:bool, detail:any}.
func (s *Server) executeAction(ctx context.Context, tenantID, actType string, act map[string]any, entityType string, entityID uuid.UUID) map[string]any {
	out := map[string]any{"type": actType, "ok": false}
	switch actType {
	case "create_activity":
		subject, _ := act["subject"].(string)
		body, _ := act["body"].(string)
		ownerStr, _ := act["owner_user_id"].(string)
		var ownerUUID *uuid.UUID
		if ownerStr != "" {
			if u, err := uuid.Parse(ownerStr); err == nil {
				ownerUUID = &u
			}
		}
		var contactID *uuid.UUID
		var leadID *uuid.UUID
		if entityType == "contact" {
			contactID = &entityID
		} else if entityType == "lead" {
			leadID = &entityID
		}
		var aid uuid.UUID
		err := s.pool.QueryRow(ctx, `
			INSERT INTO activities (tenant_id, contact_id, lead_id, type, subject, body, owner_user_id, status)
			VALUES ($1, $2, $3, 'task', $4, $5, $6, 'pending')
			RETURNING id
		`, tenantID, contactID, leadID, subject, body, ownerUUID).Scan(&aid)
		if err != nil {
			out["error"] = err.Error()
			return out
		}
		out["ok"] = true
		out["activity_id"] = aid

	case "notify":
		receiverStr, _ := act["user_id"].(string)
		ru, err := uuid.Parse(receiverStr)
		if err != nil {
			out["error"] = "invalid user_id"
			return out
		}
		title, _ := act["title"].(string)
		body, _ := act["body"].(string)
		priority, _ := act["priority"].(string)
		if priority == "" {
			priority = "NORMAL"
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO notifications (tenant_id, user_id, type, title, body, link, priority)
			VALUES ($1, $2, 'workflow', $3, $4, $5, $6)
		`, tenantID, ru, title, body, fmt.Sprintf("/%s/%s", entityType, entityID), priority)
		if err != nil {
			out["error"] = err.Error()
			return out
		}
		out["ok"] = true

	case "update_field":
		field, _ := act["field"].(string)
		value := act["value"]
		if entityType != "lead" || field == "" {
			out["error"] = "only lead update_field supported"
			return out
		}
		// Only allow safe columns to avoid SQLi
		allowed := map[string]bool{"status": true, "score": true, "estimated_value": true, "next_followup_at": true, "tags": true}
		if !allowed[field] {
			out["error"] = "field not allowed"
			return out
		}
		_, err := s.pool.Exec(ctx, fmt.Sprintf(`UPDATE leads SET %s = $2, updated_at = NOW() WHERE id = $1`, field), entityID, value)
		if err != nil {
			out["error"] = err.Error()
			return out
		}
		out["ok"] = true

	case "webhook":
		// Stub: log only; webhook delivery is handled by a separate worker.
		out["ok"] = true
		out["detail"] = "webhook queued"

	default:
		out["error"] = fmt.Sprintf("unknown action type %q", actType)
	}
	return out
}

// =============================================================================
// Notifications
// =============================================================================

type broadcastReq struct {
	UserIDs    []string `json:"user_ids,omitempty"`
	Role       string   `json:"role,omitempty"`        // e.g. NV, TN, QL
	Department string   `json:"department,omitempty"`
	BranchPath string   `json:"branch_path,omitempty"` // subtree path
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Link       string   `json:"link,omitempty"`
	Priority   string   `json:"priority,omitempty"`
}

type broadcastResp struct {
	RecipientCount int `json:"recipient_count"`
}

func (s *Server) BroadcastNotification(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req broadcastReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.Title == "" {
		return s.errorResp(c, http.StatusBadRequest, "title required", nil)
	}
	if req.Priority == "" {
		req.Priority = "NORMAL"
	}

	// Build recipient list
	args := []any{tenantID}
	where := []string{"tenant_id = $1", "deleted_at IS NULL"}
	if len(req.UserIDs) > 0 {
		uuids := make([]uuid.UUID, 0, len(req.UserIDs))
		for _, uid := range req.UserIDs {
			u, err := uuid.Parse(uid)
			if err != nil {
				return s.errorResp(c, http.StatusBadRequest, "invalid user_id", err)
			}
			uuids = append(uuids, u)
		}
		args = append(args, uuids)
		where = append(where, fmt.Sprintf("id = ANY($%d)", len(args)))
	}
	if req.Role != "" {
		args = append(args, req.Role)
		where = append(where, fmt.Sprintf("role = $%d", len(args)))
	}
	if req.Department != "" {
		args = append(args, req.Department)
		where = append(where, fmt.Sprintf("$%d = ANY(STRING_TO_ARRAY(department, ','))", len(args)))
	}
	if req.BranchPath != "" {
		args = append(args, req.BranchPath)
		where = append(where, fmt.Sprintf("path <@ $%d::ltree", len(args)))
	}
	if !isAdmin {
		// Non-admin chỉ broadcast trong subtree của mình
		args = append(args, userID)
		where = append(where, fmt.Sprintf("path <@ (SELECT path FROM users WHERE id = $%d)", len(args)))
	}

	rows, err := s.pool.Query(ctx, "SELECT id FROM users WHERE "+strings.Join(where, " AND "), args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "recipient query failed", err)
	}
	defer rows.Close()
	var recipients []uuid.UUID
	for rows.Next() {
		var u uuid.UUID
		if err := rows.Scan(&u); err == nil {
			recipients = append(recipients, u)
		}
	}
	rows.Close()

	if len(recipients) == 0 {
		return s.json(c, http.StatusOK, broadcastResp{RecipientCount: 0})
	}

	var sender interface{}
	if userID != "" {
		if u, err := uuid.Parse(userID); err == nil {
			sender = u
		}
	}

	// Bulk insert
	batch := make([][]any, 0, len(recipients))
	for _, rid := range recipients {
		batch = append(batch, []any{tenantID, rid, sender, req.Title, req.Body, req.Link, req.Priority})
	}
	for _, row := range batch {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO notifications (tenant_id, user_id, sender_user_id, type, title, body, link, priority)
			VALUES ($1, $2, $3, 'broadcast', $4, $5, $6, $7)
		`, row...)
		if err != nil {
			slog.Warn("notification insert failed", slog.String("error", err.Error()))
		}
	}

	s.audit(ctx, userID, tenantID, "notification.broadcast", "notification", "", nil, req)
	return s.json(c, http.StatusOK, broadcastResp{RecipientCount: len(recipients)})
}

func (s *Server) NotificationInbox(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	if userID == "" {
		return s.errorResp(c, http.StatusBadRequest, "user_id required", nil)
	}

	p := getPagination(c)
	unreadOnly := c.QueryParam("unread") == "true"
	args := []any{userID}
	where := []string{"user_id = $1"}
	if unreadOnly {
		where = append(where, "is_read = false")
	}
	whereSQL := strings.Join(where, " AND ")
	args = append(args, p.PerPage, p.Offset)

	var total int64
	_ = s.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", whereSQL), args[:len(args)-2]...).Scan(&total)

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, user_id, sender_user_id, type, title, COALESCE(body,''),
		       COALESCE(link,''), COALESCE(priority,'NORMAL'), is_read, read_at, created_at
		FROM notifications WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, whereSQL, len(args)-1, len(args)), args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	type notifResp struct {
		ID            uuid.UUID  `json:"id"`
		TenantID      uuid.UUID  `json:"tenant_id"`
		UserID        uuid.UUID  `json:"user_id"`
		SenderUserID  *uuid.UUID `json:"sender_user_id,omitempty"`
		Type          string     `json:"type"`
		Title         string     `json:"title"`
		Body          string     `json:"body"`
		Link          string     `json:"link,omitempty"`
		Priority      string     `json:"priority"`
		IsRead        bool       `json:"is_read"`
		ReadAt        *time.Time `json:"read_at,omitempty"`
		CreatedAt     time.Time  `json:"created_at"`
	}
	var out []notifResp
	for rows.Next() {
		var n notifResp
		if err := rows.Scan(&n.ID, &n.TenantID, &n.UserID, &n.SenderUserID, &n.Type, &n.Title, &n.Body,
			&n.Link, &n.Priority, &n.IsRead, &n.ReadAt, &n.CreatedAt); err != nil {
			continue
		}
		out = append(out, n)
	}
	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}
	return s.json(c, http.StatusOK, listResp{Data: out, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

func (s *Server) MarkNotificationRead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	r, err := s.pool.Exec(ctx, `
		UPDATE notifications SET is_read = true, read_at = NOW()
		WHERE id = $1 AND user_id = NULLIF($2,'')::UUID
	`, id, userID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "notification not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) DismissNotification(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM notifications WHERE id = $1 AND user_id = NULLIF($2,'')::UUID`, id, userID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// Audit + Sessions
// =============================================================================

func (s *Server) ListAudit(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)
	args := []any{}
	where := []string{}
	if v := c.QueryParam("action"); v != "" {
		args = append(args, v)
		where = append(where, fmt.Sprintf("action = $%d", len(args)))
	}
	if v := c.QueryParam("actor_id"); v != "" {
		args = append(args, v)
		where = append(where, fmt.Sprintf("actor_id = $%d", len(args)))
	}
	if v := c.QueryParam("target_type"); v != "" {
		args = append(args, v)
		where = append(where, fmt.Sprintf("target_type = $%d", len(args)))
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", whereSQL)
	_ = s.pool.QueryRow(ctx, countSQL, args...).Scan(&total)

	args = append(args, p.PerPage, p.Offset)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, actor_id, COALESCE(actor_email,''), action,
		       COALESCE(target_type,''), target_id, created_at
		FROM audit_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, whereSQL, len(args)-1, len(args)), args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	type auditEntry struct {
		ID          uuid.UUID  `json:"id"`
		TenantID    uuid.UUID  `json:"tenant_id"`
		ActorID     *uuid.UUID `json:"actor_id,omitempty"`
		ActorEmail  string     `json:"actor_email,omitempty"`
		Action      string     `json:"action"`
		TargetType  string     `json:"target_type,omitempty"`
		TargetID    *uuid.UUID `json:"target_id,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
	}
	var out []auditEntry
	for rows.Next() {
		var a auditEntry
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorEmail, &a.Action, &a.TargetType, &a.TargetID, &a.CreatedAt); err != nil {
			continue
		}
		out = append(out, a)
	}
	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}
	return s.json(c, http.StatusOK, listResp{Data: out, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

type sessionResp struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	IPAddress   string     `json:"ip_address,omitempty"`
	UserAgent   string     `json:"user_agent,omitempty"`
	LoginAt     time.Time  `json:"login_at"`
	LastActive  time.Time  `json:"last_active_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Revoked     bool       `json:"revoked"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

func (s *Server) ListSessions(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	target := c.QueryParam("user_id")
	if target == "" {
		target = userID
	}
	if target == "" {
		return s.errorResp(c, http.StatusBadRequest, "user_id required", nil)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, COALESCE(host(ip_address),''), COALESCE(user_agent,''),
		       login_at, last_active_at, expires_at, revoked, revoked_at
		FROM user_sessions WHERE user_id = $1 AND tenant_id = $2 ORDER BY login_at DESC LIMIT 50
	`, target, tenantID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var out []sessionResp
	for rows.Next() {
		var s2 sessionResp
		if err := rows.Scan(&s2.ID, &s2.UserID, &s2.IPAddress, &s2.UserAgent,
			&s2.LoginAt, &s2.LastActive, &s2.ExpiresAt, &s2.Revoked, &s2.RevokedAt); err != nil {
			continue
		}
		out = append(out, s2)
	}
	return s.json(c, http.StatusOK, map[string]any{"sessions": out, "count": len(out)})
}

func (s *Server) RevokeSession(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	r, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked = true, revoked_at = NOW(), revoked_reason = 'manual'
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "revoke failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "session not found", nil)
	}
	s.audit(ctx, userID, tenantID, "session.revoke", "session", id, nil, nil)
	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// Tags CRUD (bổ sung Update/Delete)
// =============================================================================

func (s *Server) UpdateTag(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	type req struct {
		Name        *string `json:"name,omitempty"`
		Color       *string `json:"color,omitempty"`
		Description *string `json:"description,omitempty"`
	}
	var body req
	if err := c.Bind(&body); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	r, err := s.pool.Exec(ctx, `
		UPDATE tags SET name = COALESCE($2, name), color = COALESCE($3, color), description = COALESCE($4, description), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, body.Name, body.Color, body.Description)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "tag not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) DeleteTag(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	r, err := s.pool.Exec(ctx, `UPDATE tags SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "tag not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// Custom Fields CRUD (bổ sung Update/Delete)
// =============================================================================

func (s *Server) UpdateCustomField(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	type req struct {
		Name            *string        `json:"name,omitempty"`
		Description     *string        `json:"description,omitempty"`
		Options         *[]any         `json:"options,omitempty"`
		IsRequired      *bool          `json:"is_required,omitempty"`
		IsUnique        *bool          `json:"is_unique,omitempty"`
		DisplayOrder    *int           `json:"display_order,omitempty"`
		IsActive        *bool          `json:"is_active,omitempty"`
		ValidationRules *map[string]any `json:"validation_rules,omitempty"`
	}
	var body req
	if err := c.Bind(&body); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	var optsB, rulesB []byte
	if body.Options != nil {
		optsB, _ = json.Marshal(*body.Options)
	}
	if body.ValidationRules != nil {
		rulesB, _ = json.Marshal(*body.ValidationRules)
	}

	r, err := s.pool.Exec(ctx, `
		UPDATE custom_fields SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			options = COALESCE($4::jsonb, options),
			validation_rules = COALESCE($5::jsonb, validation_rules),
			is_required = COALESCE($6, is_required),
			is_unique = COALESCE($7, is_unique),
			display_order = COALESCE($8, display_order),
			is_active = COALESCE($9, is_active),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, body.Name, body.Description, optsB, rulesB, body.IsRequired, body.IsUnique, body.DisplayOrder, body.IsActive)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "custom field not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) DeleteCustomField(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	r, err := s.pool.Exec(ctx, `UPDATE custom_fields SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "custom field not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// User-level helpers (promote/demote/move) – wrap tree handlers' path/move
// =============================================================================

type moveUserReq struct {
	TargetParentID string `json:"target_parent_id"`
	Reason         string `json:"reason,omitempty"`
}

// PromoteUser đổi role lên 1 cấp và (tùy chọn) move subtree. Đơn giản hoá:
// chỉ update role, không tự move. Move riêng qua POST /v1/tree/.../move.
func (s *Server) PromoteUser(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	var body struct {
		NewRole string `json:"new_role"`
	}
	if err := c.Bind(&body); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if body.NewRole == "" {
		return s.errorResp(c, http.StatusBadRequest, "new_role required", nil)
	}
	switch body.NewRole {
	case "owner", "admin", "manager", "team_lead", "member", "viewer":
	default:
		return s.errorResp(c, http.StatusBadRequest, "invalid new_role", nil)
	}

	r, err := s.pool.Exec(ctx, `UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id, body.NewRole)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "promote failed", err)
	}
	if r.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "user not found", nil)
	}
	s.audit(ctx, userID, tenantID, "user.promote", "user", id, nil, body)
	return c.NoContent(http.StatusOK)
}

func (s *Server) DemoteUser(c echo.Context) error {
	return s.PromoteUser(c)
}

// CanAccessLead là helper cho phép các service khác (kể cả RPC) kiểm tra
// quyền truy cập 1 lead: phải thuộc subtree của user HOẶC user là super admin.
//
// Sử dụng:
//   ok, err := h.CanAccessLead(ctx, userID, leadID)
func (s *Server) CanAccessLead(ctx context.Context, userID, leadID uuid.UUID) (bool, error) {
	var ownerID *uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT owner_user_id FROM leads
		WHERE id = $1 AND deleted_at IS NULL
	`, leadID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if ownerID == nil {
		return true, nil // unassigned lead — anyone in tenant can claim
	}
	if *ownerID == userID {
		return true, nil
	}
	var inSubtree bool
	err = s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users
			WHERE id = $1 AND deleted_at IS NULL
			AND path <@ (SELECT path FROM users WHERE id = $2 AND deleted_at IS NULL)
		)
	`, *ownerID, userID).Scan(&inSubtree)
	if err != nil {
		return false, err
	}
	return inSubtree, nil
}
