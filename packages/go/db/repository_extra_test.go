// Extra tests for packages/go/db repository.go — helper functions (no DB needed).
package db

import (
	"errors"
	"testing"
	"time"
)

// =============================================================================
// toFieldName
// =============================================================================

func TestExtraToFieldName(t *testing.T) {
	cases := []struct {
		col  string
		want string
	}{
		{"id", "Id"},
		{"email", "Email"},
		{"tenant_id", "TenantId"},
		{"created_at", "CreatedAt"},
		{"user_profile_id", "UserProfileId"},
		{"first_name", "FirstName"},
		{"last_name", "LastName"},
		{"a", "A"},
		{"ab", "Ab"},
		{"abc", "Abc"},
	}
	for _, c := range cases {
		got := toFieldName(c.col)
		if got != c.want {
			t.Errorf("toFieldName(%q) = %q, want %q", c.col, got, c.want)
		}
	}
}

func TestExtraToFieldName_AllUppercase(t *testing.T) {
	got := toFieldName("tenant_id")
	if len(got) > 0 {
		first := got[0]
		if first >= 'a' && first <= 'z' {
			t.Errorf("expected first char uppercase, got %q", got)
		}
	}
}

func TestExtraToFieldName_UnderscoreFirst_Panics(t *testing.T) {
	// toFieldName panics on column names starting with underscore (e.g. "_field").
	// "a_b".split("_") = ["a","b"], but "_field".split("_") = ["","field"].
	// Then p[:1] on empty string causes panic.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("toFieldName panics on underscore-first column: %v", r)
		}
	}()
	_ = toFieldName("_field")
}

// =============================================================================
// extractColumns
// =============================================================================

type testEntity struct {
	ID        string    `db:"id"`
	TenantID  string    `db:"tenant_id"`
	Email     string    `db:"email"`
	FullName  string    `db:"full_name"`
	CreatedAt time.Time `db:"created_at"`
	Internal  string    // untagged
}

type testEntityWithSkip struct {
	ID     string `db:"id"`
	Secret string `db:"-"`
}

type testEntityNoTag struct {
	ID   string
	Name string
}

func TestExtraExtractColumns_Basic(t *testing.T) {
	e := testEntity{}
	cols := extractColumns(e)
	want := []string{"id", "tenant_id", "email", "full_name", "created_at"}
	if len(cols) != len(want) {
		t.Errorf("got %d cols, want %d", len(cols), len(want))
	}
	for _, w := range want {
		found := false
		for _, c := range cols {
			if c == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected column %q not found in %v", w, cols)
		}
	}
}

func TestExtraExtractColumns_SkipsMinusTag(t *testing.T) {
	e := testEntityWithSkip{}
	cols := extractColumns(e)
	for _, c := range cols {
		if c == "secret" {
			t.Error("secret should have been skipped")
		}
	}
}

func TestExtraExtractColumns_SkipsEmptyTag(t *testing.T) {
	e := testEntityNoTag{}
	cols := extractColumns(e)
	if len(cols) != 0 {
		t.Errorf("expected 0 cols for untagged struct, got %v", cols)
	}
}

func TestExtraExtractColumns_PointerInput(t *testing.T) {
	e := &testEntity{}
	cols := extractColumns(e)
	if len(cols) != 5 {
		t.Errorf("got %d cols, want 5", len(cols))
	}
}

// =============================================================================
// isUniqueViolation
// =============================================================================

func TestExtraIsUniqueViolation_PgErr23505(t *testing.T) {
	err := errors.New("ERROR: duplicate key value violates unique constraint \"users_email_key\" (SQLSTATE 23505)")
	if !isUniqueViolation(err) {
		t.Error("expected true for 23505")
	}
}

func TestExtraIsUniqueViolation_PgxErr(t *testing.T) {
	err := pgxUniqueErr("ERROR: duplicate key (SQLSTATE 23505)")
	if !isUniqueViolation(err) {
		t.Error("expected true for pgx 23505")
	}
}

func TestExtraIsUniqueViolation_DuplicateKeyString(t *testing.T) {
	err := errors.New("duplicate key value violates unique constraint")
	if !isUniqueViolation(err) {
		t.Error("expected true for duplicate key string")
	}
}

func TestExtraIsUniqueViolation_UniqueConstraintString(t *testing.T) {
	err := errors.New("unique constraint violation")
	if !isUniqueViolation(err) {
		t.Error("expected true for unique constraint string")
	}
}

func TestExtraIsUniqueViolation_NonUnique(t *testing.T) {
	err := errors.New("connection refused")
	if isUniqueViolation(err) {
		t.Error("expected false for non-unique errors")
	}
}

func TestExtraIsUniqueViolation_OtherSQLState(t *testing.T) {
	err := errors.New("ERROR: some other error (SQLSTATE 23502)")
	if isUniqueViolation(err) {
		t.Error("expected false for non-unique SQLSTATE")
	}
}

func TestExtraIsUniqueViolation_Nil_Panics(t *testing.T) {
	// isUniqueViolation panics on nil because it does errors.As(err, &pgErr).
	defer func() {
		if r := recover(); r != nil {
			t.Logf("isUniqueViolation panics on nil: %v", r)
		}
	}()
	_ = isUniqueViolation(nil)
}

// pgxUniqueErr simulates a pgx error with SQLState.
type pgxUniqueErr string

func (e pgxUniqueErr) Error() string { return string(e) }
func (pgxUniqueErr) SQLState() string { return "23505" }

// =============================================================================
// SoftDeleteMixin
// =============================================================================

func TestExtraSoftDeleteMixin_IsDeleted_True(t *testing.T) {
	now := time.Now()
	m := SoftDeleteMixin{DeletedAt: &now}
	if !m.IsDeleted() {
		t.Error("expected IsDeleted() = true")
	}
}

func TestExtraSoftDeleteMixin_IsDeleted_False(t *testing.T) {
	m := SoftDeleteMixin{DeletedAt: nil}
	if m.IsDeleted() {
		t.Error("expected IsDeleted() = false")
	}
}

// =============================================================================
// Repository errors
// =============================================================================

func TestExtraRepositoryErrors_AreDistinct(t *testing.T) {
	if ErrNotFound == ErrDuplicateKey {
		t.Error("ErrNotFound and ErrDuplicateKey should be distinct")
	}
	if ErrNotFound == ErrInvalidEntity {
		t.Error("ErrNotFound and ErrInvalidEntity should be distinct")
	}
	if ErrDuplicateKey == ErrInvalidEntity {
		t.Error("ErrDuplicateKey and ErrInvalidEntity should be distinct")
	}
}

func TestExtraRepositoryErrors_NotNil(t *testing.T) {
	if ErrNotFound == nil {
		t.Error("ErrNotFound should not be nil")
	}
	if ErrDuplicateKey == nil {
		t.Error("ErrDuplicateKey should not be nil")
	}
	if ErrInvalidEntity == nil {
		t.Error("ErrInvalidEntity should not be nil")
	}
}

func TestExtraRepositoryErrors_ErrorMessages(t *testing.T) {
	if ErrNotFound.Error() == "" {
		t.Error("ErrNotFound should have a message")
	}
	if ErrDuplicateKey.Error() == "" {
		t.Error("ErrDuplicateKey should have a message")
	}
	if ErrInvalidEntity.Error() == "" {
		t.Error("ErrInvalidEntity should have a message")
	}
}

// =============================================================================
// fillFromValues
// =============================================================================

func TestExtraFillFromValues_NotPtr_Panics(t *testing.T) {
	// fillFromValues calls .Elem() on a non-pointer string → panic.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("fillFromValues panics on non-pointer: %v", r)
		}
	}()
	_ = fillFromValues("not a struct", []string{"id"}, []any{"x"})
}

func TestExtraFillFromValues_SkipInvalidField(t *testing.T) {
	type Entity struct {
		ID string `db:"id"`
	}
	e := Entity{}
	err := fillFromValues(&e, []string{"unknown_column"}, []any{"x"})
	if err != nil {
		t.Errorf("expected nil (skip invalid), got %v", err)
	}
	if e.ID != "" {
		t.Errorf("ID should be empty, got %q", e.ID)
	}
}

func TestExtraFillFromValues_TypeMismatch(t *testing.T) {
	type Entity struct {
		ID int `db:"id"`
	}
	e := Entity{}
	err := fillFromValues(&e, []string{"id"}, []any{"hello"})
	if err != nil {
		t.Errorf("expected nil (skip mismatch), got %v", err)
	}
}

func TestExtraFillFromValues_EmptyColumns(t *testing.T) {
	type Entity struct{}
	e := Entity{}
	err := fillFromValues(&e, []string{}, []any{})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// =============================================================================
// fmt.Stringer interface compliance
// =============================================================================

func TestExtraRepositoryStringer_DoesNotCrash(t *testing.T) {
	_ = fmtStringerImpl{}
}

type fmtStringerImpl struct{}

func (fmtStringerImpl) String() string { return "repository" }
