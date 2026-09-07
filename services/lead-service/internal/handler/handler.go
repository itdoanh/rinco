// Package handler provides HTTP handlers for Lead service.
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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// Server holds dependencies for handlers.
type Server struct {
	pool *pgxpool.Pool
}

// NewServer creates a new server.
func NewServer(pool *pgxpool.Pool) *Server {
	return &Server{pool: pool}
}

// Helper to get tenant context.
func (s *Server) tenantFromCtx(c echo.Context) (tenantID, userID string, isAdmin bool) {
	tenantID, _ = c.Get("tenant_id").(string)
	userID, _ = c.Get("user_id").(string)
	isAdmin, _ = c.Get("is_admin").(bool)
	return
}

// Helper to set RLS context.
func (s *Server) setRLS(ctx context.Context, tenantID, userID string, isAdmin bool) error {
	if _, err := s.pool.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}
	if userID != "" {
		if _, err := s.pool.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_user_id = '%s'", userID)); err != nil {
			return fmt.Errorf("set user: %w", err)
		}
	}
	adminVal := "false"
	if isAdmin {
		adminVal = "true"
	}
	if _, err := s.pool.Exec(ctx, fmt.Sprintf("SET LOCAL app.is_admin = '%s'", adminVal)); err != nil {
		return fmt.Errorf("set admin: %w", err)
	}
	return nil
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

func (s *Server) json(c echo.Context, status int, data any) error {
	return c.JSON(status, data)
}

func (s *Server) errorResp(c echo.Context, status int, msg string, err error) error {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return c.JSON(status, map[string]any{"error": msg, "details": details})
}

// =============================================================================
// Leads Handlers
// =============================================================================

type leadReq struct {
	SourceID    *uuid.UUID          `json:"source_id,omitempty"`
	OwnerUserID *uuid.UUID         `json:"owner_user_id,omitempty"`
	FullName    *string            `json:"full_name,omitempty"`
	Email       *string            `json:"email,omitempty"`
	Phone       *string            `json:"phone,omitempty"`
	CompanyName *string            `json:"company_name,omitempty"`
	Status      *string            `json:"status,omitempty"`
	UTM         *map[string]string `json:"utm,omitempty"`
	CustomFields *map[string]any   `json:"custom_fields,omitempty"`
}

type leadResp struct {
	ID              uuid.UUID         `json:"id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	SourceID        *uuid.UUID       `json:"source_id,omitempty"`
	ContactID       *uuid.UUID       `json:"contact_id,omitempty"`
	OwnerUserID     *uuid.UUID       `json:"owner_user_id,omitempty"`
	FullName        string           `json:"full_name"`
	Email           string           `json:"email,omitempty"`
	Phone           string           `json:"phone,omitempty"`
	CompanyName     string           `json:"company_name,omitempty"`
	Status          string           `json:"status"`
	Score           float64          `json:"score"`
	ScoreTier       string           `json:"score_tier,omitempty"`
	UTM             map[string]string `json:"utm,omitempty"`
	CustomFields    map[string]any   `json:"custom_fields,omitempty"`
	LastContactedAt *time.Time       `json:"last_contacted_at,omitempty"`
	NextFollowupAt  *time.Time       `json:"next_followup_at,omitempty"`
	ConvertedAt     *time.Time       `json:"converted_at,omitempty"`
	LostReason      string           `json:"lost_reason,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
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

	query := `SELECT id, tenant_id, source_id, contact_id, owner_user_id, full_name, email, phone, company_name, status, score, score_tier, utm, custom_fields, last_contacted_at, next_followup_at, converted_at, lost_reason, created_at, updated_at
		FROM leads WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var leads []leadResp
	for rows.Next() {
		var l leadResp
		var utmBytes, cfBytes []byte
		if err := rows.Scan(&l.ID, &l.TenantID, &l.SourceID, &l.ContactID, &l.OwnerUserID, &l.FullName, &l.Email, &l.Phone, &l.CompanyName, &l.Status, &l.Score, &l.ScoreTier, &utmBytes, &cfBytes, &l.LastContactedAt, &l.NextFollowupAt, &l.ConvertedAt, &l.LostReason, &l.CreatedAt, &l.UpdatedAt); err != nil {
			continue
		}
		if utmBytes != nil {
			json.Unmarshal(utmBytes, &l.UTM)
		}
		if cfBytes != nil {
			json.Unmarshal(cfBytes, &l.CustomFields)
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

	var utmBytes, cfBytes []byte
	if req.UTM != nil {
		utmBytes, _ = json.Marshal(req.UTM)
	}
	if req.CustomFields != nil {
		cfBytes, _ = json.Marshal(req.CustomFields)
	}

	status := "new"
	if req.Status != nil {
		status = *req.Status
	}

	// Get client info
	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	var resp leadResp
	var utmB, cfB []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO leads (tenant_id, source_id, owner_user_id, full_name, email, phone, company_name, status, utm, custom_fields, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, tenant_id, source_id, contact_id, owner_user_id, full_name, email, phone, company_name, status, score, score_tier, utm, custom_fields, last_contacted_at, next_followup_at, converted_at, lost_reason, created_at, updated_at
	`, tenantID, req.SourceID, req.OwnerUserID, *req.FullName, req.Email, req.Phone, req.CompanyName, status, utmBytes, cfBytes, ip, userAgent).
		Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.Status, &resp.Score, &resp.ScoreTier, &utmB, &cfB, &resp.LastContactedAt, &resp.NextFollowupAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		slog.Error("create lead failed", slog.String("error", err.Error()))
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	if utmB != nil {
		json.Unmarshal(utmB, &resp.UTM)
	}
	if cfB != nil {
		json.Unmarshal(cfB, &resp.CustomFields)
	}

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
	var utmBytes, cfBytes []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, source_id, contact_id, owner_user_id, full_name, email, phone, company_name, status, score, score_tier, utm, custom_fields, last_contacted_at, next_followup_at, converted_at, lost_reason, created_at, updated_at
		FROM leads WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.Status, &resp.Score, &resp.ScoreTier, &utmBytes, &cfBytes, &resp.LastContactedAt, &resp.NextFollowupAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	if utmBytes != nil {
		json.Unmarshal(utmBytes, &resp.UTM)
	}
	if cfBytes != nil {
		json.Unmarshal(cfBytes, &resp.CustomFields)
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
	var utmBytes, cfBytes []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE leads SET 
			source_id = COALESCE($2, source_id),
			owner_user_id = COALESCE($3, owner_user_id),
			full_name = COALESCE($4, full_name),
			email = COALESCE($5, email),
			phone = COALESCE($6, phone),
			company_name = COALESCE($7, company_name),
			status = COALESCE($8, status),
			utm = COALESCE($9, utm),
			custom_fields = COALESCE($10, custom_fields),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, source_id, contact_id, owner_user_id, full_name, email, phone, company_name, status, score, score_tier, utm, custom_fields, last_contacted_at, next_followup_at, converted_at, lost_reason, created_at, updated_at
	`, id, req.SourceID, req.OwnerUserID, req.FullName, req.Email, req.Phone, req.CompanyName, req.Status, req.UTM, req.CustomFields).
		Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.Status, &resp.Score, &resp.ScoreTier, &utmBytes, &cfBytes, &resp.LastContactedAt, &resp.NextFollowupAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if utmBytes != nil {
		json.Unmarshal(utmBytes, &resp.UTM)
	}
	if cfBytes != nil {
		json.Unmarshal(cfBytes, &resp.CustomFields)
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
// Lead Assignment
// =============================================================================

type assignLeadReq struct {
	ToUserID *uuid.UUID `json:"to_user_id"`
	Reason   string    `json:"reason,omitempty"`
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

	if req.ToUserID == nil {
		return s.errorResp(c, http.StatusBadRequest, "to_user_id is required", nil)
	}

	// Get current owner
	var currentOwner *uuid.UUID
	s.pool.QueryRow(ctx, `SELECT owner_user_id FROM leads WHERE id = $1`, id).Scan(&currentOwner)

	// Update lead owner
	_, err = s.pool.Exec(ctx, `
		UPDATE leads SET owner_user_id = $1, updated_at = NOW() WHERE id = $2
	`, req.ToUserID, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "assign failed", err)
	}

	// Record assignment
	fromStr := ""
	if currentOwner != nil {
		fromStr = currentOwner.String()
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO lead_assignments (lead_id, from_user_id, to_user_id, reason)
		VALUES ($1, $2, $3, $4)
	`, id, fromStr, req.ToUserID.String(), req.Reason)
	if err != nil {
		slog.Warn("record assignment failed", slog.String("error", err.Error()))
	}

	// Record activity
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (lead_id, type, payload)
		VALUES ($1, 'assignment', $2)
	`, id, fmt.Sprintf(`{"from": "%s", "to": "%s"}`, fromStr, req.ToUserID.String()))

	return s.json(c, http.StatusOK, map[string]string{"status": "assigned"})
}

// =============================================================================
// Lead Status
// =============================================================================

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

	// Get old status
	var oldStatus string
	s.pool.QueryRow(ctx, `SELECT status FROM leads WHERE id = $1`, id).Scan(&oldStatus)

	// Update status
	_, err = s.pool.Exec(ctx, `
		UPDATE leads SET 
			status = $1,
			lost_reason = $2,
			last_contacted_at = CASE WHEN $1 != old_status THEN NOW() ELSE last_contacted_at END,
			updated_at = NOW()
		WHERE id = $3
	`, req.Status, req.LostReason, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update status failed", err)
	}

	// Record stage history
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_stage_history (lead_id, from_stage, to_stage, changed_by)
		VALUES ($1, $2, $3, $4)
	`, id, oldStatus, req.Status, userID)

	// Record activity
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (lead_id, type, payload)
		VALUES ($1, 'status_change', $2)
	`, id, fmt.Sprintf(`{"from": "%s", "to": "%s"}`, oldStatus, req.Status))

	return s.json(c, http.StatusOK, map[string]string{"status": req.Status})
}

// =============================================================================
// Lead Scoring
// =============================================================================

type scoreLeadReq struct {
	Score     *float64 `json:"score,omitempty"`
	ScoreTier *string  `json:"score_tier,omitempty"`
}

func (s *Server) ScoreLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req scoreLeadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	// If no score provided, calculate based on engagement
	score := float64(50)
	if req.Score != nil {
		score = *req.Score
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	scoreTier := "warm"
	if score < 30 {
		scoreTier = "cold"
	} else if score > 70 {
		scoreTier = "hot"
	}
	if req.ScoreTier != nil {
		scoreTier = *req.ScoreTier
	}

	// Update score
	_, err = s.pool.Exec(ctx, `
		UPDATE leads SET score = $1, score_tier = $2, updated_at = NOW() WHERE id = $3
	`, score, scoreTier, parsedID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "score update failed", err)
	}

	// Record activity
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (lead_id, type, payload)
		VALUES ($1, 'score_update', $2)
	`, id, fmt.Sprintf(`{"score": %f, "tier": "%s"}`, score, scoreTier))

	return s.json(c, http.StatusOK, map[string]any{"score": score, "score_tier": scoreTier})
}

// =============================================================================
// Lead Notes
// =============================================================================

type noteReq struct {
	Body string `json:"body"`
}

type noteResp struct {
	ID        uuid.UUID `json:"id"`
	LeadID    uuid.UUID `json:"lead_id"`
	AuthorID  *string   `json:"author_id,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Server) ListLeadNotes(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	leadID := c.Param("id")
	_, err := uuid.Parse(leadID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid lead id", err)
	}

	query := `SELECT id, lead_id, author_id, body, created_at, updated_at 
		FROM lead_notes WHERE lead_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, leadID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var notes []noteResp
	for rows.Next() {
		var n noteResp
		if err := rows.Scan(&n.ID, &n.LeadID, &n.AuthorID, &n.Body, &n.CreatedAt, &n.UpdatedAt); err != nil {
			continue
		}
		notes = append(notes, n)
	}

	return s.json(c, http.StatusOK, map[string]any{"notes": notes})
}

func (s *Server) CreateLeadNote(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	leadID := c.Param("id")
	_, err := uuid.Parse(leadID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid lead id", err)
	}

	var req noteReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Body == "" {
		return s.errorResp(c, http.StatusBadRequest, "body is required", nil)
	}

	var resp noteResp
	err = s.pool.QueryRow(ctx, `
		INSERT INTO lead_notes (lead_id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, lead_id, author_id, body, created_at, updated_at
	`, leadID, userID, req.Body).
		Scan(&resp.ID, &resp.LeadID, &resp.AuthorID, &resp.Body, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create note failed", err)
	}

	// Record activity
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (lead_id, type, payload)
		VALUES ($1, 'note', $2)
	`, leadID, fmt.Sprintf(`{"note_id": "%s"}`, resp.ID.String()))

	return s.json(c, http.StatusCreated, resp)
}

// =============================================================================
// Lead Timeline
// =============================================================================

func (s *Server) GetLeadTimeline(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	leadID := c.Param("id")
	_, err := uuid.Parse(leadID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid lead id", err)
	}

	query := `SELECT id, lead_id, type, payload, created_at 
		FROM lead_activities WHERE lead_id = $1 ORDER BY created_at DESC LIMIT 100`

	rows, err := s.pool.Query(ctx, query, leadID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var activities []map[string]any
	for rows.Next() {
		var id, lID uuid.UUID
		var actType string
		var payload []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &lID, &actType, &payload, &createdAt); err != nil {
			continue
		}
		var payloadMap map[string]any
		if payload != nil {
			json.Unmarshal(payload, &payloadMap)
		}
		activities = append(activities, map[string]any{
			"id":         id,
			"lead_id":    lID,
			"type":       actType,
			"payload":    payloadMap,
			"created_at": createdAt,
		})
	}

	return s.json(c, http.StatusOK, map[string]any{"activities": activities})
}

// =============================================================================
// Lead Conversion
// =============================================================================

type convertLeadReq struct {
	ContactID   *uuid.UUID `json:"contact_id,omitempty"`
	DealName    *string    `json:"deal_name,omitempty"`
	DealValue   *float64   `json:"deal_value,omitempty"`
}

func (s *Server) ConvertLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 15*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req convertLeadReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	// Get lead info
	var lead leadResp
	var utmBytes, cfBytes []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, full_name, email, phone, company_name, owner_user_id FROM leads WHERE id = $1
	`, parsedID).Scan(&lead.ID, &lead.TenantID, &lead.FullName, &lead.Email, &lead.Phone, &lead.CompanyName, &lead.OwnerUserID)
	if err != nil {
		return s.errorResp(c, http.StatusNotFound, "lead not found", err)
	}

	// Update lead status to won
	_, err = s.pool.Exec(ctx, `
		UPDATE leads SET status = 'won', converted_at = NOW(), updated_at = NOW() WHERE id = $1
	`, parsedID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "conversion failed", err)
	}

	// Record activity
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (lead_id, type, payload)
		VALUES ($1, 'conversion', $2)
	`, id, fmt.Sprintf(`{"converted_at": "%s"}`, time.Now().Format(time.RFC3339)))

	return s.json(c, http.StatusOK, map[string]any{
		"status":       "converted",
		"lead_id":      id,
		"full_name":    lead.FullName,
		"email":        lead.Email,
		"company_name": lead.CompanyName,
	})
}

// =============================================================================
// Lead by Source
// =============================================================================

func (s *Server) GetLeadsBySource(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	source := c.QueryParam("source")
	if source == "" {
		return s.errorResp(c, http.StatusBadRequest, "source query param required", nil)
	}

	query := `SELECT id, tenant_id, source_id, contact_id, owner_user_id, full_name, email, phone, company_name, status, score, score_tier, utm, custom_fields, last_contacted_at, next_followup_at, converted_at, lost_reason, created_at, updated_at
		FROM leads WHERE deleted_at IS NULL AND (utm->>'source' = $1 OR source_id IN (SELECT id FROM lead_sources WHERE utm_source = $1))
		ORDER BY created_at DESC LIMIT 100`

	rows, err := s.pool.Query(ctx, query, source)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var leads []leadResp
	for rows.Next() {
		var l leadResp
		var utmBytes, cfBytes []byte
		if err := rows.Scan(&l.ID, &l.TenantID, &l.SourceID, &l.ContactID, &l.OwnerUserID, &l.FullName, &l.Email, &l.Phone, &l.CompanyName, &l.Status, &l.Score, &l.ScoreTier, &utmBytes, &cfBytes, &l.LastContactedAt, &l.NextFollowupAt, &l.ConvertedAt, &l.LostReason, &l.CreatedAt, &l.UpdatedAt); err != nil {
			continue
		}
		if utmBytes != nil {
			json.Unmarshal(utmBytes, &l.UTM)
		}
		if cfBytes != nil {
			json.Unmarshal(cfBytes, &l.CustomFields)
		}
		leads = append(leads, l)
	}

	return s.json(c, http.StatusOK, map[string]any{"leads": leads, "source": source})
}

// =============================================================================
// Lead Stats
// =============================================================================

func (s *Server) GetLeadStats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	// Status counts
	statusQuery := `SELECT status, COUNT(*) FROM leads WHERE deleted_at IS NULL GROUP BY status`
	statusRows, err := s.pool.Query(ctx, statusQuery)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer statusRows.Close()

	statusCounts := map[string]int{}
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err == nil {
			statusCounts[status] = count
		}
	}

	// Total
	var total int64
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM leads WHERE deleted_at IS NULL`).Scan(&total)

	// Score distribution
	var coldCount, warmCount, hotCount int64
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM leads WHERE score_tier = 'cold' AND deleted_at IS NULL`).Scan(&coldCount)
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM leads WHERE score_tier = 'warm' AND deleted_at IS NULL`).Scan(&warmCount)
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM leads WHERE score_tier = 'hot' AND deleted_at IS NULL`).Scan(&hotCount)

	return s.json(c, http.StatusOK, map[string]any{
		"total":       total,
		"by_status":   statusCounts,
		"by_score_tier": map[string]int64{
			"cold": coldCount,
			"warm": warmCount,
			"hot":  hotCount,
		},
	})
}

// =============================================================================
// Lead Import/Export
// =============================================================================

type importLeadRow struct {
	FullName    string
	Email       string
	Phone       string
	CompanyName string
	Status      string
}

func (s *Server) ImportLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	file, _, err := c.Request().FormFile("file")
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "file required", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid CSV", err)
	}

	// Map headers to indices
	colMap := map[string]int{}
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var imported, failed int
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			failed++
			continue
		}

		fullName := getCol(row, colMap, "full_name", "name")
		if fullName == "" {
			failed++
			continue
		}

		email := getCol(row, colMap, "email")
		phone := getCol(row, colMap, "phone", "mobile")
		companyName := getCol(row, colMap, "company", "company_name", "organization")
		status := getCol(row, colMap, "status")
		if status == "" {
			status = "new"
		}

		_, err = s.pool.Exec(ctx, `
			INSERT INTO leads (tenant_id, full_name, email, phone, company_name, status)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, tenantID, fullName, email, phone, companyName, status)

		if err != nil {
			failed++
			slog.Warn("import row failed", slog.String("error", err.Error()))
		} else {
			imported++
		}
	}

	return s.json(c, http.StatusOK, map[string]int{"imported": imported, "failed": failed})
}

func getCol(row []string, colMap map[string]int, keys ...string) string {
	for _, key := range keys {
		if idx, ok := colMap[key]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
	}
	return ""
}

func (s *Server) ExportLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT full_name, email, phone, company_name, status, score, score_tier, created_at 
		FROM leads WHERE deleted_at IS NULL ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=leads.csv")
	c.Response().Write([]byte("full_name,email,phone,company_name,status,score,score_tier,created_at\n"))

	for rows.Next() {
		var fullName, email, phone, companyName, status, scoreTier string
		var score float64
		var createdAt time.Time
		if err := rows.Scan(&fullName, &email, &phone, &companyName, &status, &score, &scoreTier, &createdAt); err != nil {
			continue
		}
		line := fmt.Sprintf("%s,%s,%s,%s,%s,%.2f,%s,%s\n",
			escapeCSV(fullName), escapeCSV(email), escapeCSV(phone), escapeCSV(companyName),
			status, score, scoreTier, createdAt.Format("2006-01-02 15:04:05"))
		c.Response().Write([]byte(line))
	}

	return nil
}

func escapeCSV(s string) string {
	s = strings.ReplaceAll(s, "\"", "\"\"")
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + s + "\""
	}
	return s
}

// =============================================================================
// Lead Sources
// =============================================================================

type sourceReq struct {
	Name        *string `json:"name,omitempty"`
	UTMSource   *string `json:"utm_source,omitempty"`
	UTMMedium   *string `json:"utm_medium,omitempty"`
	UTMCampaign *string `json:"utm_campaign,omitempty"`
	Description *string `json:"description,omitempty"`
}

type sourceResp struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	UTMSource   string    `json:"utm_source,omitempty"`
	UTMMedium   string    `json:"utm_medium,omitempty"`
	UTMCampaign string    `json:"utm_campaign,omitempty"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Server) ListLeadSources(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT id, tenant_id, name, utm_source, utm_medium, utm_campaign, description, is_active, created_at
		FROM lead_sources WHERE is_active = true ORDER BY name`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var sources []sourceResp
	for rows.Next() {
		var src sourceResp
		if err := rows.Scan(&src.ID, &src.TenantID, &src.Name, &src.UTMSource, &src.UTMMedium, &src.UTMCampaign, &src.Description, &src.IsActive, &src.CreatedAt); err != nil {
			continue
		}
		sources = append(sources, src)
	}

	return s.json(c, http.StatusOK, map[string]any{"sources": sources})
}

func (s *Server) CreateLeadSource(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req sourceReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Name == nil || *req.Name == "" {
		return s.errorResp(c, http.StatusBadRequest, "name is required", nil)
	}

	var resp sourceResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO lead_sources (tenant_id, name, utm_source, utm_medium, utm_campaign, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, name, utm_source, utm_medium, utm_campaign, description, is_active, created_at
	`, tenantID, *req.Name, req.UTMSource, req.UTMMedium, req.UTMCampaign, req.Description).
		Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.UTMSource, &resp.UTMMedium, &resp.UTMCampaign, &resp.Description, &resp.IsActive, &resp.CreatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

// =============================================================================
// Pipelines
// =============================================================================

type pipelineReq struct {
	Name      *string `json:"name,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
}

type pipelineResp struct {
	ID        uuid.UUID              `json:"id"`
	TenantID  uuid.UUID             `json:"tenant_id"`
	Name      string                `json:"name"`
	IsDefault bool                  `json:"is_default"`
	Stages    []pipelineStageResp   `json:"stages,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

type pipelineStageResp struct {
	ID           uuid.UUID `json:"id"`
	PipelineID   uuid.UUID `json:"pipeline_id"`
	Name         string    `json:"name"`
	DisplayOrder int       `json:"display_order"`
	Probability  int       `json:"probability"`
	Color       string    `json:"color"`
	IsDefault   bool      `json:"is_default"`
}

func (s *Server) ListPipelines(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT id, tenant_id, name, is_default, created_at FROM pipelines ORDER BY is_default DESC, name`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var pipelines []pipelineResp
	for rows.Next() {
		var p pipelineResp
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.IsDefault, &p.CreatedAt); err != nil {
			continue
		}
		pipelines = append(pipelines, p)
	}

	return s.json(c, http.StatusOK, map[string]any{"pipelines": pipelines})
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

	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	var resp pipelineResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO pipelines (tenant_id, name, is_default)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, name, is_default, created_at
	`, tenantID, *req.Name, isDefault).
		Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.IsDefault, &resp.CreatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

// =============================================================================
// Pipeline Stages
// =============================================================================

type stageReq struct {
	Name         *string `json:"name,omitempty"`
	DisplayOrder *int    `json:"display_order,omitempty"`
	Probability  *int    `json:"probability,omitempty"`
	Color        *string `json:"color,omitempty"`
}

func (s *Server) ListStages(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	pipelineID := c.QueryParam("pipeline_id")
	var query string
	var args []any

	if pipelineID != "" {
		query = `SELECT id, pipeline_id, name, display_order, probability, color, is_default FROM pipeline_stages WHERE pipeline_id = $1 ORDER BY display_order`
		args = []any{pipelineID}
	} else {
		query = `SELECT ps.id, ps.pipeline_id, ps.name, ps.display_order, ps.probability, ps.color, ps.is_default 
			FROM pipeline_stages ps JOIN pipelines p ON ps.pipeline_id = p.id 
			WHERE p.is_default = true ORDER BY ps.display_order`
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var stages []pipelineStageResp
	for rows.Next() {
		var st pipelineStageResp
		if err := rows.Scan(&st.ID, &st.PipelineID, &st.Name, &st.DisplayOrder, &st.Probability, &st.Color, &st.IsDefault); err != nil {
			continue
		}
		stages = append(stages, st)
	}

	return s.json(c, http.StatusOK, map[string]any{"stages": stages})
}

func (s *Server) CreateStage(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req struct {
		PipelineID  *uuid.UUID `json:"pipeline_id"`
		Name        *string   `json:"name"`
		DisplayOrder *int      `json:"display_order"`
		Probability  *int       `json:"probability"`
		Color       *string   `json:"color"`
	}
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.PipelineID == nil || req.Name == nil {
		return s.errorResp(c, http.StatusBadRequest, "pipeline_id and name are required", nil)
	}

	prob := 0
	if req.Probability != nil {
		prob = *req.Probability
	}
	color := "#6366f1"
	if req.Color != nil {
		color = *req.Color
	}
	order := 0
	if req.DisplayOrder != nil {
		order = *req.DisplayOrder
	}

	var resp pipelineStageResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO pipeline_stages (pipeline_id, name, display_order, probability, color)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, pipeline_id, name, display_order, probability, color, is_default
	`, req.PipelineID, *req.Name, order, prob, color).
		Scan(&resp.ID, &resp.PipelineID, &resp.Name, &resp.DisplayOrder, &resp.Probability, &resp.Color, &resp.IsDefault)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}
