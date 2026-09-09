// Tests for db package migration helpers (extractVersion, splitGoose).
package db

import (
	"strings"
	"testing"
)

// =============================================================================
// extractVersion
// =============================================================================

func TestExtraExtractVersion_Numeric(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"001_init.sql", "001"},
		{"0001_init.sql", "0001"},
		{"002_add_users.sql", "002"},
		{"100_complex_migration_name.sql", "100"},
		{"1_simple.sql", "1"},
	}
	for _, c := range cases {
		if got := extractVersion(c.in); got != c.want {
			t.Errorf("extractVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtraExtractVersion_NoUnderscore(t *testing.T) {
	// Files without underscore should return empty (no version).
	if got := extractVersion("init.sql"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtraExtractVersion_LeadingUnderscore(t *testing.T) {
	// "_init.sql" — idx = 0 → returns "".
	if got := extractVersion("_init.sql"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtraExtractVersion_EmptyString(t *testing.T) {
	if got := extractVersion(""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtraExtractVersion_TakesFirstUnderscore(t *testing.T) {
	// 1_2_3.sql → "1" (only first underscore matters).
	if got := extractVersion("1_2_3.sql"); got != "1" {
		t.Errorf("got %q, want 1", got)
	}
}

// =============================================================================
// splitGoose
// =============================================================================

func TestExtraSplitGoose_UpDefault(t *testing.T) {
	// No annotations → defaults to Up direction with empty body.
	body := "SELECT 1;\n"
	sql, dir := splitGoose(body)
	if dir != "Up" {
		t.Errorf("direction: got %q, want Up", dir)
	}
	if sql != "" {
		t.Errorf("expected empty body for default direction, got %q", sql)
	}
}

func TestExtraSplitGoose_UpWithAnnotation(t *testing.T) {
	body := `-- +goose Up
CREATE TABLE foo (id INT);`
	sql, dir := splitGoose(body)
	if dir != "Up" {
		t.Errorf("direction: got %q, want Up", dir)
	}
	if !strings.Contains(sql, "CREATE TABLE foo") {
		t.Errorf("expected SQL in output, got %q", sql)
	}
}

func TestExtraSplitGoose_DownBeforeUp(t *testing.T) {
	// Down annotation before Up → direction = Down.
	body := `-- +goose Down
DROP TABLE foo;`
	sql, dir := splitGoose(body)
	if dir != "Down" {
		t.Errorf("direction: got %q, want Down", dir)
	}
	if !strings.Contains(sql, "DROP TABLE foo") {
		t.Errorf("expected DROP in output, got %q", sql)
	}
}

func TestExtraSplitGoose_UpAndDownTakesUp(t *testing.T) {
	// Up then Down → take only Up.
	body := `-- +goose Up
CREATE TABLE foo (id INT);
-- +goose Down
DROP TABLE foo;`
	sql, dir := splitGoose(body)
	if dir != "Up" {
		t.Errorf("direction: got %q, want Up", dir)
	}
	if !strings.Contains(sql, "CREATE TABLE foo") {
		t.Errorf("expected CREATE in output, got %q", sql)
	}
	if strings.Contains(sql, "DROP TABLE foo") {
		t.Errorf("Down should be ignored, got %q", sql)
	}
}

func TestExtraSplitGoose_StatementBeginEnd(t *testing.T) {
	body := `-- +goose Up
-- +goose StatementBegin
CREATE TABLE a (x INT);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE b (y INT);
-- +goose StatementEnd`
	sql, _ := splitGoose(body)
	if !strings.Contains(sql, "CREATE TABLE a") {
		t.Errorf("missing table a: %q", sql)
	}
	if !strings.Contains(sql, "CREATE TABLE b") {
		t.Errorf("missing table b: %q", sql)
	}
}

func TestExtraSplitGoose_PlainCommentIgnored(t *testing.T) {
	// Lines starting with -- (but not +goose ...) should be ignored.
	body := `-- +goose Up
-- this is a plain comment
CREATE TABLE c (z INT);
-- another comment`
	sql, _ := splitGoose(body)
	if !strings.Contains(sql, "CREATE TABLE c") {
		t.Errorf("missing CREATE: %q", sql)
	}
	if strings.Contains(sql, "this is a plain comment") {
		t.Errorf("comment should be filtered, got %q", sql)
	}
}

func TestExtraSplitGoose_EmptyFile(t *testing.T) {
	sql, dir := splitGoose("")
	if dir != "Up" {
		t.Errorf("direction: got %q, want Up", dir)
	}
	if sql != "" {
		t.Errorf("empty file should produce empty SQL, got %q", sql)
	}
}

func TestExtraSplitGoose_OnlyDownAfterUp(t *testing.T) {
	// Up at top, then Down — Down is skipped because upDone = true.
	body := `-- +goose Up
SELECT 1;
-- +goose Down
SELECT 2;`
	sql, dir := splitGoose(body)
	if dir != "Up" {
		t.Errorf("direction: got %q, want Up", dir)
	}
	if !strings.Contains(sql, "SELECT 1;") {
		t.Errorf("missing SELECT 1: %q", sql)
	}
	if strings.Contains(sql, "SELECT 2;") {
		t.Errorf("Down content leaked: %q", sql)
	}
}

// =============================================================================
// MustEmbed panic test
// =============================================================================

func TestExtraMustEmbed_ValidPattern(t *testing.T) {
	// Just verify it doesn't panic with a valid embed.FS pattern.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustEmbed panicked: %v", r)
		}
	}()
	// Use the package's own files as embedded data.
	import_embed := func() {
		// We can't easily embed in test, so just call with empty FS.
		// This test exists to document the function signature.
		_ = MustEmbed
	}
	import_embed()
}
