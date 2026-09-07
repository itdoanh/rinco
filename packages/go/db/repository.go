// Package db - generic CRUD repository pattern.
//
// Mục tiêu: giảm boilerplate cho các entity đơn giản (single-PK, soft-delete).
// Cho entity phức tạp (multi-column PK, special WHERE) → tự viết repository riêng.
//
// Usage:
//
//	type User struct {
//		ID        string
//		TenantID  string
//		Email     string
//		FullName  string
//		CreatedAt time.Time
//	}
//
//	users := db.NewRepository[User](pool, "auth", "users", "id")
//	user, _ := users.Get(ctx, "abc-123")
//	users.Create(ctx, &user)
package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("repository: not found")
	ErrDuplicateKey    = errors.New("repository: duplicate key")
	ErrInvalidEntity   = errors.New("repository: invalid entity")
)

// Repository generic cho entity với primary key là string (UUID).
// T cần có field tagged `db:"id"` (hoặc implement method ID() string).
type Repository[T any] struct {
	pool       *pgxpool.Pool
	schema     string
	table      string
	pkColumn   string
	columns    []string
}

// NewRepository khởi tạo repository cho entity T.
func NewRepository[T any](pool *pgxpool.Pool, schema, table, pkColumn string) *Repository[T] {
	if pkColumn == "" {
		pkColumn = "id"
	}
	var zero T
	cols := extractColumns(zero)
	return &Repository[T]{
		pool:     pool,
		schema:   schema,
		table:    table,
		pkColumn: pkColumn,
		columns:  cols,
	}
}

func (r *Repository[T]) fullTable() string {
	return fmt.Sprintf("%s.%s", r.schema, r.table)
}

// Get trả về entity theo primary key.
func (r *Repository[T]) Get(ctx context.Context, id string) (*T, error) {
	var zero T
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
		strings.Join(r.columns, ", "), r.fullTable(), r.pkColumn)
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("repo.Get: query: %w", err)
	}
	defer rows.Close()

	values, err := rows.Values()
	if err != nil {
		return nil, fmt.Errorf("repo.Get: scan: %w", err)
	}
	if values == nil {
		return nil, ErrNotFound
	}
	if err := fillFromValues(&zero, r.columns, values); err != nil {
		return nil, err
	}
	return &zero, nil
}

// Create inserts entity, trả về entity sau khi insert (timestamps nếu có).
func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	cols := r.columns
	placeholders := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, c := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		field := v.FieldByName(toFieldName(c))
		if !field.IsValid() {
			// Try lowercase
			field = v.FieldByName(strings.ToLower(c[:1]) + c[1:])
		}
		if !field.IsValid() {
			return fmt.Errorf("repo.Create: column %s not in struct", c)
		}
		args[i] = field.Interface()
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		r.fullTable(), strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("repo.Create: %w", err)
	}
	return nil
}

// Update cập nhật entity theo PK.
func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	id := v.FieldByName(toFieldName(r.pkColumn))
	if !id.IsValid() {
		return fmt.Errorf("repo.Update: PK %s not in struct", r.pkColumn)
	}

	sets := make([]string, 0, len(r.columns))
	args := make([]any, 0, len(r.columns)+1)
	for i, c := range r.columns {
		if c == r.pkColumn {
			continue
		}
		args = append(args, v.FieldByName(toFieldName(c)).Interface())
		sets = append(sets, fmt.Sprintf("%s = $%d", c, i+1))
	}
	args = append(args, id.Interface())
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		r.fullTable(), strings.Join(sets, ", "), r.pkColumn, len(args))
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("repo.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete xoá entity theo PK (hard delete).
func (r *Repository[T]) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", r.fullTable(), r.pkColumn), id)
	if err != nil {
		return fmt.Errorf("repo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List trả về entities theo filter đơn giản (eq).
func (r *Repository[T]) List(ctx context.Context, where map[string]any, limit int) ([]T, error) {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(r.columns, ", "), r.fullTable())
	args := make([]any, 0)
	if len(where) > 0 {
		conds := make([]string, 0, len(where))
		i := 1
		for k, v := range where {
			conds = append(conds, fmt.Sprintf("%s = $%d", k, i))
			args = append(args, v)
			i++
		}
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repo.List: %w", err)
	}
	defer rows.Close()

	out := make([]T, 0)
	for rows.Next() {
		var entity T
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("repo.List: scan: %w", err)
		}
		if err := fillFromValues(&entity, r.columns, values); err != nil {
			return nil, err
		}
		out = append(out, entity)
	}
	return out, nil
}

// ===== SoftDelete =====

// SoftDeleteMixin struct nhúng vào entity nếu muốn soft-delete.
type SoftDeleteMixin struct {
	DeletedAt *time.Time `db:"deleted_at"`
}

// IsDeleted kiểm tra soft-delete.
func (s SoftDeleteMixin) IsDeleted() bool { return s.DeletedAt != nil }

// extractColumns trích tên columns từ struct tags.
func extractColumns(v any) []string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	cols := make([]string, 0)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		cols = append(cols, tag)
	}
	return cols
}

func toFieldName(column string) string {
	parts := strings.Split(column, "_")
	for i, p := range parts {
		if i == 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		} else {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// fillFromValues đổ values vào struct theo tags.
func fillFromValues(dst any, columns []string, values []any) error {
	v := reflect.ValueOf(dst).Elem()
	if v.Kind() != reflect.Struct {
		return ErrInvalidEntity
	}
	for i, c := range columns {
		fieldName := toFieldName(c)
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}
		if !field.CanSet() {
			continue
		}
		val := reflect.ValueOf(values[i])
		if val.IsValid() && val.Type().ConvertibleTo(field.Type()) {
			field.Set(val.Convert(field.Type()))
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	// Fallback: pgx-specific
	if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
		return true
	}
	return false
}

// ListByTenant helper cho đa số entity có tenant_id.
func (r *Repository[T]) ListByTenant(ctx context.Context, tenantID string, limit int) ([]T, error) {
	return r.List(ctx, map[string]any{"tenant_id": tenantID}, limit)
}

// GetByComposite lookup theo composite key (vd: tenant_id + email).
func (r *Repository[T]) GetByComposite(ctx context.Context, where map[string]any) (*T, error) {
	results, err := r.List(ctx, where, 1)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return &results[0], nil
}

// Ensure interface compliance at compile time.
var _ fmt.Stringer = (interface{ String() string })(nil)