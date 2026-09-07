// Package handler provides tree-related HTTP handlers for CRM service with LTREE.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
// User Tree (LTREE) Handlers
// =============================================================================

type userReq struct {
	Email      *string `json:"email,omitempty"`
	FullName   *string `json:"full_name,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Role       *string `json:"role,omitempty"`
	Status     *string `json:"status,omitempty"`
	ParentID   *string `json:"parent_id,omitempty"`
	Department *string `json:"department,omitempty"`
	Position   *string `json:"position,omitempty"`
}

type userResp struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Email      string    `json:"email"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone,omitempty"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	Path       string    `json:"path"`
	Depth      int       `json:"depth"`
	ParentID   *string   `json:"parent_id,omitempty"`
	Department string    `json:"department,omitempty"`
	Position   string    `json:"position,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Server) ListTreeUsers(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		tenantID, _, _ = s.tenantFromCtx(c)
	}
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT id, tenant_id, email, full_name, phone, role, status, path::text, depth, parent_id, department, position, created_at, updated_at
		FROM users WHERE deleted_at IS NULL ORDER BY path`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var users []userResp
	for rows.Next() {
		var u userResp
		var parentID *string
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Phone, &u.Role, &u.Status, &u.Path, &u.Depth, &parentID, &u.Department, &u.Position, &u.CreatedAt, &u.UpdatedAt); err != nil {
			continue
		}
		u.ParentID = parentID
		users = append(users, u)
	}

	return s.json(c, http.StatusOK, map[string]any{"users": users})
}

func (s *Server) CreateTreeUser(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		tenantID, _, _ = s.tenantFromCtx(c)
	}
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req userReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Email == nil || *req.Email == "" || req.FullName == nil || *req.FullName == "" {
		return s.errorResp(c, http.StatusBadRequest, "email and full_name are required", nil)
	}

	// Generate slug from email
	slug := strings.ToLower(strings.Split(*req.Email, "@")[0])
	slug = strings.ReplaceAll(slug, ".", "_")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1
	}, slug)

	role := "member"
	if req.Role != nil {
		role = *req.Role
	}
	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	var parentPath string
	var parentDepth int
	var parentID *string

	if req.ParentID != nil && *req.ParentID != "" {
		err := s.pool.QueryRow(ctx, `SELECT path::text, depth FROM users WHERE id = $1 AND deleted_at IS NULL`, *req.ParentID).
			Scan(&parentPath, &parentDepth)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return s.errorResp(c, http.StatusInternalServerError, "get parent failed", err)
		}
		if parentPath == "" {
			parentPath = "root"
			parentDepth = 0
		}
		parentID = req.ParentID
	} else {
		// Check if this is the first user (root)
		var count int
		s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID).Scan(&count)
		if count == 0 {
			parentPath = "root"
			parentDepth = 0
		} else {
			parentPath = "root"
			parentDepth = 0
		}
	}

	newPath := parentPath + "." + slug
	newDepth := parentDepth + 1

	var resp userResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, full_name, phone, role, status, path, depth, parent_id, department, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7::ltree, $8, $9, $10, $11)
		RETURNING id, tenant_id, email, full_name, phone, role, status, path::text, depth, parent_id, department, position, created_at, updated_at
	`, tenantID, *req.Email, *req.FullName, req.Phone, role, status, newPath, newDepth, parentID, req.Department, req.Position).
		Scan(&resp.ID, &resp.TenantID, &resp.Email, &resp.FullName, &resp.Phone, &resp.Role, &resp.Status, &resp.Path, &resp.Depth, &resp.ParentID, &resp.Department, &resp.Position, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		slog.Error("create tree user failed", slog.String("error", err.Error()))
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetUserTree(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var resp userResp
	var parentID *string
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, full_name, phone, role, status, path::text, depth, parent_id, department, position, created_at, updated_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.Email, &resp.FullName, &resp.Phone, &resp.Role, &resp.Status, &resp.Path, &resp.Depth, &parentID, &resp.Department, &resp.Position, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "user not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	resp.ParentID = parentID

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateTreeUser(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	currentUserID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, currentUserID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var req userReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var resp userResp
	var parentID *string
	err = s.pool.QueryRow(ctx, `
		UPDATE users SET 
			email = COALESCE($2, email),
			full_name = COALESCE($3, full_name),
			phone = COALESCE($4, phone),
			role = COALESCE($5, role),
			status = COALESCE($6, status),
			department = COALESCE($7, department),
			position = COALESCE($8, position),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, email, full_name, phone, role, status, path::text, depth, parent_id, department, position, created_at, updated_at
	`, id, req.Email, req.FullName, req.Phone, req.Role, req.Status, req.Department, req.Position).
		Scan(&resp.ID, &resp.TenantID, &resp.Email, &resp.FullName, &resp.Phone, &resp.Role, &resp.Status, &resp.Path, &resp.Depth, &parentID, &resp.Department, &resp.Position, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "user not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	resp.ParentID = parentID

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteTreeUser(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	// Check if user has children
	var childCount int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE parent_id = $1 AND deleted_at IS NULL`, id).Scan(&childCount)
	if childCount > 0 {
		return s.errorResp(c, http.StatusConflict, "cannot delete user with subordinates", nil)
	}

	result, err := s.pool.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if result.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "user not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

type moveSubtreeReq struct {
	SourceUserID  string `json:"source_user_id"`
	TargetParentID string `json:"target_parent_id"`
	Reason        string `json:"reason,omitempty"`
}

func (s *Server) MoveSubtree(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req moveSubtreeReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.SourceUserID == req.TargetParentID {
		return s.errorResp(c, http.StatusBadRequest, "cannot move user to itself", nil)
	}

	// Get source path
	var oldPath string
	err := s.pool.QueryRow(ctx, `SELECT path::text FROM users WHERE id = $1 AND deleted_at IS NULL`, req.SourceUserID).Scan(&oldPath)
	if err != nil {
		return s.errorResp(c, http.StatusNotFound, "source user not found", err)
	}

	// Get target parent path
	var newParentPath string
	var newParentDepth int
	err = s.pool.QueryRow(ctx, `SELECT path::text, depth FROM users WHERE id = $1 AND deleted_at IS NULL`, req.TargetParentID).Scan(&newParentPath, &newParentDepth)
	if err != nil {
		return s.errorResp(c, http.StatusNotFound, "target parent not found", err)
	}

	// Check for circular move (target is descendant of source)
	var descendantCount int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE path::text <@ $1::ltree AND id = $2`, oldPath, req.TargetParentID).Scan(&descendantCount)
	if descendantCount > 0 {
		return s.errorResp(c, http.StatusBadRequest, "cannot move parent into its own subtree", nil)
	}

	// Calculate new path for source
	lastSegment := oldPath
	if idx := strings.LastIndex(oldPath, "."); idx >= 0 {
		lastSegment = oldPath[idx+1:]
	}
	newPath := newParentPath + "." + lastSegment
	newDepth := newParentDepth + 1

	// Update source user
	_, err = s.pool.Exec(ctx, `
		UPDATE users SET 
			path = $1::ltree,
			depth = $2,
			parent_id = $3,
			updated_at = NOW()
		WHERE id = $4
	`, newPath, newDepth, req.TargetParentID, req.SourceUserID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "move source failed", err)
	}

	// Update all descendants
	_, err = s.pool.Exec(ctx, `
		UPDATE users SET
			path = $1::ltree || subpath(path, nlevel($2::ltree)),
			depth = nlevel(path),
			updated_at = NOW()
		WHERE path <@ $2::ltree AND id != $3
	`, newPath, oldPath, req.SourceUserID)
	if err != nil {
		slog.Warn("update descendants path failed", slog.String("error", err.Error()))
	}

	// Refresh materialized view
	_, _ = s.pool.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY users_with_depth`)

	return s.json(c, http.StatusOK, map[string]any{
		"status":       "moved",
		"old_path":     oldPath,
		"new_path":     newPath,
		"new_depth":    newDepth,
		"source_id":    req.SourceUserID,
		"target_id":    req.TargetParentID,
	})
}

func (s *Server) GetUserPath(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("user_id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid user_id", err)
	}

	var path string
	err = s.pool.QueryRow(ctx, `SELECT path::text FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&path)
	if err != nil {
		return s.errorResp(c, http.StatusNotFound, "user not found", err)
	}

	// Get full path with user names
	rows, err := s.pool.Query(ctx, `
		SELECT full_name, role FROM users 
		WHERE path @> (SELECT path FROM users WHERE id = $1) AND deleted_at IS NULL 
		ORDER BY nlevel(path)
	`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "path query failed", err)
	}
	defer rows.Close()

	var pathParts []map[string]string
	for rows.Next() {
		var name, role string
		if err := rows.Scan(&name, &role); err != nil {
			continue
		}
		pathParts = append(pathParts, map[string]string{"name": name, "role": role})
	}

	return s.json(c, http.StatusOK, map[string]any{
		"user_id": id,
		"path":    path,
		"path_parts": pathParts,
	})
}

func (s *Server) GetSubordinates(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("user_id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid user_id", err)
	}

	// Get user's path
	var userPath string
	err = s.pool.QueryRow(ctx, `SELECT path::text FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&userPath)
	if err != nil {
		return s.errorResp(c, http.StatusNotFound, "user not found", err)
	}

	// Get all subordinates (recursive)
	query := `SELECT id, tenant_id, email, full_name, phone, role, status, path::text, depth, parent_id, department, position, created_at, updated_at
		FROM users WHERE path <@ $1::ltree AND id != $2 AND deleted_at IS NULL ORDER BY depth, full_name`

	rows, err := s.pool.Query(ctx, query, userPath, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var users []userResp
	for rows.Next() {
		var u userResp
		var parentID *string
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Phone, &u.Role, &u.Status, &u.Path, &u.Depth, &parentID, &u.Department, &u.Position, &u.CreatedAt, &u.UpdatedAt); err != nil {
			continue
		}
		u.ParentID = parentID
		users = append(users, u)
	}

	return s.json(c, http.StatusOK, map[string]any{"subordinates": users, "count": len(users)})
}

func (s *Server) GetAncestors(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("user_id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid user_id", err)
	}

	query := `SELECT id, tenant_id, email, full_name, phone, role, status, path::text, depth, parent_id, department, position, created_at, updated_at
		FROM users WHERE path @> (SELECT path FROM users WHERE id = $1) AND deleted_at IS NULL ORDER BY depth ASC`

	rows, err := s.pool.Query(ctx, query, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var users []userResp
	for rows.Next() {
		var u userResp
		var parentID *string
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Phone, &u.Role, &u.Status, &u.Path, &u.Depth, &parentID, &u.Department, &u.Position, &u.CreatedAt, &u.UpdatedAt); err != nil {
			continue
		}
		u.ParentID = parentID
		users = append(users, u)
	}

	return s.json(c, http.StatusOK, map[string]any{"ancestors": users, "count": len(users)})
}

type createInviteLinkReq struct {
	TargetRole  string `json:"target_role"`
	MaxUses     int    `json:"max_uses"`
	ExpiresInDays int  `json:"expires_in_days"`
}

type inviteLinkResp struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Token       string     `json:"token"`
	TargetRole  string     `json:"target_role"`
	MaxUses     int        `json:"max_uses"`
	CurrentUses int        `json:"current_uses"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (s *Server) CreateInviteLink(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req createInviteLinkReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.TargetRole == "" {
		return s.errorResp(c, http.StatusBadRequest, "target_role is required", nil)
	}

	// Generate secure token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	maxUses := 1
	if req.MaxUses > 0 {
		maxUses = req.MaxUses
	}
	expiresInDays := 7
	if req.ExpiresInDays > 0 {
		expiresInDays = req.ExpiresInDays
	}
	expiresAt := time.Now().AddDate(0, 0, expiresInDays)

	parsedUserID, _ := uuid.Parse(userID)
	var resp inviteLinkResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO invite_links (tenant_id, parent_user_id, target_role, token, max_uses, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, token, target_role, max_uses, current_uses, expires_at, created_at
	`, tenantID, parsedUserID, req.TargetRole, token, maxUses, expiresAt, parsedUserID).
		Scan(&resp.ID, &resp.TenantID, &resp.Token, &resp.TargetRole, &resp.MaxUses, &resp.CurrentUses, &resp.ExpiresAt, &resp.CreatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create invite link failed", err)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetSubordinatesCount(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID := c.Param("tenant_id")
	userID, _ := s.GetUserID(c)
	isAdmin := s.IsAdmin(c)

	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}

	var userPath string
	err = s.pool.QueryRow(ctx, `SELECT path::text FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&userPath)
	if err != nil {
		return s.errorResp(c, http.StatusNotFound, "user not found", err)
	}

	var directReports, totalSubordinates int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE parent_id = $1 AND deleted_at IS NULL`, id).Scan(&directReports)
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE path <@ $1::ltree AND id != $2 AND deleted_at IS NULL`, userPath, id).Scan(&totalSubordinates)

	return s.json(c, http.StatusOK, map[string]any{
		"user_id":          id,
		"direct_reports":   directReports,
		"total_subordinates": totalSubordinates,
	})
}

// =============================================================================
// Reports Handlers
// =============================================================================

func (s *Server) GetPipelineReport(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `SELECT 
		stage,
		COUNT(*) as deal_count,
		COALESCE(SUM(value), 0) as total_value,
		COALESCE(AVG(value), 0) as avg_value,
		COALESCE(AVG(probability), 0) as avg_probability
	FROM deals 
	WHERE deleted_at IS NULL AND stage NOT IN ('won', 'lost')
	GROUP BY stage
	ORDER BY 
		CASE stage 
			WHEN 'prospecting' THEN 1 
			WHEN 'qualification' THEN 2 
			WHEN 'proposal' THEN 3 
			WHEN 'negotiation' THEN 4 
			ELSE 5 
		END`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var stages []map[string]any
	for rows.Next() {
		var stage string
		var count int
		var total, avg, prob float64
		if err := rows.Scan(&stage, &count, &total, &avg, &prob); err != nil {
			continue
		}
		stages = append(stages, map[string]any{
			"stage":            stage,
			"deal_count":       count,
			"total_value":      total,
			"avg_value":        avg,
			"avg_probability":  prob,
		})
	}

	return s.json(c, http.StatusOK, map[string]any{"pipeline": stages})
}

func (s *Server) GetConversionReport(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	query := `WITH total AS (
		SELECT COUNT(*) as total_leads FROM contacts WHERE deleted_at IS NULL
	),
	qualified AS (
		SELECT COUNT(*) as qualified FROM contacts WHERE status = 'active' AND deleted_at IS NULL
	)
	SELECT 
		(SELECT total_leads FROM total) as total,
		(SELECT qualified FROM qualified) as qualified,
		CASE WHEN (SELECT total_leads FROM total) > 0 
			THEN ROUND((SELECT qualified FROM qualified)::numeric / (SELECT total_leads FROM total) * 100, 2)
			ELSE 0 
		END as conversion_rate`

	var total, qualified int
	var rate float64
	err := s.pool.QueryRow(ctx, query).Scan(&total, &qualified, &rate)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}

	return s.json(c, http.StatusOK, map[string]any{
		"total_contacts":   total,
		"qualified":        qualified,
		"conversion_rate":   rate,
	})
}

func (s *Server) GetLeaderboard(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	query := `SELECT 
		u.id,
		u.full_name,
		u.role,
		COUNT(DISTINCT d.id) as deals_count,
		COALESCE(SUM(d.value), 0) as total_value,
		COUNT(DISTINCT a.id) as activities_count
	FROM users u
	LEFT JOIN deals d ON d.owner_user_id = u.id AND d.deleted_at IS NULL
	LEFT JOIN activities a ON a.owner_user_id = u.id AND a.deleted_at IS NULL
	WHERE u.deleted_at IS NULL
	GROUP BY u.id, u.full_name, u.role
	ORDER BY total_value DESC
	LIMIT $1`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var leaders []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var name, role string
		var dealsCount int
		var totalValue float64
		var activitiesCount int
		if err := rows.Scan(&id, &name, &role, &dealsCount, &totalValue, &activitiesCount); err != nil {
			continue
		}
		leaders = append(leaders, map[string]any{
			"user_id":           id,
			"full_name":         name,
			"role":              role,
			"deals_count":       dealsCount,
			"total_value":       totalValue,
			"activities_count":   activitiesCount,
		})
	}

	return s.json(c, http.StatusOK, map[string]any{"leaderboard": leaders})
}

// Helper methods
func (s *Server) GetUserID(c echo.Context) (string, bool) {
	userID, _ := c.Get("user_id").(string)
	return userID, userID != ""
}

func (s *Server) IsAdmin(c echo.Context) bool {
	isAdmin, _ := c.Get("is_admin").(bool)
	return isAdmin
}
