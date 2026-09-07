package platform

import (
	"os"
	"testing"
)

func TestEnv(t *testing.T) {
	// Test default value
	os.Unsetenv("ENV")
	if got := Env(); got != "development" {
		t.Errorf("Env() = %v, want development", got)
	}

	// Test custom value
	os.Setenv("ENV", "production")
	if got := Env(); got != "production" {
		t.Errorf("Env() = %v, want production", got)
	}
	os.Unsetenv("ENV")
}

func TestGetenv(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		def   string
		setup func()
		want  string
	}{
		{
			name:  "existing key",
			key:   "TEST_VAR_EXISTS",
			def:   "default",
			setup: func() { os.Setenv("TEST_VAR_EXISTS", "value") },
			want:  "value",
		},
		{
			name:  "missing key returns default",
			key:   "TEST_VAR_MISSING",
			def:   "default",
			setup: func() { os.Unsetenv("TEST_VAR_MISSING") },
			want:  "default",
		},
		{
			name:  "empty string returns default",
			key:   "TEST_VAR_EMPTY",
			def:   "default",
			setup: func() { os.Setenv("TEST_VAR_EMPTY", "") },
			want:  "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer os.Unsetenv(tt.key)

			got := Getenv(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("Getenv(%q, %q) = %q, want %q", tt.key, tt.def, got, tt.want)
			}
		})
	}
}

func TestGetenvInt(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		def   int
		setup func()
		want  int
	}{
		{
			name:  "valid integer",
			key:   "TEST_INT_VALID",
			def:   0,
			setup: func() { os.Setenv("TEST_INT_VALID", "42") },
			want:  42,
		},
		{
			name:  "missing returns default",
			key:   "TEST_INT_MISSING",
			def:   100,
			setup: func() { os.Unsetenv("TEST_INT_MISSING") },
			want:  100,
		},
		{
			name:  "invalid integer returns default",
			key:   "TEST_INT_INVALID",
			def:   200,
			setup: func() { os.Setenv("TEST_INT_INVALID", "notanumber") },
			want:  200,
		},
		{
			name:  "negative integer",
			key:   "TEST_INT_NEG",
			def:   0,
			setup: func() { os.Setenv("TEST_INT_NEG", "-5") },
			want:  -5,
		},
		{
			name:  "zero value",
			key:   "TEST_INT_ZERO",
			def:   999,
			setup: func() { os.Setenv("TEST_INT_ZERO", "0") },
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer os.Unsetenv(tt.key)

			got := GetenvInt(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("GetenvInt(%q, %d) = %d, want %d", tt.key, tt.def, got, tt.want)
			}
		})
	}
}

func TestGetenvBool(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		def   bool
		setup func()
		want  bool
	}{
		{
			name:  "true lowercase",
			key:   "TEST_BOOL_TRUE1",
			def:   false,
			setup: func() { os.Setenv("TEST_BOOL_TRUE1", "true") },
			want:  true,
		},
		{
			name:  "True capitalized",
			key:   "TEST_BOOL_TRUE2",
			def:   false,
			setup: func() { os.Setenv("TEST_BOOL_TRUE2", "True") },
			want:  true,
		},
		{
			name:  "1 as true",
			key:   "TEST_BOOL_TRUE3",
			def:   false,
			setup: func() { os.Setenv("TEST_BOOL_TRUE3", "1") },
			want:  true,
		},
		{
			name:  "yes as true",
			key:   "TEST_BOOL_TRUE4",
			def:   false,
			setup: func() { os.Setenv("TEST_BOOL_TRUE4", "yes") },
			want:  true,
		},
		{
			name:  "false value",
			key:   "TEST_BOOL_FALSE1",
			def:   true,
			setup: func() { os.Setenv("TEST_BOOL_FALSE1", "false") },
			want:  false,
		},
		{
			name:  "0 as false",
			key:   "TEST_BOOL_FALSE2",
			def:   true,
			setup: func() { os.Setenv("TEST_BOOL_FALSE2", "0") },
			want:  false,
		},
		{
			name:  "missing returns default true",
			key:   "TEST_BOOL_MISSING1",
			def:   true,
			setup: func() { os.Unsetenv("TEST_BOOL_MISSING1") },
			want:  true,
		},
		{
			name:  "missing returns default false",
			key:   "TEST_BOOL_MISSING2",
			def:   false,
			setup: func() { os.Unsetenv("TEST_BOOL_MISSING2") },
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer os.Unsetenv(tt.key)

			got := GetenvBool(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("GetenvBool(%q, %v) = %v, want %v", tt.key, tt.def, got, tt.want)
			}
		})
	}
}

func TestDBConfigFromEnv(t *testing.T) {
	// Save original env
	orig := os.Getenv("AUTH_DATABASE_URL")
	defer func() {
		if orig != "" {
			os.Setenv("AUTH_DATABASE_URL", orig)
		} else {
			os.Unsetenv("AUTH_DATABASE_URL")
		}
	}()

	tests := []struct {
		name  string
		dsn   string
		setup func()
		check func(*testing.T, DBConfig)
	}{
		{
			name: "default DSN",
			dsn:  "postgres://user:pass@localhost:5432/db?sslmode=disable",
			setup: func() {
				os.Setenv("AUTH_DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
			},
			check: func(t *testing.T, cfg DBConfig) {
				if cfg.Host != "localhost" {
					t.Errorf("Host = %v, want localhost", cfg.Host)
				}
				if cfg.Port != 5432 {
					t.Errorf("Port = %v, want 5432", cfg.Port)
				}
				if cfg.User != "user" {
					t.Errorf("User = %v, want user", cfg.User)
				}
				if cfg.Password != "pass" {
					t.Errorf("Password = %v, want pass", cfg.Password)
				}
				if cfg.Database != "db" {
					t.Errorf("Database = %v, want db", cfg.Database)
				}
				if cfg.SSLMode != "disable" {
					t.Errorf("SSLMode = %v, want disable", cfg.SSLMode)
				}
			},
		},
		{
			name: "custom port",
			dsn:  "postgres://user:pass@db.example.com:5433/mydb",
			setup: func() {
				os.Setenv("AUTH_DATABASE_URL", "postgres://user:pass@db.example.com:5433/mydb")
			},
			check: func(t *testing.T, cfg DBConfig) {
				if cfg.Host != "db.example.com" {
					t.Errorf("Host = %v, want db.example.com", cfg.Host)
				}
				if cfg.Port != 5433 {
					t.Errorf("Port = %v, want 5433", cfg.Port)
				}
				if cfg.Database != "mydb" {
					t.Errorf("Database = %v, want mydb", cfg.Database)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			cfg := DBConfigFromEnv("AUTH_")
			tt.check(t, cfg)
		})
	}
}

func TestContextHelpers(t *testing.T) {
	// Test WithTenant and TenantFromContext
	ctx := WithTenant(nil, "tenant-123", "user-456", true)

	got := TenantFromContext(ctx)
	if got != "tenant-123" {
		t.Errorf("TenantFromContext() = %v, want tenant-123", got)
	}

	gotUser := UserFromContext(ctx)
	if gotUser != "user-456" {
		t.Errorf("UserFromContext() = %v, want user-456", gotUser)
	}

	isAdmin := IsAdminFromContext(ctx)
	if !isAdmin {
		t.Error("IsAdminFromContext() = false, want true")
	}

	// Test with non-admin context
	ctx = WithTenant(nil, "tenant-789", "user-000", false)
	isAdmin = IsAdminFromContext(ctx)
	if isAdmin {
		t.Error("IsAdminFromContext() = true, want false")
	}
}

func TestTraceHelpers(t *testing.T) {
	ctx := WithTraceID(nil, "trace-abc-123")

	got := TraceFromContext(ctx)
	if got != "trace-abc-123" {
		t.Errorf("TraceFromContext() = %v, want trace-abc-123", got)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrNotFound sentinel",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "error containing 'no rows'",
			err:  &testError{msg: "no rows in result"},
			want: true,
		},
		{
			name: "error containing 'not found'",
			err:  &testError{msg: "record not found"},
			want: true,
		},
		{
			name: "other error",
			err:  &testError{msg: "some other error"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestInitLogger(t *testing.T) {
	// Test that logger is created without panic
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")

	logger := InitLogger("test-service", "development", "1.0.0")
	if logger == nil {
		t.Error("InitLogger() returned nil")
	}
}
