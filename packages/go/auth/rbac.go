// Package auth - Argon2id + RBAC engine.
//
// RBAC engine dùng pattern role → permissions:
//   role = { name, permissions }
//   user_roles = (user_id, role)
//   user_extra = (user_id, permission)  // optional per-user override
//
// Engine hỗ trợ:
//   - Check(role, permission): bool
//   - Grant(user, role): thêm role cho user
//   - Revoke(user, role): gỡ role
//   - HasPermission(user, perm): kiểm tra tổng hợp roles + extras
package auth

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

var (
	ErrRoleNotFound    = errors.New("rbac: role not found")
	ErrPermissionDeny  = errors.New("rbac: permission denied")
	ErrRoleExists      = errors.New("rbac: role already exists")
	ErrRoleInUse       = errors.New("rbac: role still in use")
)

// Role định nghĩa một tập permissions.
type Role struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	BuiltIn     bool     `json:"built_in,omitempty"` // built-in role không được xoá
}

// Permission mô tả một quyền (resource:action[:scope]).
// Format: "tenant.read", "lead.create", "crm.tree.move.subordinates".
type Permission string

// Resource tách permission thành resource + action.
func (p Permission) Parts() (resource, action string) {
	parts := strings.SplitN(string(p), ".", 2)
	if len(parts) != 2 {
		return string(p), ""
	}
	return parts[0], parts[1]
}

// Engine lưu trữ roles và quản lý assignment.
type Engine struct {
	mu    sync.RWMutex
	roles map[string]Role // key = role name
}

// NewEngine trả về engine với built-in roles.
func NewEngine() *Engine {
	e := &Engine{roles: make(map[string]Role)}
	for _, r := range BuiltInRoles() {
		e.roles[r.Name] = r
	}
	return e
}

// BuiltInRoles trả về các role mặc định của RINCO platform.
func BuiltInRoles() []Role {
	return []Role{
		{
			Name:        "super_admin",
			DisplayName: "Super Administrator",
			Description: "Toàn quyền trên toàn bộ hệ thống (cross-tenant).",
			BuiltIn:     true,
			Permissions: []string{"*"},
		},
		{
			Name:        "tenant_admin",
			DisplayName: "Tenant Admin",
			Description: "Toàn quyền trong 1 tenant.",
			BuiltIn:     true,
			Permissions: []string{
				"tenant.read", "tenant.update",
				"user.read", "user.create", "user.update", "user.delete",
				"role.read", "role.assign",
				"lead.*", "crm.*", "billing.read", "billing.update",
				"page.*", "form.*", "api_key.*",
			},
		},
		{
			Name:        "manager",
			DisplayName: "Manager",
			Description: "Quản lý team và lead pipeline.",
			BuiltIn:     true,
			Permissions: []string{
				"lead.read", "lead.create", "lead.update", "lead.assign", "lead.score",
				"contact.read", "contact.create", "contact.update",
				"company.read", "company.create", "company.update",
				"deal.read", "deal.create", "deal.update",
				"user.read",
				"crm.tree.read", "crm.tree.move",
			},
		},
		{
			Name:        "user",
			DisplayName: "User",
			Description: "User thường - xem data của mình.",
			BuiltIn:     true,
			Permissions: []string{
				"lead.read.own", "lead.update.own",
				"contact.read.own",
				"profile.read", "profile.update",
			},
		},
		{
			Name:        "guest",
			DisplayName: "Guest",
			Description: "Read-only public.",
			BuiltIn:     true,
			Permissions: []string{
				"page.read.published",
			},
		},
	}
}

// AddRole thêm role tùy chỉnh (không được ghi đè built-in).
func (e *Engine) AddRole(r Role) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if existing, ok := e.roles[r.Name]; ok && existing.BuiltIn {
		return ErrRoleExists
	}
	e.roles[r.Name] = r
	return nil
}

// RemoveRole xoá role (built-in không thể xoá).
func (e *Engine) RemoveRole(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	r, ok := e.roles[name]
	if !ok {
		return ErrRoleNotFound
	}
	if r.BuiltIn {
		return ErrRoleInUse
	}
	delete(e.roles, name)
	return nil
}

// GetRole trả về role theo tên.
func (e *Engine) GetRole(name string) (Role, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.roles[name]
	if !ok {
		return Role{}, ErrRoleNotFound
	}
	return r, nil
}

// ListRoles trả về tất cả roles (sorted by name).
func (e *Engine) ListRoles() []Role {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Role, 0, len(e.roles))
	for _, r := range e.roles {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// CheckPermission kiểm tra permission có nằm trong tập của role (hỗ trợ wildcard).
func (e *Engine) CheckPermission(roleName, perm string) bool {
	e.mu.RLock()
	r, ok := e.roles[roleName]
	e.mu.RUnlock()
	if !ok {
		return false
	}
	for _, p := range r.Permissions {
		if matchPermission(p, perm) {
			return true
		}
	}
	return false
}

// matchPermission hỗ trợ wildcard:
//   "*"   match all
//   "lead.*" match "lead.create", "lead.read", ...
//   exact match exact.
func matchPermission(pattern, perm string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == perm {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(perm, prefix+".")
	}
	return false
}

// PrincipalPermissions gộp permissions từ nhiều roles + extras.
func (e *Engine) PrincipalPermissions(roles []string, extras []string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, roleName := range roles {
		if r, ok := e.roles[roleName]; ok {
			for _, p := range r.Permissions {
				seen[p] = struct{}{}
			}
		}
	}
	for _, p := range extras {
		seen[p] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// CheckPrincipal quyết định xem principal (roles + extras) có permission không.
func (e *Engine) CheckPrincipal(roles []string, extras []string, perm string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, roleName := range roles {
		if r, ok := e.roles[roleName]; ok {
			for _, p := range r.Permissions {
				if matchPermission(p, perm) {
					return true
				}
			}
		}
	}
	for _, p := range extras {
		if matchPermission(p, perm) {
			return true
		}
	}
	return false
}

// Repository interface cho RBAC persistence layer.
type Repository interface {
	GetUserRoles(ctx context.Context, userID, tenantID string) ([]string, error)
	SetUserRoles(ctx context.Context, userID, tenantID string, roles []string) error
	AddUserRole(ctx context.Context, userID, tenantID, role string) error
	RemoveUserRole(ctx context.Context, userID, tenantID, role string) error
	GetUserExtras(ctx context.Context, userID, tenantID string) ([]string, error)
}

// UserStore là high-level facade sử dụng engine + repository.
type UserStore struct {
	engine *Engine
	repo   Repository
}

func NewUserStore(engine *Engine, repo Repository) *UserStore {
	return &UserStore{engine: engine, repo: repo}
}

// GrantRole thêm role cho user.
func (s *UserStore) GrantRole(ctx context.Context, userID, tenantID, role string) error {
	if _, err := s.engine.GetRole(role); err != nil {
		return err
	}
	return s.repo.AddUserRole(ctx, userID, tenantID, role)
}

// RevokeRole gỡ role khỏi user.
func (s *UserStore) RevokeRole(ctx context.Context, userID, tenantID, role string) error {
	return s.repo.RemoveUserRole(ctx, userID, tenantID, role)
}

// UserPermissions trả về toàn bộ permission áp dụng cho user.
func (s *UserStore) UserPermissions(ctx context.Context, userID, tenantID string) ([]string, error) {
	roles, err := s.repo.GetUserRoles(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	extras, err := s.repo.GetUserExtras(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	return s.engine.PrincipalPermissions(roles, extras), nil
}

// UserCan kiểm tra quyền cụ thể.
func (s *UserStore) UserCan(ctx context.Context, userID, tenantID, perm string) (bool, error) {
	roles, err := s.repo.GetUserRoles(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	extras, err := s.repo.GetUserExtras(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	return s.engine.CheckPrincipal(roles, extras, perm), nil
}