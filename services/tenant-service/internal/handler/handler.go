// Package handler provides HTTP handlers for the tenant-service.
//
// tenant-service responsibilities:
//   - Manage tenant lifecycle (create, suspend, restore, delete)
//   - Map domain ↔ tenant (subpath, subdomain, custom domain)
//   - Track quota / plan / billing state
//   - Enforce 2-of-3 quorum for destructive operations
package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// =============================================================================
// Types
// =============================================================================

// Tenant is the in-memory representation used by this scaffold.
// In production this maps to PostgreSQL row + Valkey cache.
type Tenant struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Plan        string    `json:"plan"`        // starter | pro | enterprise
	Status      string    `json:"status"`      // active | suspended | deleted
	CustomDomain string   `json:"custom_domain,omitempty"`
	OwnerUserID  uuid.UUID `json:"owner_user_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// QuorumRequest tracks a 2-of-3 multi-party approval for sensitive actions.
type QuorumRequest struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Action        string    `json:"action"`        // "delete" | "key_rotate"
	Signatures    []string  `json:"signatures"`    // admin user IDs that signed
	RequiredCount int       `json:"required_count"`
	ExpiresAt     time.Time `json:"expires_at"`
	Executed      bool      `json:"executed"`
}

// Server holds dependencies. In production: *pgxpool.Pool, *redis.Client, etc.
type Server struct {
	// mu protects the in-memory tenant store used by the scaffold/tests.
	mu       sync.RWMutex
	tenants  map[uuid.UUID]*Tenant
	slugs    map[string]uuid.UUID
	domains  map[string]uuid.UUID
	quorum   map[uuid.UUID]*QuorumRequest
}

// NewServer returns an empty Server.
func NewServer() *Server {
	return &Server{
		tenants: map[uuid.UUID]*Tenant{},
		slugs:   map[string]uuid.UUID{},
		domains: map[string]uuid.UUID{},
		quorum:  map[uuid.UUID]*QuorumRequest{},
	}
}

// =============================================================================
// Validation
// =============================================================================

var slugRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$`)
var domainRegex = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)

var reservedSlugs = map[string]bool{
	"admin": true, "api": true, "www": true, "system": true, "root": true,
	"internal": true, "support": true, "billing": true, "docs": true,
}

func validSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	if reservedSlugs[slug] {
		return fmt.Errorf("slug %q is reserved", slug)
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("slug must be lowercase, 2-40 chars, [a-z0-9-], no leading/trailing dash")
	}
	return nil
}

func validDomain(d string) error {
	if d == "" {
		return nil // optional
	}
	d = strings.ToLower(strings.TrimSpace(d))
	if !domainRegex.MatchString(d) {
		return fmt.Errorf("domain %q is not a valid FQDN", d)
	}
	return nil
}

func validPlan(p string) bool {
	return p == "starter" || p == "pro" || p == "enterprise"
}

// =============================================================================
// HTTP handlers
// =============================================================================

type createTenantReq struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Plan         string `json:"plan"`
	CustomDomain string `json:"custom_domain,omitempty"`
	OwnerEmail   string `json:"owner_email"`
}

type errorResp struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// CreateTenant handles POST /tenant/v1/tenants.
func (s *Server) CreateTenant(c echo.Context) error {
	var req createTenantReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid request", Details: err.Error()})
	}
	if err := validSlug(req.Slug); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid slug", Details: err.Error()})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "name required"})
	}
	if !validPlan(req.Plan) {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid plan (allowed: starter, pro, enterprise)"})
	}
	if err := validDomain(req.CustomDomain); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid custom_domain", Details: err.Error()})
	}
	if req.OwnerEmail == "" {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "owner_email required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.slugs[req.Slug]; exists {
		return c.JSON(http.StatusConflict, errorResp{Error: "slug already taken"})
	}
	if req.CustomDomain != "" {
		if _, exists := s.domains[req.CustomDomain]; exists {
			return c.JSON(http.StatusConflict, errorResp{Error: "custom_domain already in use"})
		}
	}
	now := time.Now().UTC()
	t := &Tenant{
		ID:           uuid.New(),
		Slug:         req.Slug,
		Name:         req.Name,
		Plan:         req.Plan,
		Status:       "active",
		CustomDomain: strings.ToLower(req.CustomDomain),
		OwnerUserID:  uuid.New(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.tenants[t.ID] = t
	s.slugs[t.Slug] = t.ID
	if t.CustomDomain != "" {
		s.domains[t.CustomDomain] = t.ID
	}
	return c.JSON(http.StatusCreated, t)
}

// GetTenant handles GET /tenant/v1/tenants/:id.
func (s *Server) GetTenant(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[id]
	if !ok {
		return c.JSON(http.StatusNotFound, errorResp{Error: "tenant not found"})
	}
	return c.JSON(http.StatusOK, t)
}

// ResolveDomain returns the tenant for a hostname.
func (s *Server) ResolveDomain(c echo.Context) error {
	host := strings.ToLower(c.QueryParam("host"))
	if host == "" {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "host query param required"})
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if id, ok := s.domains[host]; ok {
		return c.JSON(http.StatusOK, s.tenants[id])
	}
	return c.JSON(http.StatusNotFound, errorResp{Error: "no tenant for host"})
}

// SuspendTenant marks a tenant as suspended.
func (s *Server) SuspendTenant(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return c.JSON(http.StatusNotFound, errorResp{Error: "tenant not found"})
	}
	if t.Status == "deleted" {
		return c.JSON(http.StatusConflict, errorResp{Error: "cannot suspend deleted tenant"})
	}
	t.Status = "suspended"
	t.UpdatedAt = time.Now().UTC()
	return c.JSON(http.StatusOK, t)
}

// RequestDeletion starts a 2-of-3 quorum for deletion.
func (s *Server) RequestDeletion(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[id]; !ok {
		return c.JSON(http.StatusNotFound, errorResp{Error: "tenant not found"})
	}
	q := &QuorumRequest{
		ID:            uuid.New(),
		TenantID:      id,
		Action:        "delete",
		Signatures:    []string{},
		RequiredCount: 2,
		ExpiresAt:     time.Now().Add(5 * time.Minute).UTC(),
	}
	s.quorum[q.ID] = q
	return c.JSON(http.StatusCreated, q)
}

// SignDeletion appends an admin's signature to the quorum request.
// When the signature count reaches RequiredCount, the deletion is executed.
func (s *Server) SignDeletion(c echo.Context) error {
	qid, err := uuid.Parse(c.Param("qid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid quorum id"})
	}
	var body struct {
		AdminID string `json:"admin_id"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid request"})
	}
	if body.AdminID == "" {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "admin_id required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.quorum[qid]
	if !ok {
		return c.JSON(http.StatusNotFound, errorResp{Error: "quorum request not found"})
	}
	if time.Now().After(q.ExpiresAt) {
		return c.JSON(http.StatusGone, errorResp{Error: "quorum request expired"})
	}
	if q.Executed {
		return c.JSON(http.StatusConflict, errorResp{Error: "already executed"})
	}
	for _, sig := range q.Signatures {
		if sig == body.AdminID {
			return c.JSON(http.StatusConflict, errorResp{Error: "duplicate signature"})
		}
	}
	q.Signatures = append(q.Signatures, body.AdminID)
	if len(q.Signatures) >= q.RequiredCount {
		// Execute deletion.
		if t, ok := s.tenants[q.TenantID]; ok {
			t.Status = "deleted"
			t.UpdatedAt = time.Now().UTC()
			delete(s.slugs, t.Slug)
			if t.CustomDomain != "" {
				delete(s.domains, t.CustomDomain)
			}
			q.Executed = true
		}
	}
	return c.JSON(http.StatusOK, q)
}

// ListTenants returns all tenants (paginated).
func (s *Server) ListTenants(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		out = append(out, t)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"data":  out,
		"total": len(out),
	})
}
