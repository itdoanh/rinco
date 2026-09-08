// Package handler provides HTTP handlers for lead service.
package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	leadnats "github.com/itdoanh/rinco/services/lead-service/internal/nats"
)

// Server holds dependencies for handlers.
type Server struct {
	pool       *pgxpool.Pool
	rdb        *RedisClient
	nats       *leadnats.Client
	scoringURL string
}

// RedisClient wraps Redis configuration.
type RedisClient struct {
	Addr     string
	Password string
	DB       int
}

func NewServer(pool *pgxpool.Pool, rdb *RedisClient, natsClient *leadnats.Client, scoringURL string) *Server {
	return &Server{pool: pool, rdb: rdb, nats: natsClient, scoringURL: scoringURL}
}

// Helper to get tenant context.
func (s *Server) tenantFromCtx(c echo.Context) (tenantID, userID string, isAdmin bool) {
	tenantID, _ = c.Get("tenant_id").(string)
	userID, _ = c.Get("user_id").(string)
	isAdmin, _ = c.Get("is_admin").(bool)
	return
}

// Helper to set RLS context with parameterized queries (SQL-injection-safe).
//
// KNOWN LIMITATION: Same as crm-service: SET LOCAL outside a tx has no
// effect. The variables are bound to a tx that's immediately committed,
// so subsequent handler queries on a different pool conn still see NULL.
//
// SQL injection is prevented by:
//   1. UUID-format validation for tenant_id/user_id
//   2. Parameterized queries via set_config()
func (s *Server) setRLS(ctx context.Context, tenantID, userID string, isAdmin bool) error {
	if tenantID == "" {
		return errors.New("setRLS: tenantID is required")
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("setRLS: invalid tenant_id: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}
	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return fmt.Errorf("setRLS: invalid user_id: %w", err)
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID); err != nil {
			return fmt.Errorf("set user: %w", err)
		}
	}
	adminVal := "false"
	if isAdmin {
		adminVal = "true"
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.is_admin', $1, true)", adminVal); err != nil {
		return fmt.Errorf("set admin: %w", err)
	}
	return tx.Commit(ctx)
}

// Pagination helper.
type pagination struct {
	Page    int
	PerPage int
	Offset  int
}

func getPagination(c echo.Context) pagination {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return pagination{Page: page, PerPage: perPage, Offset: (page - 1) * perPage}
}

// Response types.
type listResp struct {
	Data       any   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

type errResp struct {
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"`
}

func (s *Server) json(c echo.Context, status int, data any) error {
	return c.JSON(status, data)
}

func (s *Server) errorResp(c echo.Context, status int, msg string, err error) error {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return c.JSON(status, errResp{Error: msg, Details: details})
}

// =============================================================================
// Lead Handlers
// =============================================================================

type leadReq struct {
	SourceID        *uuid.UUID      `json:"source_id,omitempty"`
	ContactID       *uuid.UUID      `json:"contact_id,omitempty"`
	OwnerUserID     *uuid.UUID      `json:"owner_user_id,omitempty"`
	PipelineID      *uuid.UUID      `json:"pipeline_id,omitempty"`
	StageID         *uuid.UUID      `json:"stage_id,omitempty"`
	FullName        *string         `json:"full_name,omitempty"`
	Email           *string         `json:"email,omitempty"`
	Phone           *string         `json:"phone,omitempty"`
	CompanyName     *string         `json:"company_name,omitempty"`
	JobTitle        *string         `json:"job_title,omitempty"`
	Status          *string         `json:"status,omitempty"`
	Score           *float64        `json:"score,omitempty"`
	ScoreTier       *string         `json:"score_tier,omitempty"`
	EstimatedValue  *float64        `json:"estimated_value,omitempty"`
	CustomFields    *map[string]any `json:"custom_fields,omitempty"`
	UTM             *map[string]any `json:"utm,omitempty"`
	IP              *string         `json:"ip,omitempty"`
	UserAgent       *string         `json:"user_agent,omitempty"`
	Referrer        *string         `json:"referrer,omitempty"`
	FBCLID          *string         `json:"fbclid,omitempty"`
	FBP             *string         `json:"fbp,omitempty"`
	GCLID           *string         `json:"gclid,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	NextFollowupAt  *time.Time      `json:"next_followup_at,omitempty"`
	LostReason      *string         `json:"lost_reason,omitempty"`
}

type leadResp struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	SourceID        *uuid.UUID     `json:"source_id,omitempty"`
	ContactID       *uuid.UUID     `json:"contact_id,omitempty"`
	OwnerUserID     *uuid.UUID     `json:"owner_user_id,omitempty"`
	PipelineID      *uuid.UUID     `json:"pipeline_id,omitempty"`
	StageID         *uuid.UUID     `json:"stage_id,omitempty"`
	FullName        string         `json:"full_name"`
	Email           string         `json:"email,omitempty"`
	Phone           string         `json:"phone,omitempty"`
	CompanyName     string         `json:"company_name,omitempty"`
	JobTitle        string         `json:"job_title,omitempty"`
	Status          string         `json:"status"`
	Score           float64        `json:"score"`
	ScoreTier       string         `json:"score_tier,omitempty"`
	EstimatedValue  float64        `json:"estimated_value"`
	CustomFields    map[string]any `json:"custom_fields,omitempty"`
	UTM             map[string]any `json:"utm,omitempty"`
	Tags            []string       `json:"tags"`
	NextFollowupAt  *time.Time     `json:"next_followup_at,omitempty"`
	LastContactedAt *time.Time     `json:"last_contacted_at,omitempty"`
	ConvertedAt     *time.Time     `json:"converted_at,omitempty"`
	LostReason      string         `json:"lost_reason,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (s *Server) ListLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)

	var total int64
	countQuery := `SELECT COUNT(*) FROM leads WHERE deleted_at IS NULL`
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}

	query := `SELECT id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
		FROM leads WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var leads []leadResp
	for rows.Next() {
		var l leadResp
		var cf, utm []byte
		if err := rows.Scan(&l.ID, &l.TenantID, &l.SourceID, &l.ContactID, &l.OwnerUserID, &l.PipelineID, &l.StageID, &l.FullName, &l.Email, &l.Phone, &l.CompanyName, &l.JobTitle, &l.Status, &l.Score, &l.ScoreTier, &l.EstimatedValue, &cf, &utm, &l.Tags, &l.NextFollowupAt, &l.LastContactedAt, &l.ConvertedAt, &l.LostReason, &l.CreatedAt, &l.UpdatedAt); err != nil {
			continue
		}
		if cf != nil {
			json.Unmarshal(cf, &l.CustomFields)
		}
		if utm != nil {
			json.Unmarshal(utm, &l.UTM)
		}
		leads = append(leads, l)
	}

	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}

	return s.json(c, http.StatusOK, listResp{Data: leads, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
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

	if req.FullName == nil || *req.FullName == "" {
		return s.errorResp(c, http.StatusBadRequest, "full_name is required", nil)
	}

	var cfBytes, utmBytes []byte
	if req.CustomFields != nil {
		cfBytes, _ = json.Marshal(req.CustomFields)
	}
	if req.UTM != nil {
		utmBytes, _ = json.Marshal(req.UTM)
	}

	status := "new"
	if req.Status != nil {
		status = *req.Status
	}
	score := 0.0
	if req.Score != nil {
		score = *req.Score
	}
	estValue := 0.0
	if req.EstimatedValue != nil {
		estValue = *req.EstimatedValue
	}

	var resp leadResp
	var cf, utm []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO leads (tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, ip, user_agent, referrer, fbclid, fbp, gclid, tags, next_followup_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		RETURNING id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
	`, tenantID, req.SourceID, req.ContactID, req.OwnerUserID, req.PipelineID, req.StageID, *req.FullName, req.Email, req.Phone, req.CompanyName, req.JobTitle, status, score, req.ScoreTier, estValue, cfBytes, utmBytes, req.IP, req.UserAgent, req.Referrer, req.FBCLID, req.FBP, req.GCLID, req.Tags, req.NextFollowupAt).
		Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.PipelineID, &resp.StageID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.JobTitle, &resp.Status, &resp.Score, &resp.ScoreTier, &resp.EstimatedValue, &cf, &utm, &resp.Tags, &resp.NextFollowupAt, &resp.LastContactedAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		slog.Error("create lead failed", slog.String("error", err.Error()))
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}
	if utm != nil {
		json.Unmarshal(utm, &resp.UTM)
	}

	// Publish to NATS
	if s.nats != nil {
		go s.nats.Publish(ctx, leadnats.SubjectLeadCreated, resp)
	}

	// Log activity
	actorID, _ := uuid.Parse(userID)
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (tenant_id, lead_id, type, actor_id, description, payload)
		VALUES ($1, $2, 'NOTE', $3, $4, $5)
	`, tenantID, resp.ID, actorID, "Lead created", "{}")

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var resp leadResp
	var cf, utm []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
		FROM leads WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.PipelineID, &resp.StageID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.JobTitle, &resp.Status, &resp.Score, &resp.ScoreTier, &resp.EstimatedValue, &cf, &utm, &resp.Tags, &resp.NextFollowupAt, &resp.LastContactedAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}
	if utm != nil {
		json.Unmarshal(utm, &resp.UTM)
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
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req leadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var resp leadResp
	var cf, utm []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE leads SET 
			source_id = COALESCE($2, source_id),
			contact_id = COALESCE($3, contact_id),
			owner_user_id = COALESCE($4, owner_user_id),
			pipeline_id = COALESCE($5, pipeline_id),
			stage_id = COALESCE($6, stage_id),
			full_name = COALESCE($7, full_name),
			email = COALESCE($8, email),
			phone = COALESCE($9, phone),
			company_name = COALESCE($10, company_name),
			job_title = COALESCE($11, job_title),
			status = COALESCE($12, status),
			score = COALESCE($13, score),
			score_tier = COALESCE($14, score_tier),
			estimated_value = COALESCE($15, estimated_value),
			custom_fields = COALESCE($16, custom_fields),
			utm = COALESCE($17, utm),
			tags = COALESCE($18, tags),
			next_followup_at = COALESCE($19, next_followup_at),
			lost_reason = COALESCE($20, lost_reason),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
	`, id, req.SourceID, req.ContactID, req.OwnerUserID, req.PipelineID, req.StageID, req.FullName, req.Email, req.Phone, req.CompanyName, req.JobTitle, req.Status, req.Score, req.ScoreTier, req.EstimatedValue, req.CustomFields, req.UTM, req.Tags, req.NextFollowupAt, req.LostReason).
		Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.PipelineID, &resp.StageID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.JobTitle, &resp.Status, &resp.Score, &resp.ScoreTier, &resp.EstimatedValue, &cf, &utm, &resp.Tags, &resp.NextFollowupAt, &resp.LastContactedAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}
	if utm != nil {
		json.Unmarshal(utm, &resp.UTM)
	}

	// Publish update event
	if s.nats != nil {
		go s.nats.Publish(ctx, leadnats.SubjectLeadUpdated, resp)
	}

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
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	result, err := s.pool.Exec(ctx, `UPDATE leads SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if result.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// Lead Operations
// =============================================================================

type assignLeadReq struct {
	OwnerUserID *uuid.UUID `json:"owner_user_id,omitempty"`
	Subtree     *bool      `json:"subtree,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

func (s *Server) AssignLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req assignLeadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.OwnerUserID == nil {
		return s.errorResp(c, http.StatusBadRequest, "owner_user_id is required", nil)
	}

	var resp leadResp
	var cf, utm []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE leads SET owner_user_id = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
	`, id, *req.OwnerUserID).
		Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.PipelineID, &resp.StageID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.JobTitle, &resp.Status, &resp.Score, &resp.ScoreTier, &resp.EstimatedValue, &cf, &utm, &resp.Tags, &resp.NextFollowupAt, &resp.LastContactedAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}

	// Record assignment
	oldOwner, _ := uuid.Parse(userID)
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_assignments (tenant_id, lead_id, from_user_id, to_user_id, reason, assigned_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, id, oldOwner, *req.OwnerUserID, req.Reason, oldOwner)

	// Log activity
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (tenant_id, lead_id, type, actor_id, description, payload)
		VALUES ($1, $2, 'ASSIGN', $3, $4, $5)
	`, tenantID, id, oldOwner, fmt.Sprintf("Lead assigned to %s", req.OwnerUserID.String()), "{}")

	// Publish event
	if s.nats != nil {
		go s.nats.Publish(ctx, leadnats.SubjectLeadAssigned, map[string]any{
			"lead_id":      id,
			"to_user_id":   req.OwnerUserID,
			"from_user_id": oldOwner,
			"reason":       req.Reason,
		})
	}

	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}
	if utm != nil {
		json.Unmarshal(utm, &resp.UTM)
	}
	return s.json(c, http.StatusOK, resp)
}

type updateStatusReq struct {
	Status     string `json:"status"`
	LostReason string `json:"lost_reason,omitempty"`
}

func (s *Server) UpdateLeadStatus(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req updateStatusReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	validStatuses := map[string]bool{"new": true, "contacted": true, "qualified": true, "proposal": true, "won": true, "lost": true, "archived": true}
	if !validStatuses[req.Status] {
		return s.errorResp(c, http.StatusBadRequest, "invalid status", nil)
	}

	// Get old status for history
	var oldStatus string
	s.pool.QueryRow(ctx, `SELECT status FROM leads WHERE id = $1`, id).Scan(&oldStatus)

	var resp leadResp
	var cf, utm []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE leads SET 
			status = $2, 
			lost_reason = CASE WHEN $2 = 'lost' THEN $3 ELSE NULL END,
			converted_at = CASE WHEN $2 = 'won' THEN NOW() ELSE converted_at END,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
	`, id, req.Status, req.LostReason).
		Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.PipelineID, &resp.StageID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.JobTitle, &resp.Status, &resp.Score, &resp.ScoreTier, &resp.EstimatedValue, &cf, &utm, &resp.Tags, &resp.NextFollowupAt, &resp.LastContactedAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}

	// Record history
	actorID, _ := uuid.Parse(userID)
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_stage_history (tenant_id, lead_id, from_stage, to_stage, changed_by)
		VALUES ($1, $2, $3, $4, $5)
	`, tenantID, id, oldStatus, req.Status, actorID)

	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (tenant_id, lead_id, type, actor_id, description, payload)
		VALUES ($1, $2, 'STATUS_CHANGE', $3, $4, $5)
	`, tenantID, id, actorID, fmt.Sprintf("Status changed from %s to %s", oldStatus, req.Status), "{}")

	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}
	if utm != nil {
		json.Unmarshal(utm, &resp.UTM)
	}
	return s.json(c, http.StatusOK, resp)
}

// =============================================================================
// Lead Bulk Operations
// =============================================================================

func (s *Server) ImportLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	// Read CSV
	file, err := c.FormFile("file")
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "file is required", err)
	}
	src, err := file.Open()
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "open file failed", err)
	}
	defer src.Close()

	reader := csv.NewReader(src)
	reader.FieldsPerRecord = -1 // Allow variable fields

	header, err := reader.Read()
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "csv read failed", err)
	}

	actorID, _ := uuid.Parse(userID)
	var imported int
	var errors []string

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		// Map CSV row to lead
		lead := map[string]string{}
		for i, h := range header {
			if i < len(row) {
				lead[strings.ToLower(strings.TrimSpace(h))] = row[i]
			}
		}

		fullName := lead["full_name"]
		if fullName == "" {
			fullName = lead["name"]
		}
		if fullName == "" {
			errors = append(errors, "missing full_name")
			continue
		}

		_, err = s.pool.Exec(ctx, `
			INSERT INTO leads (tenant_id, full_name, email, phone, company_name, status, source_id)
			VALUES ($1, $2, $3, $4, $5, 'new', NULL)
		`, tenantID, fullName, lead["email"], lead["phone"], lead["company_name"])

		if err != nil {
			errors = append(errors, fmt.Sprintf("row '%s': %s", fullName, err.Error()))
			continue
		}
		imported++
	}

	_, _ = actorID, tenantID
	return s.json(c, http.StatusOK, map[string]any{
		"imported": imported,
		"errors":   errors,
	})
}

func (s *Server) ExportLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="leads.csv"`)

	writer := csv.NewWriter(c.Response())
	defer writer.Flush()

	headers := []string{"id", "full_name", "email", "phone", "company_name", "status", "score", "score_tier", "created_at"}
	if err := writer.Write(headers); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "write headers failed", err)
	}

	rows, err := s.pool.Query(ctx, `SELECT id, full_name, email, phone, company_name, status, score, score_tier, created_at FROM leads WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var fullName, email, phone, companyName, status, scoreTier string
		var score float64
		var createdAt time.Time
		if err := rows.Scan(&id, &fullName, &email, &phone, &companyName, &status, &score, &scoreTier, &createdAt); err != nil {
			continue
		}
		writer.Write([]string{
			id.String(), fullName, email, phone, companyName, status,
			strconv.FormatFloat(score, 'f', 2, 64), scoreTier, createdAt.Format(time.RFC3339),
		})
	}

	return nil
}

func (s *Server) GetLeadsBySource(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	source := c.Param("source")

	p := getPagination(c)
	query := `SELECT id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
		FROM leads WHERE deleted_at IS NULL AND utm->>'source' = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, query, source, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var leads []leadResp
	for rows.Next() {
		var l leadResp
		var cf, utm []byte
		if err := rows.Scan(&l.ID, &l.TenantID, &l.SourceID, &l.ContactID, &l.OwnerUserID, &l.PipelineID, &l.StageID, &l.FullName, &l.Email, &l.Phone, &l.CompanyName, &l.JobTitle, &l.Status, &l.Score, &l.ScoreTier, &l.EstimatedValue, &cf, &utm, &l.Tags, &l.NextFollowupAt, &l.LastContactedAt, &l.ConvertedAt, &l.LostReason, &l.CreatedAt, &l.UpdatedAt); err != nil {
			continue
		}
		if cf != nil {
			json.Unmarshal(cf, &l.CustomFields)
		}
		if utm != nil {
			json.Unmarshal(utm, &l.UTM)
		}
		leads = append(leads, l)
	}

	return s.json(c, http.StatusOK, map[string]any{"leads": leads})
}

func (s *Server) StreamLeads(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return s.errorResp(c, http.StatusInternalServerError, "streaming unsupported", nil)
	}

	ctx := c.Request().Context()

	// Send initial event
	fmt.Fprintf(c.Response(), "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			tenantID, _, _ := s.tenantFromCtx(c)
			if tenantID != "" {
				var count int
				s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM leads WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at > NOW() - INTERVAL '10 seconds'`, tenantID).Scan(&count)
				fmt.Fprintf(c.Response(), "event: tick\ndata: {\"new_leads\": %d}\n\n", count)
				flusher.Flush()
			}
		}
	}
}

func (s *Server) GetLeadStats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	// By stage
	stageQuery := `SELECT status, COUNT(*) FROM leads WHERE deleted_at IS NULL GROUP BY status`
	rows, err := s.pool.Query(ctx, stageQuery)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	byStage := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		byStage[status] = count
	}

	// By source
	sourceQuery := `SELECT COALESCE(utm->>'source', 'unknown') as src, COUNT(*) FROM leads WHERE deleted_at IS NULL GROUP BY src`
	rows2, err := s.pool.Query(ctx, sourceQuery)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows2.Close()

	bySource := map[string]int{}
	for rows2.Next() {
		var src string
		var count int
		if err := rows2.Scan(&src, &count); err != nil {
			continue
		}
		bySource[src] = count
	}

	// Total count
	var total int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM leads WHERE deleted_at IS NULL`).Scan(&total)

	return s.json(c, http.StatusOK, map[string]any{
		"total":     total,
		"by_stage":  byStage,
		"by_source": bySource,
	})
}

func (s *Server) GetLeadTimeline(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	query := `SELECT id, tenant_id, lead_id, type, payload, actor_id, description, created_at FROM lead_activities WHERE lead_id = $1 ORDER BY created_at DESC LIMIT 100`

	rows, err := s.pool.Query(ctx, query, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var activities []map[string]any
	for rows.Next() {
		var activityID, tenantIDAct, leadID uuid.UUID
		var activityType, description string
		var payload []byte
		var actorID *uuid.UUID
		var createdAt time.Time
		if err := rows.Scan(&activityID, &tenantIDAct, &leadID, &activityType, &payload, &actorID, &description, &createdAt); err != nil {
			continue
		}
		var payloadMap map[string]any
		if payload != nil {
			json.Unmarshal(payload, &payloadMap)
		}
		activities = append(activities, map[string]any{
			"id":          activityID,
			"type":        activityType,
			"actor_id":    actorID,
			"description": description,
			"payload":     payloadMap,
			"created_at":  createdAt,
		})
	}

	return s.json(c, http.StatusOK, map[string]any{"activities": activities})
}

func (s *Server) LeadScore(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	// Trigger async scoring via NATS
	if s.nats != nil {
		go s.nats.Publish(ctx, "lead.score.requested", map[string]any{
			"lead_id":   id,
			"tenant_id": tenantID,
		})
	}

	// For sync, compute basic score locally as fallback
	var email, phone string
	var utmBytes []byte
	var estimatedValue float64
	err = s.pool.QueryRow(ctx, `SELECT email, phone, utm, estimated_value FROM leads WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&email, &phone, &utmBytes, &estimatedValue)
	if err != nil {
		return s.errorResp(c, http.StatusNotFound, "lead not found", err)
	}

	var utm map[string]any
	if utmBytes != nil {
		json.Unmarshal(utmBytes, &utm)
	}

	score := 50.0
	if email != "" {
		score += 10
	}
	if phone != "" {
		score += 10
	}
	if estimatedValue > 0 {
		score += 15
	}
	if utm != nil && utm["source"] != nil {
		score += 5
	}
	if score > 100 {
		score = 100
	}
	scoreTier := "cold"
	if score >= 70 {
		scoreTier = "hot"
	} else if score >= 40 {
		scoreTier = "warm"
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE leads SET score = $2, score_tier = $3, updated_at = NOW() WHERE id = $1
	`, id, score, scoreTier)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}

	actorID, _ := uuid.Parse(userID)
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (tenant_id, lead_id, type, actor_id, description, payload)
		VALUES ($1, $2, 'SCORE_UPDATE', $3, 'Score updated', $4)
	`, tenantID, id, actorID, fmt.Sprintf(`{"score":%f,"tier":"%s"}`, score, scoreTier))

	return s.json(c, http.StatusOK, map[string]any{
		"lead_id":    id,
		"score":      score,
		"score_tier": scoreTier,
	})
}
