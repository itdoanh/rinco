// Tests for auth RBAC engine (rbac.go).
package auth

import (
	"testing"
)

func TestExtra_BuiltInRoles(t *testing.T) {
	roles := BuiltInRoles()
	if len(roles) == 0 {
		t.Fatal("BuiltInRoles should not be empty")
	}
	
	// Check super_admin exists
	found := false
	for _, r := range roles {
		if r.Name == "super_admin" {
			found = true
			if r.BuiltIn != true {
				t.Error("super_admin should be built-in")
			}
			break
		}
	}
	if !found {
		t.Error("super_admin not found")
	}
}

func TestExtra_BuiltInRoles_AllHaveNames(t *testing.T) {
	roles := BuiltInRoles()
	for _, r := range roles {
		if r.Name == "" {
			t.Error("role with empty name")
		}
	}
}

func TestExtra_NewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	if e.roles == nil {
		t.Error("roles map should be initialized")
	}
	// Should have built-in roles
	if len(e.roles) == 0 {
		t.Error("should have built-in roles")
	}
}

func TestExtra_Engine_AddRole(t *testing.T) {
	e := NewEngine()
	initialCount := len(e.roles)
	
	role := Role{
		Name:        "custom_role",
		DisplayName: "Custom Role",
		Permissions: []string{"custom.read", "custom.write"},
	}
	err := e.AddRole(role)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	if len(e.roles) != initialCount+1 {
		t.Error("role not added")
	}
}

func TestExtra_Engine_AddRole_BuiltinNotAllowed(t *testing.T) {
	e := NewEngine()
	
	role := Role{
		Name:        "super_admin", // Built-in
		DisplayName: "Try Override",
	}
	err := e.AddRole(role)
	if err != ErrRoleExists {
		t.Errorf("expected ErrRoleExists, got: %v", err)
	}
}

func TestExtra_Engine_AddRole_Duplicate(t *testing.T) {
	e := NewEngine()
	
	role := Role{Name: "custom1", Permissions: []string{"read"}}
	_ = e.AddRole(role)
	
	role2 := Role{Name: "custom1", Permissions: []string{"write"}}
	err := e.AddRole(role2)
	// Should update existing (not ErrRoleExists)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtra_Engine_RemoveRole(t *testing.T) {
	e := NewEngine()
	role := Role{Name: "removable", Permissions: []string{"read"}}
	_ = e.AddRole(role)
	
	err := e.RemoveRole("removable")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtra_Engine_RemoveRole_BuiltinNotAllowed(t *testing.T) {
	e := NewEngine()
	err := e.RemoveRole("super_admin")
	if err != ErrRoleInUse {
		t.Errorf("expected ErrRoleInUse, got: %v", err)
	}
}

func TestExtra_Engine_RemoveRole_NotFound(t *testing.T) {
	e := NewEngine()
	err := e.RemoveRole("nonexistent")
	if err != ErrRoleNotFound {
		t.Errorf("expected ErrRoleNotFound, got: %v", err)
	}
}

func TestExtra_Engine_GetRole(t *testing.T) {
	e := NewEngine()
	role, err := e.GetRole("super_admin")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if role.Name != "super_admin" {
		t.Errorf("got %s", role.Name)
	}
}

func TestExtra_Engine_GetRole_NotFound(t *testing.T) {
	e := NewEngine()
	_, err := e.GetRole("nonexistent")
	if err != ErrRoleNotFound {
		t.Errorf("expected ErrRoleNotFound, got: %v", err)
	}
}

func TestExtra_Engine_ListRoles(t *testing.T) {
	e := NewEngine()
	roles := e.ListRoles()
	if len(roles) == 0 {
		t.Error("ListRoles should not be empty")
	}
	// Should be sorted by name
	for i := 1; i < len(roles); i++ {
		if roles[i-1].Name > roles[i].Name {
			t.Error("ListRoles should be sorted by name")
		}
	}
}

func TestExtra_Engine_CheckPermission(t *testing.T) {
	e := NewEngine()
	
	// super_admin has "*"
	if !e.CheckPermission("super_admin", "anything.here") {
		t.Error("super_admin should match anything")
	}
	
	// guest only has page.read.published
	if e.CheckPermission("guest", "lead.read") {
		t.Error("guest should not have lead.read")
	}
	if !e.CheckPermission("guest", "page.read.published") {
		t.Error("guest should have page.read.published")
	}
}

func TestExtra_Engine_CheckPermission_RoleNotFound(t *testing.T) {
	e := NewEngine()
	if e.CheckPermission("nonexistent", "read") {
		t.Error("nonexistent role should return false")
	}
}

func TestExtra_Engine_CheckPermission_Wildcard(t *testing.T) {
	e := NewEngine()
	
	// tenant_admin has "lead.*"
	if !e.CheckPermission("tenant_admin", "lead.read") {
		t.Error("tenant_admin should have lead.read")
	}
	if !e.CheckPermission("tenant_admin", "lead.create") {
		t.Error("tenant_admin should have lead.create")
	}
	if !e.CheckPermission("tenant_admin", "lead.delete") {
		t.Error("tenant_admin should have lead.delete")
	}
}

func TestExtra_matchPermission(t *testing.T) {
	tests := []struct {
		pattern string
		perm    string
		want    bool
	}{
		{"*", "anything", true},
		{"lead.read", "lead.read", true},
		{"lead.read", "lead.write", false},
		{"lead.*", "lead.read", true},
		{"lead.*", "lead.create", true},
		{"lead.*", "other.read", false},
		// Note: "page.*.published" only matches exact "page.*.published", not "page.read.published"
		{"page.*.published", "page.*.published", true},
		{"page.*.published", "page.read.published", false}, // Not a wildcard pattern
		{"page.read.published", "page.read.published", true},
	}
	
	for _, tt := range tests {
		got := matchPermission(tt.pattern, tt.perm)
		if got != tt.want {
			t.Errorf("matchPermission(%q, %q) = %v, want %v", tt.pattern, tt.perm, got, tt.want)
		}
	}
}

func TestExtra_Engine_PrincipalPermissions(t *testing.T) {
	e := NewEngine()
	
	perms := e.PrincipalPermissions([]string{"manager"}, nil)
	if len(perms) == 0 {
		t.Error("manager should have permissions")
	}
	
	// Check some specific permissions
	found := false
	for _, p := range perms {
		if p == "lead.read" {
			found = true
			break
		}
	}
	if !found {
		t.Error("manager should have lead.read")
	}
}

func TestExtra_Engine_PrincipalPermissions_Extras(t *testing.T) {
	e := NewEngine()
	
	perms := e.PrincipalPermissions([]string{"user"}, []string{"custom.perm"})
	found := false
	for _, p := range perms {
		if p == "custom.perm" {
			found = true
			break
		}
	}
	if !found {
		t.Error("extras should be included")
	}
}

func TestExtra_Engine_PrincipalPermissions_Sorted(t *testing.T) {
	e := NewEngine()
	
	perms := e.PrincipalPermissions([]string{"manager"}, nil)
	for i := 1; i < len(perms); i++ {
		if perms[i-1] > perms[i] {
			t.Error("permissions should be sorted")
		}
	}
}

func TestExtra_Engine_PrincipalPermissions_Deduplication(t *testing.T) {
	e := NewEngine()
	
	// Same role twice should not duplicate permissions
	perms := e.PrincipalPermissions([]string{"manager", "manager"}, nil)
	seen := make(map[string]int)
	for _, p := range perms {
		seen[p]++
	}
	for p, count := range seen {
		if count > 1 {
			t.Errorf("permission %q duplicated %d times", p, count)
		}
	}
}

func TestExtra_Engine_PrincipalPermissions_UnknownRole(t *testing.T) {
	e := NewEngine()
	
	// Unknown role should be ignored
	perms := e.PrincipalPermissions([]string{"unknown_role"}, nil)
	if len(perms) > 0 {
		t.Error("unknown role should contribute no permissions")
	}
}

func TestExtra_Engine_CheckPermission_Manager(t *testing.T) {
	e := NewEngine()
	
	// Manager should have lead permissions
	if !e.CheckPermission("manager", "lead.read") {
		t.Error("manager should have lead.read")
	}
	if !e.CheckPermission("manager", "lead.update") {
		t.Error("manager should have lead.update")
	}
	
	// Manager should NOT have user management
	if e.CheckPermission("manager", "user.delete") {
		t.Error("manager should not have user.delete")
	}
}

func TestExtra_Engine_CheckPermission_TenantAdmin(t *testing.T) {
	e := NewEngine()
	
	// Tenant admin should have billing
	if !e.CheckPermission("tenant_admin", "billing.read") {
		t.Error("tenant_admin should have billing.read")
	}
	
	// Tenant admin should NOT have cross-tenant access
	if e.CheckPermission("tenant_admin", "*") {
		t.Error("tenant_admin should not have wildcard")
	}
}

func TestExtra_Role_Fields(t *testing.T) {
	r := Role{
		Name:        "test_role",
		DisplayName: "Test Role",
		Description: "A test role",
		Permissions: []string{"read", "write"},
		BuiltIn:    false,
	}
	if r.Name != "test_role" {
		t.Error("Name")
	}
	if r.DisplayName != "Test Role" {
		t.Error("DisplayName")
	}
	if r.Description != "A test role" {
		t.Error("Description")
	}
	if len(r.Permissions) != 2 {
		t.Error("Permissions")
	}
	if r.BuiltIn {
		t.Error("BuiltIn")
	}
}

func TestExtra_Role_BuiltIn(t *testing.T) {
	roles := BuiltInRoles()
	for _, r := range roles {
		if !r.BuiltIn {
			t.Errorf("role %s should be built-in", r.Name)
		}
	}
}

func TestExtra_Permission_Parts(t *testing.T) {
	tests := []struct {
		p        Permission
		resource string
		action   string
	}{
		{"lead.read", "lead", "read"},
		{"page.view.published", "page", "view.published"},
		{"simple", "simple", ""},
		{"a.b.c", "a", "b.c"},
	}
	
	for _, tt := range tests {
		res, act := tt.p.Parts()
		if res != tt.resource {
			t.Errorf("Permission(%q).Parts() resource = %q, want %q", tt.p, res, tt.resource)
		}
		if act != tt.action {
			t.Errorf("Permission(%q).Parts() action = %q, want %q", tt.p, act, tt.action)
		}
	}
}

func TestExtra_Permission_Parts_NoDot(t *testing.T) {
	p := Permission("nodots")
	res, act := p.Parts()
	if res != "nodots" {
		t.Errorf("got %s", res)
	}
	if act != "" {
		t.Errorf("action should be empty, got %s", act)
	}
}

func TestExtra_Engine_ConcurrentAddRemove(t *testing.T) {
	e := NewEngine()
	
	// Add many roles concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			role := Role{Name: "concurrent_role", Permissions: []string{"read"}}
			_ = e.AddRole(role)
			done <- true
		}(i)
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestExtra_Engine_ListRoles_Copy(t *testing.T) {
	e := NewEngine()
	
	roles1 := e.ListRoles()
	roles2 := e.ListRoles()
	
	if &roles1[0] == &roles2[0] {
		t.Error("ListRoles should return a copy, not the same slice")
	}
}

func TestExtra_RBAC_Errors(t *testing.T) {
	if ErrRoleNotFound.Error() == "" {
		t.Error("ErrRoleNotFound should have a message")
	}
	if ErrPermissionDeny.Error() == "" {
		t.Error("ErrPermissionDeny should have a message")
	}
	if ErrRoleExists.Error() == "" {
		t.Error("ErrRoleExists should have a message")
	}
	if ErrRoleInUse.Error() == "" {
		t.Error("ErrRoleInUse should have a message")
	}
}
