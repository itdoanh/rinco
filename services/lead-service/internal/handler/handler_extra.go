// Package handler provides additional HTTP handlers for lead service.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// =============================================================================
// Lead Notes Handlers
// =============================================================================

type leadNoteReq struct {
	Body     *string `json:"body,omitempty"`
	IsPinned *bool   `json:"is_pinned,omitempty"`
}

type leadNoteResp struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	LeadID    uuid.UUID  `json:"lead_id"`
	AuthorID  *uuid.UUID `json:"author_id,omitempty"`
	Body      string     `json:"body"`
	IsPinned  bool       `json:"is_pinned"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
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
		return s.errorResp(c, http.StatusBadRequest, "invalid lead_id", err)
	}

	query := `SELECT id, tenant_id, lead_id, author_id, body, is_pinned, created_at, updated_at FROM lead_notes WHERE lead_id = $1 AND deleted_at IS NULL ORDER BY is_pinned DESC, created_at DESC`

	rows, err := s.pool.Query(ctx, query, leadID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var notes []leadNoteResp
	for rows.Next() {
		var n leadNoteResp
		if err := rows.Scan(&n.ID, &n.TenantID, &n.LeadID, &n.AuthorID, &n.Body, &n.IsPinned, &n.CreatedAt, &n.UpdatedAt); err != nil {
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
	parsedLeadID, err := uuid.Parse(leadID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid lead_id", err)
	}

	var req leadNoteReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Body == nil || *req.Body == "" {
		return s.errorResp(c, http.StatusBadRequest, "body is required", nil)
	}

	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}

	authorID, _ := uuid.Parse(userID)
	var resp leadNoteResp
	err = s.pool.QueryRow(ctx, `
		INSERT INTO lead_notes (tenant_id, lead_id, author_id, body, is_pinned)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, lead_id, author_id, body, is_pinned, created_at, updated_at
	`, tenantID, parsedLeadID, authorID, *req.Body, isPinned).
		Scan(&resp.ID, &resp.TenantID, &resp.LeadID, &resp.AuthorID, &resp.Body, &resp.IsPinned, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

// =============================================================================
// Lead Source Handlers (UTM Templates)
// =============================================================================

type leadSourceReq struct {
	Name        *string `json:"name,omitempty"`
	UTMSource   *string `json:"utm_source,omitempty"`
	UTMMedium   *string `json:"utm_medium,omitempty"`
	UTMCampaign *string `json:"utm_campaign,omitempty"`
	UTMTerm     *string `json:"utm_term,omitempty"`
	UTMContent  *string `json:"utm_content,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type leadSourceResp struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	UTMSource   string    `json:"utm_source,omitempty"`
	UTMMedium   string    `json:"utm_medium,omitempty"`
	UTMCampaign string    `json:"utm_campaign,omitempty"`
	UTMTerm     string    `json:"utm_term,omitempty"`
	UTMContent  string    `json:"utm_content,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *Server) ListLeadSources(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT id, tenant_id, name, utm_source, utm_medium, utm_campaign, utm_term, utm_content, is_active, created_at, updated_at FROM lead_sources WHERE deleted_at IS NULL ORDER BY name`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var sources []leadSourceResp
	for rows.Next() {
		var src leadSourceResp
		if err := rows.Scan(&src.ID, &src.TenantID, &src.Name, &src.UTMSource, &src.UTMMedium, &src.UTMCampaign, &src.UTMTerm, &src.UTMContent, &src.IsActive, &src.CreatedAt, &src.UpdatedAt); err != nil {
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

	var req leadSourceReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Name == nil || *req.Name == "" {
		return s.errorResp(c, http.StatusBadRequest, "name is required", nil)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var resp leadSourceResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO lead_sources (tenant_id, name, utm_source, utm_medium, utm_campaign, utm_term, utm_content, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, name, utm_source, utm_medium, utm_campaign, utm_term, utm_content, is_active, created_at, updated_at
	`, tenantID, *req.Name, req.UTMSource, req.UTMMedium, req.UTMCampaign, req.UTMTerm, req.UTMContent, isActive).
		Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.UTMSource, &resp.UTMMedium, &resp.UTMCampaign, &resp.UTMTerm, &resp.UTMContent, &resp.IsActive, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

// =============================================================================
// Pipeline Handlers
// =============================================================================

type pipelineReq struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsDefault   *bool   `json:"is_default,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type pipelineResp struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsDefault   bool      `json:"is_default"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *Server) ListPipelines(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT id, tenant_id, name, description, is_default, color, created_at, updated_at FROM pipelines WHERE deleted_at IS NULL ORDER BY is_default DESC, name`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var pipelines []pipelineResp
	for rows.Next() {
		var p pipelineResp
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.IsDefault, &p.Color, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
	color := "#6366f1"
	if req.Color != nil {
		color = *req.Color
	}

	// If setting as default, unset others
	if isDefault {
		_, _ = s.pool.Exec(ctx, `UPDATE pipelines SET is_default = false WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID)
	}

	var resp pipelineResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO pipelines (tenant_id, name, description, is_default, color)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, name, description, is_default, color, created_at, updated_at
	`, tenantID, *req.Name, req.Description, isDefault, color).
		Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.Description, &resp.IsDefault, &resp.Color, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

// =============================================================================
// Pipeline Stage Handlers
// =============================================================================

type stageReq struct {
	PipelineID   *uuid.UUID `json:"pipeline_id,omitempty"`
	Name         *string    `json:"name,omitempty"`
	DisplayOrder *int       `json:"display_order,omitempty"`
	Probability  *int       `json:"probability,omitempty"`
	Color        *string    `json:"color,omitempty"`
	IsWon        *bool      `json:"is_won,omitempty"`
	IsLost       *bool      `json:"is_lost,omitempty"`
}

type stageResp struct {
	ID           uuid.UUID `json:"id"`
	PipelineID   uuid.UUID `json:"pipeline_id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	DisplayOrder int       `json:"display_order"`
	Probability  int       `json:"probability"`
	Color        string    `json:"color"`
	IsWon        bool      `json:"is_won"`
	IsLost       bool      `json:"is_lost"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *Server) ListStages(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT id, pipeline_id, tenant_id, name, display_order, probability, color, is_won, is_lost, created_at, updated_at FROM pipeline_stages ORDER BY pipeline_id, display_order`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var stages []stageResp
	for rows.Next() {
		var st stageResp
		if err := rows.Scan(&st.ID, &st.PipelineID, &st.TenantID, &st.Name, &st.DisplayOrder, &st.Probability, &st.Color, &st.IsWon, &st.IsLost, &st.CreatedAt, &st.UpdatedAt); err != nil {
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

	var req stageReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.PipelineID == nil || req.Name == nil {
		return s.errorResp(c, http.StatusBadRequest, "pipeline_id and name are required", nil)
	}

	displayOrder := 0
	if req.DisplayOrder != nil {
		displayOrder = *req.DisplayOrder
	}
	probability := 50
	if req.Probability != nil {
		probability = *req.Probability
	}
	color := "#6366f1"
	if req.Color != nil {
		color = *req.Color
	}
	isWon := false
	if req.IsWon != nil {
		isWon = *req.IsWon
	}
	isLost := false
	if req.IsLost != nil {
		isLost = *req.IsLost
	}

	var resp stageResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO pipeline_stages (pipeline_id, tenant_id, name, display_order, probability, color, is_won, is_lost)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, pipeline_id, tenant_id, name, display_order, probability, color, is_won, is_lost, created_at, updated_at
	`, *req.PipelineID, tenantID, *req.Name, displayOrder, probability, color, isWon, isLost).
		Scan(&resp.ID, &resp.PipelineID, &resp.TenantID, &resp.Name, &resp.DisplayOrder, &resp.Probability, &resp.Color, &resp.IsWon, &resp.IsLost, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

// =============================================================================
// Convert Lead
// =============================================================================

type convertReq struct {
	CreateContact bool        `json:"create_contact"`
	ContactID     *uuid.UUID  `json:"contact_id,omitempty"`
	DeadlineDays  int         `json:"deadline_days,omitempty"`
	DealValue     *float64    `json:"deal_value,omitempty"`
}

func (s *Server) ConvertLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 15*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	parsedLeadID, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req convertReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	// Update lead to won status
	var resp leadResp
	var cf, utm []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE leads SET 
			status = 'won',
			contact_id = COALESCE($2, contact_id),
			converted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, source_id, contact_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, tags, next_followup_at, last_contacted_at, converted_at, lost_reason, created_at, updated_at
	`, parsedLeadID, req.ContactID).
		Scan(&resp.ID, &resp.TenantID, &resp.SourceID, &resp.ContactID, &resp.OwnerUserID, &resp.PipelineID, &resp.StageID, &resp.FullName, &resp.Email, &resp.Phone, &resp.CompanyName, &resp.JobTitle, &resp.Status, &resp.Score, &resp.ScoreTier, &resp.EstimatedValue, &cf, &utm, &resp.Tags, &resp.NextFollowupAt, &resp.LastContactedAt, &resp.ConvertedAt, &resp.LostReason, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "lead not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "convert failed", err)
	}

	// Log activity
	actorID, _ := uuid.Parse(userID)
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO lead_activities (tenant_id, lead_id, type, actor_id, description, payload)
		VALUES ($1, $2, 'STATUS_CHANGE', $3, $4, $5)
	`, tenantID, parsedLeadID, actorID, "Lead converted to won", "{}")

	// Publish event
	if s.nats != nil {
		go s.nats.Publish(ctx, "lead.converted", map[string]any{
			"lead_id":   parsedLeadID,
			"tenant_id": tenantID,
			"contact_id": req.ContactID,
		})
	}

	if cf != nil {
		// simplified
	}
	if utm != nil {
		// simplified
	}
	_ = utm

	return s.json(c, http.StatusOK, map[string]any{
		"lead_id":     resp.ID,
		"status":      "converted",
		"contact_id":  req.ContactID,
		"converted_at": resp.ConvertedAt,
	})
}

// =============================================================================
// Helper to subscribe to NATS events
// =============================================================================

func (s *Server) SubscribeEvents(ctx context.Context) error {
	if s.nats == nil {
		return nil
	}

	// Subscribe to user.created for auto-assignment logic
	_, err := s.nats.Subscribe("user.created", func(data []byte) {
		slog.Info("user.created event received", slog.Int("bytes", len(data)))
		// Auto-assign leads in pool: distribute unassigned leads to new users
	})

	if err != nil {
		return fmt.Errorf("subscribe user.created: %w", err)
	}

	return nil
}
