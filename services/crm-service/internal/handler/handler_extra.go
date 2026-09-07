// Package handler provides additional HTTP handlers for CRM service.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// =============================================================================
// Activities Handlers
// =============================================================================

type activityReq struct {
	ContactID   *uuid.UUID    `json:"contact_id,omitempty"`
	DealID      *uuid.UUID    `json:"deal_id,omitempty"`
	Type        *string       `json:"type,omitempty"`
	Subject     *string       `json:"subject,omitempty"`
	Body        *string       `json:"body,omitempty"`
	DueAt       *time.Time    `json:"due_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Status      *string       `json:"status,omitempty"`
	Priority    *string       `json:"priority,omitempty"`
	OwnerUserID *uuid.UUID    `json:"owner_user_id,omitempty"`
}

type activityResp struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ContactID   *uuid.UUID `json:"contact_id,omitempty"`
	DealID      *uuid.UUID `json:"deal_id,omitempty"`
	Type        string     `json:"type"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body,omitempty"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority,omitempty"`
	OwnerUserID *uuid.UUID `json:"owner_user_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (s *Server) ListActivities(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)

	var total int64
	countQuery := `SELECT COUNT(*) FROM activities WHERE deleted_at IS NULL`
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}

	query := `SELECT id, tenant_id, contact_id, deal_id, type, subject, body, due_at, completed_at, status, priority, owner_user_id, created_at, updated_at
		FROM activities WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var activities []activityResp
	for rows.Next() {
		var a activityResp
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ContactID, &a.DealID, &a.Type, &a.Subject, &a.Body, &a.DueAt, &a.CompletedAt, &a.Status, &a.Priority, &a.OwnerUserID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		activities = append(activities, a)
	}

	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}

	return s.json(c, http.StatusOK, listResp{Data: activities, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

func (s *Server) CreateActivity(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req activityReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Type == nil || req.Subject == nil {
		return s.errorResp(c, http.StatusBadRequest, "type and subject are required", nil)
	}

	status := "pending"
	if req.Status != nil {
		status = *req.Status
	}
	priority := "medium"
	if req.Priority != nil {
		priority = *req.Priority
	}

	var resp activityResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO activities (tenant_id, contact_id, deal_id, type, subject, body, due_at, completed_at, status, priority, owner_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, tenant_id, contact_id, deal_id, type, subject, body, due_at, completed_at, status, priority, owner_user_id, created_at, updated_at
	`, tenantID, req.ContactID, req.DealID, *req.Type, *req.Subject, req.Body, req.DueAt, req.CompletedAt, status, priority, req.OwnerUserID).
		Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.DealID, &resp.Type, &resp.Subject, &resp.Body, &resp.DueAt, &resp.CompletedAt, &resp.Status, &resp.Priority, &resp.OwnerUserID, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetActivity(c echo.Context) error {
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

	var resp activityResp
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, contact_id, deal_id, type, subject, body, due_at, completed_at, status, priority, owner_user_id, created_at, updated_at
		FROM activities WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.DealID, &resp.Type, &resp.Subject, &resp.Body, &resp.DueAt, &resp.CompletedAt, &resp.Status, &resp.Priority, &resp.OwnerUserID, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "activity not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateActivity(c echo.Context) error {
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

	var req activityReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var resp activityResp
	err = s.pool.QueryRow(ctx, `
		UPDATE activities SET 
			contact_id = COALESCE($2, contact_id),
			deal_id = COALESCE($3, deal_id),
			type = COALESCE($4, type),
			subject = COALESCE($5, subject),
			body = COALESCE($6, body),
			due_at = COALESCE($7, due_at),
			completed_at = COALESCE($8, completed_at),
			status = COALESCE($9, status),
			priority = COALESCE($10, priority),
			owner_user_id = COALESCE($11, owner_user_id),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, contact_id, deal_id, type, subject, body, due_at, completed_at, status, priority, owner_user_id, created_at, updated_at
	`, id, req.ContactID, req.DealID, req.Type, req.Subject, req.Body, req.DueAt, req.CompletedAt, req.Status, req.Priority, req.OwnerUserID).
		Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.DealID, &resp.Type, &resp.Subject, &resp.Body, &resp.DueAt, &resp.CompletedAt, &resp.Status, &resp.Priority, &resp.OwnerUserID, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "activity not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteActivity(c echo.Context) error {
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

	result, err := s.pool.Exec(ctx, `UPDATE activities SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if result.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "activity not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) GetContactTimeline(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	contactID := c.Param("contact_id")
	_, err := uuid.Parse(contactID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid contact_id", err)
	}

	query := `SELECT id, tenant_id, contact_id, deal_id, type, subject, body, due_at, completed_at, status, priority, owner_user_id, created_at, updated_at
		FROM activities WHERE contact_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 50`

	rows, err := s.pool.Query(ctx, query, contactID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var activities []activityResp
	for rows.Next() {
		var a activityResp
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ContactID, &a.DealID, &a.Type, &a.Subject, &a.Body, &a.DueAt, &a.CompletedAt, &a.Status, &a.Priority, &a.OwnerUserID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		activities = append(activities, a)
	}

	return s.json(c, http.StatusOK, map[string]any{"activities": activities})
}

// =============================================================================
// Notes Handlers
// =============================================================================

type noteReq struct {
	ContactID *uuid.UUID `json:"contact_id,omitempty"`
	DealID    *uuid.UUID `json:"deal_id,omitempty"`
	Body      *string    `json:"body,omitempty"`
	IsPinned  *bool      `json:"is_pinned,omitempty"`
}

type noteResp struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	ContactID *uuid.UUID `json:"contact_id,omitempty"`
	DealID    *uuid.UUID `json:"deal_id,omitempty"`
	Body      string     `json:"body"`
	AuthorID  *uuid.UUID `json:"author_id,omitempty"`
	IsPinned  bool       `json:"is_pinned"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (s *Server) ListNotes(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)

	var total int64
	countQuery := `SELECT COUNT(*) FROM notes WHERE deleted_at IS NULL`
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}

	query := `SELECT id, tenant_id, contact_id, deal_id, body, author_id, is_pinned, created_at, updated_at
		FROM notes WHERE deleted_at IS NULL ORDER BY is_pinned DESC, created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var notes []noteResp
	for rows.Next() {
		var n noteResp
		if err := rows.Scan(&n.ID, &n.TenantID, &n.ContactID, &n.DealID, &n.Body, &n.AuthorID, &n.IsPinned, &n.CreatedAt, &n.UpdatedAt); err != nil {
			continue
		}
		notes = append(notes, n)
	}

	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}

	return s.json(c, http.StatusOK, listResp{Data: notes, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

func (s *Server) CreateNote(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req noteReq
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
	var resp noteResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notes (tenant_id, contact_id, deal_id, body, author_id, is_pinned)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, contact_id, deal_id, body, author_id, is_pinned, created_at, updated_at
	`, tenantID, req.ContactID, req.DealID, *req.Body, authorID, isPinned).
		Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.DealID, &resp.Body, &resp.AuthorID, &resp.IsPinned, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetNote(c echo.Context) error {
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

	var resp noteResp
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, contact_id, deal_id, body, author_id, is_pinned, created_at, updated_at
		FROM notes WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.DealID, &resp.Body, &resp.AuthorID, &resp.IsPinned, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "note not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateNote(c echo.Context) error {
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

	var req noteReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var resp noteResp
	err = s.pool.QueryRow(ctx, `
		UPDATE notes SET 
			contact_id = COALESCE($2, contact_id),
			deal_id = COALESCE($3, deal_id),
			body = COALESCE($4, body),
			is_pinned = COALESCE($5, is_pinned),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, contact_id, deal_id, body, author_id, is_pinned, created_at, updated_at
	`, id, req.ContactID, req.DealID, req.Body, req.IsPinned).
		Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.DealID, &resp.Body, &resp.AuthorID, &resp.IsPinned, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "note not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteNote(c echo.Context) error {
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

	result, err := s.pool.Exec(ctx, `UPDATE notes SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if result.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "note not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// Tags Handlers
// =============================================================================

type tagReq struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

type tagResp struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description,omitempty"`
	UsageCount  int       `json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *Server) ListTags(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT id, tenant_id, name, color, description, usage_count, created_at, updated_at
		FROM tags WHERE deleted_at IS NULL ORDER BY usage_count DESC, name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var tags []tagResp
	for rows.Next() {
		var t tagResp
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Color, &t.Description, &t.UsageCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		tags = append(tags, t)
	}

	return s.json(c, http.StatusOK, map[string]any{"tags": tags})
}

func (s *Server) CreateTag(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req tagReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Name == nil || *req.Name == "" {
		return s.errorResp(c, http.StatusBadRequest, "name is required", nil)
	}

	color := "#6366f1"
	if req.Color != nil {
		color = *req.Color
	}

	var resp tagResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tags (tenant_id, name, color, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, name, color, description, usage_count, created_at, updated_at
	`, tenantID, *req.Name, color, req.Description).
		Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.Color, &resp.Description, &resp.UsageCount, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) AddTagToContact(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	contactID := c.Param("id")
	_, err := uuid.Parse(contactID)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid contact_id", err)
	}

	var req struct {
		TagID *uuid.UUID `json:"tag_id"`
		Name  *string    `json:"name"`
	}
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.TagID == nil && (req.Name == nil || *req.Name == "") {
		return s.errorResp(c, http.StatusBadRequest, "tag_id or name is required", nil)
	}

	var tagID uuid.UUID
	if req.TagID != nil {
		tagID = *req.TagID
	} else {
		// Create tag if not exists
		var newTag tagResp
		err = s.pool.QueryRow(ctx, `
			INSERT INTO tags (tenant_id, name)
			VALUES ($1, $2)
			ON CONFLICT (tenant_id, name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, tenantID, *req.Name).Scan(&tagID)
		if err != nil {
			return s.errorResp(c, http.StatusInternalServerError, "tag creation failed", err)
		}
		_ = newTag
	}

	// Add tag to contact
	_, err = s.pool.Exec(ctx, `
		INSERT INTO contact_tags (contact_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, contactID, tagID)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "add tag failed", err)
	}

	// Update usage count
	_, _ = s.pool.Exec(ctx, `
		UPDATE tags SET usage_count = (
			SELECT COUNT(*) FROM contact_tags WHERE tag_id = $1
		) WHERE id = $1
	`, tagID)

	return s.json(c, http.StatusOK, map[string]string{"status": "tag added"})
}

// =============================================================================
// Custom Fields Handlers
// =============================================================================

type customFieldReq struct {
	EntityType      *string         `json:"entity_type,omitempty"`
	Name            *string         `json:"name,omitempty"`
	FieldKey        *string         `json:"field_key,omitempty"`
	FieldType       *string         `json:"field_type,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Options         []string        `json:"options,omitempty"`
	DefaultValue    any              `json:"default_value,omitempty"`
	ValidationRules *map[string]any `json:"validation_rules,omitempty"`
	IsRequired      *bool           `json:"is_required,omitempty"`
	IsUnique        *bool           `json:"is_unique,omitempty"`
	DisplayOrder    *int            `json:"display_order,omitempty"`
	Width           *string         `json:"width,omitempty"`
}

type customFieldResp struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID     `json:"tenant_id"`
	EntityType      string        `json:"entity_type"`
	Name            string        `json:"name"`
	FieldKey        string        `json:"field_key"`
	FieldType       string        `json:"field_type"`
	Description     string        `json:"description,omitempty"`
	Options         []string      `json:"options,omitempty"`
	DefaultValue    any           `json:"default_value,omitempty"`
	ValidationRules map[string]any `json:"validation_rules,omitempty"`
	IsRequired      bool          `json:"is_required"`
	IsUnique        bool          `json:"is_unique"`
	DisplayOrder    int           `json:"display_order"`
	Width           string        `json:"width"`
	IsActive        bool          `json:"is_active"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

func (s *Server) ListCustomFields(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	entityType := c.QueryParam("entity_type")

	var query string
	var args []any

	if entityType != "" {
		query = `SELECT id, tenant_id, entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width, is_active, created_at, updated_at
			FROM custom_fields WHERE deleted_at IS NULL AND is_active = true AND entity_type = $1 ORDER BY display_order ASC`
		args = []any{entityType}
	} else {
		query = `SELECT id, tenant_id, entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width, is_active, created_at, updated_at
			FROM custom_fields WHERE deleted_at IS NULL AND is_active = true ORDER BY entity_type, display_order ASC`
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var fields []customFieldResp
	for rows.Next() {
		var cf customFieldResp
		var opts, defVal, valRules []byte
		if err := rows.Scan(&cf.ID, &cf.TenantID, &cf.EntityType, &cf.Name, &cf.FieldKey, &cf.FieldType, &cf.Description, &opts, &defVal, &valRules, &cf.IsRequired, &cf.IsUnique, &cf.DisplayOrder, &cf.Width, &cf.IsActive, &cf.CreatedAt, &cf.UpdatedAt); err != nil {
			continue
		}
		if opts != nil {
			json.Unmarshal(opts, &cf.Options)
		}
		if defVal != nil {
			json.Unmarshal(defVal, &cf.DefaultValue)
		}
		if valRules != nil {
			json.Unmarshal(valRules, &cf.ValidationRules)
		}
		fields = append(fields, cf)
	}

	return s.json(c, http.StatusOK, map[string]any{"custom_fields": fields})
}

func (s *Server) CreateCustomField(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req customFieldReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.EntityType == nil || req.Name == nil || req.FieldKey == nil || req.FieldType == nil {
		return s.errorResp(c, http.StatusBadRequest, "entity_type, name, field_key, and field_type are required", nil)
	}

	validTypes := map[string]bool{"text": true, "textarea": true, "number": true, "currency": true, "date": true, "datetime": true, "boolean": true, "select": true, "multiselect": true, "email": true, "phone": true, "url": true}
	if !validTypes[*req.FieldType] {
		return s.errorResp(c, http.StatusBadRequest, "invalid field_type", nil)
	}

	isRequired := false
	if req.IsRequired != nil {
		isRequired = *req.IsRequired
	}
	isUnique := false
	if req.IsUnique != nil {
		isUnique = *req.IsUnique
	}
	displayOrder := 0
	if req.DisplayOrder != nil {
		displayOrder = *req.DisplayOrder
	}
	width := "full"
	if req.Width != nil {
		width = *req.Width
	}

	optsBytes, _ := json.Marshal(req.Options)
	defBytes, _ := json.Marshal(req.DefaultValue)
	valBytes, _ := json.Marshal(req.ValidationRules)

	var resp customFieldResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO custom_fields (tenant_id, entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, tenant_id, entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width, is_active, created_at, updated_at
	`, tenantID, *req.EntityType, *req.Name, *req.FieldKey, *req.FieldType, req.Description, optsBytes, defBytes, valBytes, isRequired, isUnique, displayOrder, width).
		Scan(&resp.ID, &resp.TenantID, &resp.EntityType, &resp.Name, &resp.FieldKey, &resp.FieldType, &resp.Description, &optsBytes, &defBytes, &valBytes, &resp.IsRequired, &resp.IsUnique, &resp.DisplayOrder, &resp.Width, &resp.IsActive, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	if optsBytes != nil {
		json.Unmarshal(optsBytes, &resp.Options)
	}
	if defBytes != nil {
		json.Unmarshal(defBytes, &resp.DefaultValue)
	}
	if valBytes != nil {
		json.Unmarshal(valBytes, &resp.ValidationRules)
	}

	return s.json(c, http.StatusCreated, resp)
}
