// Package sqlc chứa code được generate bởi sqlc từ queries.sql + schema.sql.
//
// Để generate, cd vào folder này và chạy:
//   sqlc generate
//
// File này là stub demo; production sẽ được replace bởi generated code.
package sqlc

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User model mirror schema.sql.
type User struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Email         string
	PasswordHash  *string
	FullName      *string
	Roles         []string
	IsSuperAdmin  bool
	MFAEnabled    bool
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

// Querier interface cho queries - sqlc sẽ generate implementation.
type Querier interface {
	GetUser(ctx context.Context, id uuid.UUID) (User, error)
	GetUserByEmail(ctx context.Context, tenantID uuid.UUID, email string) (User, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	UpdateUserLastLogin(ctx context.Context, id uuid.UUID) error
	RevokeRefreshTokensByUser(ctx context.Context, userID uuid.UUID) error
}

// CreateUserParams parameter struct cho CreateUser query.
type CreateUserParams struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Email        string
	PasswordHash *string
	FullName     *string
	Roles        []string
}