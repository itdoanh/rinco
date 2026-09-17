// Package integration — tree_test.go exercises the LTREE-backed CRM tree
// endpoint. We assert that:
//
//   - nodes can be created with ltree paths
//   - moving a node re-parents its subtree
//   - attempting to move a node into its own descendant creates a cycle
//     and is rejected
//
//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crmTreeResp is a stub response shape used by the crm-service /crm/v1/tree.
type crmTreeResp struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	ParentID string         `json:"parent_id,omitempty"`
	Children []crmTreeResp  `json:"children,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
}

func createNode(t *testing.T, ctx *TestContext, crmURL, token, name, parentID, path string) crmTreeResp {
	t.Helper()
	body := map[string]any{
		"name":      name,
		"parent_id": parentID,
		"path":      path,
	}
	status, raw := ctx.DoJSON(http.MethodPost,
		crmURL+"/crm/v1/tree/nodes", body, token)
	require.True(t,
		status == http.StatusCreated || status == http.StatusOK,
		"create node %s → %d: %s", name, status, raw)
	var out crmTreeResp
	require.NoError(t, json.Unmarshal(raw, &out))
	require.NotEmpty(t, out.ID, "create returned empty id")
	if path == "" {
		out.Path = buildLtreePath(out.ID, parentID)
	}
	return out
}

// buildLtreePath constructs a placeholder ltree string from the parent
// path + new node id (UUIDs use dashes which ltree accepts).
func buildLtreePath(id, parentID string) string {
	if parentID == "" {
		return "root." + strings.ReplaceAll(id, "-", "_")
	}
	return "root." + strings.ReplaceAll(parentID, "-", "_") + "." +
		strings.ReplaceAll(id, "-", "_")
}

// TestCRMTreeHierarchy walks the canonical happy path: create root,
// create child, create grandchild, then verify GET /crm/v1/tree/{id}
// returns the nested structure.
func TestCRMTreeHierarchy(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	if env.Users == nil {
		t.Skip("no demo users")
	}
	ctx := NewTestContext(t, env)
	tp := ctx.Login(env.Users.ApexAdmin.Email, env.Users.ApexAdmin.Password, env.Users.ApexAdmin.TenantID)

	root := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken,
		"Region North", "", "")
	child := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken,
		"HCM Branch", root.ID, buildLtreePath(uuid.NewString(), root.ID))
	grand := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken,
		"District 1", child.ID, buildLtreePath(uuid.NewString(), child.ID))

	require.NotEmpty(t, root.ID)
	require.NotEmpty(t, child.ID)
	require.NotEmpty(t, grand.ID)
	assert.Contains(t, grand.Path, child.ID[:8])
}

// TestCRMTreeMoveSubtree verifies that moving a node under a new parent
// re-parents its subtree (paths are updated).
func TestCRMTreeMoveSubtree(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	if env.Users == nil {
		t.Skip("no demo users")
	}
	ctx := NewTestContext(t, env)
	tp := ctx.Login(env.Users.ApexAdmin.Email, env.Users.ApexAdmin.Password, env.Users.ApexAdmin.TenantID)

	root := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken, "R1", "", "")
	a := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken, "A", root.ID,
		buildLtreePath(uuid.NewString(), root.ID))
	b := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken, "B", root.ID,
		buildLtreePath(uuid.NewString(), root.ID))
	leaf := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken, "leaf", a.ID,
		buildLtreePath(uuid.NewString(), a.ID))

	// Move A under B
	status, body := ctx.DoJSON(http.MethodPost,
		fmt.Sprintf("%s/crm/v1/tree/nodes/%s/move", env.Service.CRMURL, a.ID),
		map[string]any{"new_parent_id": b.ID, "new_index": 0},
		tp.AccessToken)
	require.True(t,
		status == http.StatusOK || status == http.StatusNoContent,
		"move A → B: %d %s", status, body)

	// Confirm leaf now belongs to B (transitively)
	_, body = ctx.DoJSON(http.MethodGet,
		fmt.Sprintf("%s/crm/v1/tree/nodes/%s", env.Service.CRMURL, leaf.ID),
		nil, tp.AccessToken)
	if status == 200 || status == 201 {
		var got crmTreeResp
		require.NoError(t, json.Unmarshal(body, &got))
		assert.True(t,
			strings.Contains(got.Path, strings.ReplaceAll(b.ID, "-", "_")) ||
				got.ParentID == b.ID,
			"leaf should now be under B; got path=%s parent=%s", got.Path, got.ParentID,
		)
	}
}

// TestCRMTreeCycleRejected exercises the no-self-parent / no-descendant
// invariant. Moving a node into one of its own descendants would create
// a cycle and must be rejected (typically with 400/422).
func TestCRMTreeCycleRejected(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	if env.Users == nil {
		t.Skip("no demo users")
	}
	ctx := NewTestContext(t, env)
	tp := ctx.Login(env.Users.ApexAdmin.Email, env.Users.ApexAdmin.Password, env.Users.ApexAdmin.TenantID)

	root := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken, "CycleRoot", "", "")
	child := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken, "CycleChild", root.ID,
		buildLtreePath(uuid.NewString(), root.ID))

	// Attempt to move root INTO child → cycle.
	status, body := ctx.DoJSON(http.MethodPost,
		fmt.Sprintf("%s/crm/v1/tree/nodes/%s/move", env.Service.CRMURL, root.ID),
		map[string]any{"new_parent_id": child.ID},
		tp.AccessToken)
	require.True(t,
		status == http.StatusBadRequest || status == http.StatusUnprocessableEntity || status == http.StatusConflict,
		"cycle move must be rejected, got %d: %s", status, body)
}

// TestCRMTreeNoSelfParent ensures a node cannot be its own parent.
func TestCRMTreeNoSelfParent(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	if env.Users == nil {
		t.Skip("no demo users")
	}
	ctx := NewTestContext(t, env)
	tp := ctx.Login(env.Users.ApexAdmin.Email, env.Users.ApexAdmin.Password, env.Users.ApexAdmin.TenantID)

	root := createNode(t, ctx, env.Service.CRMURL, tp.AccessToken, "SelfParent", "", "")
	status, _ := ctx.DoJSON(http.MethodPost,
		fmt.Sprintf("%s/crm/v1/tree/nodes/%s/move", env.Service.CRMURL, root.ID),
		map[string]any{"new_parent_id": root.ID},
		tp.AccessToken)
	require.True(t, status >= 400, "self-parent must be rejected, got %d", status)
}
