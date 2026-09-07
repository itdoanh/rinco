// Package handler provides HTTP handlers for CRM service.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	rdb  *RedisClient
}

// RedisClient wraps go-redis client.
type RedisClient struct {
	Addr     string
	Password string
	DB       int
}

func NewServer(pool *pgxpool.Pool, rdb *RedisClient) *Server {
	return &Server{pool: pool, rdb: rdb}
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
// Companies Handlers
// =============================================================================

type companyReq struct {
	Name         *string                `json:"name,omitempty"`
	Industry     *string                `json:"industry,omitempty"`
	Size         *string                `json:"size,omitempty"`
	Website      *string                `json:"website,omitempty"`
	Phone        *string                `json:"phone,omitempty"`
	Email        *string                `json:"email,omitempty"`
	Address      *string                `json:"address,omitempty"`
	City         *string                `json:"city,omitempty"`
	Country      *string                `json:"country,omitempty"`
	CustomFields *map[string]any        `json:"custom_fields,omitempty"`
}

type companyResp struct {
	ID           uuid.UUID         `json:"id"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	Name         string            `json:"name"`
	Industry     string            `json:"industry,omitempty"`
	Size         string            `json:"size,omitempty"`
	Website      string            `json:"website,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Email        string            `json:"email,omitempty"`
	Address      string            `json:"address,omitempty"`
	City         string            `json:"city,omitempty"`
	Country      string            `json:"country,omitempty"`
	CustomFields map[string]any    `json:"custom_fields,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func (s *Server) ListCompanies(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)

	var total int64
	countQuery := `SELECT COUNT(*) FROM companies WHERE deleted_at IS NULL`
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}

	query := `SELECT id, tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields, created_at, updated_at
		FROM companies WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var companies []companyResp
	for rows.Next() {
		var c companyResp
		var cf []byte
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Industry, &c.Size, &c.Website, &c.Phone, &c.Email, &c.Address, &c.City, &c.Country, &cf, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		if cf != nil {
			json.Unmarshal(cf, &c.CustomFields)
		}
		companies = append(companies, c)
	}

	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}

	return s.json(c, http.StatusOK, listResp{Data: companies, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

func (s *Server) CreateCompany(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req companyReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Name == nil || *req.Name == "" {
		return s.errorResp(c, http.StatusBadRequest, "name is required", nil)
	}

	var cfBytes []byte
	if req.CustomFields != nil {
		cfBytes, _ = json.Marshal(req.CustomFields)
	}

	var resp companyResp
	err := s.pool.QueryRow(ctx, `
		INSERT INTO companies (tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields, created_at, updated_at
	`, tenantID, *req.Name, req.Industry, req.Size, req.Website, req.Phone, req.Email, req.Address, req.City, req.Country, cfBytes).
		Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.Industry, &resp.Size, &resp.Website, &resp.Phone, &resp.Email, &resp.Address, &resp.City, &resp.Country, &cfBytes, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	if cfBytes != nil {
		json.Unmarshal(cfBytes, &resp.CustomFields)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetCompany(c echo.Context) error {
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

	var resp companyResp
	var cfBytes []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields, created_at, updated_at
		FROM companies WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.Industry, &resp.Size, &resp.Website, &resp.Phone, &resp.Email, &resp.Address, &resp.City, &resp.Country, &cfBytes, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "company not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	if cfBytes != nil {
		json.Unmarshal(cfBytes, &resp.CustomFields)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateCompany(c echo.Context) error {
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

	var req companyReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var resp companyResp
	var cfBytes []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE companies SET 
			name = COALESCE($2, name),
			industry = COALESCE($3, industry),
			size = COALESCE($4, size),
			website = COALESCE($5, website),
			phone = COALESCE($6, phone),
			email = COALESCE($7, email),
			address = COALESCE($8, address),
			city = COALESCE($9, city),
			country = COALESCE($10, country),
			custom_fields = COALESCE($11, custom_fields),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields, created_at, updated_at
	`, id, req.Name, req.Industry, req.Size, req.Website, req.Phone, req.Email, req.Address, req.City, req.Country, req.CustomFields).
		Scan(&resp.ID, &resp.TenantID, &resp.Name, &resp.Industry, &resp.Size, &resp.Website, &resp.Phone, &resp.Email, &resp.Address, &resp.City, &resp.Country, &cfBytes, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "company not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if cfBytes != nil {
		json.Unmarshal(cfBytes, &resp.CustomFields)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteCompany(c echo.Context) error {
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

	result, err := s.pool.Exec(ctx, `UPDATE companies SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if result.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "company not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// Contacts Handlers
// =============================================================================

type contactReq struct {
	CompanyID    *uuid.UUID             `json:"company_id,omitempty"`
	FirstName    *string                `json:"first_name,omitempty"`
	LastName     *string                `json:"last_name,omitempty"`
	Email        *string                `json:"email,omitempty"`
	Phone        *string                `json:"phone,omitempty"`
	JobTitle     *string                `json:"job_title,omitempty"`
	Department   *string                `json:"department,omitempty"`
	OwnerUserID  *uuid.UUID             `json:"owner_user_id,omitempty"`
	Status       *string                `json:"status,omitempty"`
	Source       *string                `json:"source,omitempty"`
	CustomFields *map[string]any        `json:"custom_fields,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

type contactResp struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	CompanyID       *uuid.UUID     `json:"company_id,omitempty"`
	FirstName       string         `json:"first_name"`
	LastName        string         `json:"last_name"`
	Email           string         `json:"email,omitempty"`
	Phone           string         `json:"phone,omitempty"`
	JobTitle        string         `json:"job_title,omitempty"`
	Department      string         `json:"department,omitempty"`
	OwnerUserID     *uuid.UUID     `json:"owner_user_id,omitempty"`
	Status          string         `json:"status"`
	Source          string         `json:"source,omitempty"`
	CustomFields    map[string]any `json:"custom_fields,omitempty"`
	Tags            []string       `json:"tags"`
	LastContactedAt *time.Time     `json:"last_contacted_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (s *Server) ListContacts(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)

	var total int64
	countQuery := `SELECT COUNT(*) FROM contacts WHERE deleted_at IS NULL`
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}

	query := `SELECT id, tenant_id, company_id, first_name, last_name, email, phone, job_title, department, owner_user_id, status, source, custom_fields, tags, last_contacted_at, created_at, updated_at
		FROM contacts WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var contacts []contactResp
	for rows.Next() {
		var ct contactResp
		var cf []byte
		if err := rows.Scan(&ct.ID, &ct.TenantID, &ct.CompanyID, &ct.FirstName, &ct.LastName, &ct.Email, &ct.Phone, &ct.JobTitle, &ct.Department, &ct.OwnerUserID, &ct.Status, &ct.Source, &cf, &ct.Tags, &ct.LastContactedAt, &ct.CreatedAt, &ct.UpdatedAt); err != nil {
			continue
		}
		if cf != nil {
			json.Unmarshal(cf, &ct.CustomFields)
		}
		contacts = append(contacts, ct)
	}

	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}

	return s.json(c, http.StatusOK, listResp{Data: contacts, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

func (s *Server) CreateContact(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req contactReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.FirstName == nil || *req.FirstName == "" || req.LastName == nil || *req.LastName == "" {
		return s.errorResp(c, http.StatusBadRequest, "first_name and last_name are required", nil)
	}

	var cfBytes []byte
	if req.CustomFields != nil {
		cfBytes, _ = json.Marshal(req.CustomFields)
	}

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	var resp contactResp
	var cf []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO contacts (tenant_id, company_id, first_name, last_name, email, phone, job_title, department, owner_user_id, status, source, custom_fields, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, tenant_id, company_id, first_name, last_name, email, phone, job_title, department, owner_user_id, status, source, custom_fields, tags, last_contacted_at, created_at, updated_at
	`, tenantID, req.CompanyID, *req.FirstName, *req.LastName, req.Email, req.Phone, req.JobTitle, req.Department, req.OwnerUserID, status, req.Source, cfBytes, req.Tags).
		Scan(&resp.ID, &resp.TenantID, &resp.CompanyID, &resp.FirstName, &resp.LastName, &resp.Email, &resp.Phone, &resp.JobTitle, &resp.Department, &resp.OwnerUserID, &resp.Status, &resp.Source, &cf, &resp.Tags, &resp.LastContactedAt, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		slog.Error("create contact failed", slog.String("error", err.Error()))
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetContact(c echo.Context) error {
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

	var resp contactResp
	var cf []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, company_id, first_name, last_name, email, phone, job_title, department, owner_user_id, status, source, custom_fields, tags, last_contacted_at, created_at, updated_at
		FROM contacts WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.CompanyID, &resp.FirstName, &resp.LastName, &resp.Email, &resp.Phone, &resp.JobTitle, &resp.Department, &resp.OwnerUserID, &resp.Status, &resp.Source, &cf, &resp.Tags, &resp.LastContactedAt, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "contact not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateContact(c echo.Context) error {
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

	var req contactReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var resp contactResp
	var cf []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE contacts SET 
			company_id = COALESCE($2, company_id),
			first_name = COALESCE($3, first_name),
			last_name = COALESCE($4, last_name),
			email = COALESCE($5, email),
			phone = COALESCE($6, phone),
			job_title = COALESCE($7, job_title),
			department = COALESCE($8, department),
			owner_user_id = COALESCE($9, owner_user_id),
			status = COALESCE($10, status),
			source = COALESCE($11, source),
			custom_fields = COALESCE($12, custom_fields),
			tags = COALESCE($13, tags),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, company_id, first_name, last_name, email, phone, job_title, department, owner_user_id, status, source, custom_fields, tags, last_contacted_at, created_at, updated_at
	`, id, req.CompanyID, req.FirstName, req.LastName, req.Email, req.Phone, req.JobTitle, req.Department, req.OwnerUserID, req.Status, req.Source, req.CustomFields, req.Tags).
		Scan(&resp.ID, &resp.TenantID, &resp.CompanyID, &resp.FirstName, &resp.LastName, &resp.Email, &resp.Phone, &resp.JobTitle, &resp.Department, &resp.OwnerUserID, &resp.Status, &resp.Source, &cf, &resp.Tags, &resp.LastContactedAt, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "contact not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteContact(c echo.Context) error {
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

	result, err := s.pool.Exec(ctx, `UPDATE contacts SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if result.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "contact not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// Deals Handlers
// =============================================================================

type dealReq struct {
	ContactID         *uuid.UUID              `json:"contact_id,omitempty"`
	OwnerUserID       *uuid.UUID              `json:"owner_user_id,omitempty"`
	Name              *string                 `json:"name,omitempty"`
	Value             *float64                `json:"value,omitempty"`
	Currency          *string                 `json:"currency,omitempty"`
	Stage             *string                 `json:"stage,omitempty"`
	Probability       *int                     `json:"probability,omitempty"`
	ExpectedCloseDate *string                 `json:"expected_close_date,omitempty"`
	LostReason        *string                 `json:"lost_reason,omitempty"`
	CustomFields      *map[string]any         `json:"custom_fields,omitempty"`
}

type dealResp struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	ContactID         *uuid.UUID     `json:"contact_id,omitempty"`
	OwnerUserID       *uuid.UUID     `json:"owner_user_id,omitempty"`
	Name              string         `json:"name"`
	Value             float64        `json:"value"`
	Currency          string         `json:"currency"`
	Stage             string         `json:"stage"`
	Probability       *int           `json:"probability,omitempty"`
	ExpectedCloseDate *string        `json:"expected_close_date,omitempty"`
	ActualCloseDate   *time.Time     `json:"actual_close_date,omitempty"`
	LostReason        string         `json:"lost_reason,omitempty"`
	CustomFields      map[string]any `json:"custom_fields,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func (s *Server) ListDeals(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	p := getPagination(c)

	var total int64
	countQuery := `SELECT COUNT(*) FROM deals WHERE deleted_at IS NULL`
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}

	query := `SELECT id, tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date, actual_close_date, lost_reason, custom_fields, created_at, updated_at
		FROM deals WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, p.PerPage, p.Offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()

	var deals []dealResp
	for rows.Next() {
		var d dealResp
		var cf []byte
		if err := rows.Scan(&d.ID, &d.TenantID, &d.ContactID, &d.OwnerUserID, &d.Name, &d.Value, &d.Currency, &d.Stage, &d.Probability, &d.ExpectedCloseDate, &d.ActualCloseDate, &d.LostReason, &cf, &d.CreatedAt, &d.UpdatedAt); err != nil {
			continue
		}
		if cf != nil {
			json.Unmarshal(cf, &d.CustomFields)
		}
		deals = append(deals, d)
	}

	totalPages := int(total) / p.PerPage
	if int(total)%p.PerPage > 0 {
		totalPages++
	}

	return s.json(c, http.StatusOK, listResp{Data: deals, Total: total, Page: p.Page, PerPage: p.PerPage, TotalPages: totalPages})
}

func (s *Server) CreateDeal(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}

	var req dealReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	if req.Name == nil || *req.Name == "" {
		return s.errorResp(c, http.StatusBadRequest, "name is required", nil)
	}

	var cfBytes []byte
	if req.CustomFields != nil {
		cfBytes, _ = json.Marshal(req.CustomFields)
	}

	currency := "VND"
	if req.Currency != nil {
		currency = *req.Currency
	}
	stage := "prospecting"
	if req.Stage != nil {
		stage = *req.Stage
	}

	var resp dealResp
	var cf []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO deals (tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date, actual_close_date, lost_reason, custom_fields, created_at, updated_at
	`, tenantID, req.ContactID, req.OwnerUserID, *req.Name, req.Value, currency, stage, req.Probability, req.ExpectedCloseDate, cfBytes).
		Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.OwnerUserID, &resp.Name, &resp.Value, &resp.Currency, &resp.Stage, &resp.Probability, &resp.ExpectedCloseDate, &resp.ActualCloseDate, &resp.LostReason, &cf, &resp.CreatedAt, &resp.UpdatedAt)

	if err != nil {
		slog.Error("create deal failed", slog.String("error", err.Error()))
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}

	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetDeal(c echo.Context) error {
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

	var resp dealResp
	var cf []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date, actual_close_date, lost_reason, custom_fields, created_at, updated_at
		FROM deals WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.OwnerUserID, &resp.Name, &resp.Value, &resp.Currency, &resp.Stage, &resp.Probability, &resp.ExpectedCloseDate, &resp.ActualCloseDate, &resp.LostReason, &cf, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "deal not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateDeal(c echo.Context) error {
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

	var req dealReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	var resp dealResp
	var cf []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE deals SET 
			contact_id = COALESCE($2, contact_id),
			owner_user_id = COALESCE($3, owner_user_id),
			name = COALESCE($4, name),
			value = COALESCE($5, value),
			currency = COALESCE($6, currency),
			stage = COALESCE($7, stage),
			probability = COALESCE($8, probability),
			expected_close_date = COALESCE($9, expected_close_date),
			lost_reason = COALESCE($10, lost_reason),
			custom_fields = COALESCE($11, custom_fields),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date, actual_close_date, lost_reason, custom_fields, created_at, updated_at
	`, id, req.ContactID, req.OwnerUserID, req.Name, req.Value, req.Currency, req.Stage, req.Probability, req.ExpectedCloseDate, req.LostReason, req.CustomFields).
		Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.OwnerUserID, &resp.Name, &resp.Value, &resp.Currency, &resp.Stage, &resp.Probability, &resp.ExpectedCloseDate, &resp.ActualCloseDate, &resp.LostReason, &cf, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "deal not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}

	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteDeal(c echo.Context) error {
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

	result, err := s.pool.Exec(ctx, `UPDATE deals SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if result.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "deal not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

type moveDealStageReq struct {
	Stage      string `json:"stage"`
	Notes      string `json:"notes,omitempty"`
}

func (s *Server) MoveDealStage(c echo.Context) error {
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

	var req moveDealStageReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	validStages := map[string]bool{"prospecting": true, "qualification": true, "proposal": true, "negotiation": true, "won": true, "lost": true, "on_hold": true}
	if !validStages[req.Stage] {
		return s.errorResp(c, http.StatusBadRequest, "invalid stage", nil)
	}

	var oldStage string
	var resp dealResp
	var cf []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE deals SET 
			stage = $2,
			actual_close_date = CASE WHEN $2 IN ('won', 'lost') THEN NOW() ELSE actual_close_date END,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date, actual_close_date, lost_reason, custom_fields, created_at, updated_at
	`, id, req.Stage).Scan(&resp.ID, &resp.TenantID, &resp.ContactID, &resp.OwnerUserID, &resp.Name, &resp.Value, &resp.Currency, &resp.Stage, &resp.Probability, &resp.ExpectedCloseDate, &resp.ActualCloseDate, &resp.LostReason, &cf, &resp.CreatedAt, &resp.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "deal not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	_ = oldStage

	// Record stage history
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO deal_stage_history (deal_id, from_stage, to_stage, changed_by, notes)
		VALUES ($1, $2, $3, $4, $5)
	`, id, oldStage, req.Stage, userID, req.Notes)

	if cf != nil {
		json.Unmarshal(cf, &resp.CustomFields)
	}

	return s.json(c, http.StatusOK, resp)
}
