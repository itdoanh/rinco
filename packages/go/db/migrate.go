// Package db - migration runner.
//
// Hỗ trợ 2 mode:
//   1. Embed migrations từ folder thông qua go:embed → Migrator.Up()
//   2. Goose-compatible: read .sql files với -- +goose Up/Down annotations.
//
// Usage:
//
//	//go:embed migrations/*.sql
//	var migrations embed.FS
//
//	m := db.NewMigrator(pool, migrations)
//	if err := m.Up(ctx); err != nil { ... }
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrator quản lý migrations.
type Migrator struct {
	pool    *pgxpool.Pool
	fs      fs.FS
	table   string // schema_migrations
	verbose bool
}

// NewMigrator tạo migrator từ embed.FS.
func NewMigrator(pool *pgxpool.Pool, fs fs.FS) *Migrator {
	return &Migrator{pool: pool, fs: fs, table: "schema_migrations"}
}

// SetTable override table name (default "schema_migrations").
func (m *Migrator) SetTable(name string) *Migrator {
	m.table = name
	return m
}

// SetVerbose bật log chi tiết.
func (m *Migrator) SetVerbose(v bool) *Migrator {
	m.verbose = v
	return m
}

// ensureTable tạo bảng schema_migrations nếu chưa có.
func (m *Migrator) ensureTable(ctx context.Context) error {
	_, err := m.pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`, m.table))
	return err
}

// appliedVersions trả về versions đã apply.
func (m *Migrator) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pool.Query(ctx, fmt.Sprintf("SELECT version FROM %s", m.table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, nil
}

// Up apply tất cả migrations chưa chạy.
func (m *Migrator) Up(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return fmt.Errorf("migrator: ensure table: %w", err)
	}
	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("migrator: applied versions: %w", err)
	}

	entries, err := fs.ReadDir(m.fs, ".")
	if err != nil {
		return fmt.Errorf("migrator: read fs: %w", err)
	}

	type migFile struct {
		version string
		name    string
	}
	files := make([]migFile, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		// Skip "down" migrations
		if strings.Contains(strings.ToLower(e.Name()), ".down.") {
			continue
		}
		v := extractVersion(e.Name())
		if v == "" {
			continue
		}
		files = append(files, migFile{version: v, name: e.Name()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })

	for _, f := range files {
		if applied[f.version] {
			if m.verbose {
				fmt.Printf("[migrator] skip %s (already applied)\n", f.name)
			}
			continue
		}
		body, err := fs.ReadFile(m.fs, f.name)
		if err != nil {
			return fmt.Errorf("migrator: read %s: %w", f.name, err)
		}
		sql, direction := splitGoose(string(body))
		if direction == "Down" {
			continue
		}
		if m.verbose {
			fmt.Printf("[migrator] applying %s\n", f.name)
		}
		tx, err := m.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, sql); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrator: exec %s: %w", f.name, err)
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s (version) VALUES ($1)", m.table), f.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrator: insert %s: %w", f.name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrator: commit %s: %w", f.name, err)
		}
	}
	return nil
}

// extractVersion lấy version từ filename: "001_init.sql" → "001".
func extractVersion(name string) string {
	// Tìm phần prefix trước dấu "_" đầu tiên.
	idx := strings.Index(name, "_")
	if idx <= 0 {
		return ""
	}
	return name[:idx]
}

// splitGoose parse file theo Goose convention:
//
//	-- +goose Up
//	-- +goose StatementBegin
//	CREATE TABLE ...
//	-- +goose StatementEnd
//
//	-- +goose Down
//	DROP TABLE ...
//
// Trả về phần Up/Down SQL.
func splitGoose(body string) (sql, direction string) {
	direction = "Up"
	lines := strings.Split(body, "\n")
	inUp := false
	inDown := false
	inStmt := false
	var buf strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- +goose Up") {
			inUp = true
			inDown = false
			continue
		}
		if strings.HasPrefix(trimmed, "-- +goose Down") {
			direction = "Down"
			inUp = false
			inDown = true
			continue
		}
		if strings.HasPrefix(trimmed, "-- +goose StatementBegin") {
			inStmt = true
			continue
		}
		if strings.HasPrefix(trimmed, "-- +goose StatementEnd") {
			inStmt = false
			continue
		}
		// Skip annotation lines bắt đầu bằng --
		if !inStmt && strings.HasPrefix(trimmed, "--") {
			continue
		}
		if inUp || inDown {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	return buf.String(), direction
}

// MustEmbed là helper cho go:embed.
func MustEmbed(fsys embed.FS, pattern string) embed.FS {
	sub, err := fs.Sub(fsys, strings.TrimSuffix(pattern, "/*"))
	if err != nil {
		panic(err)
	}
	return sub
}