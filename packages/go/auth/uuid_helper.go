// Package auth - UUID helper nội bộ.
package auth

import (
	"github.com/google/uuid"
)

// uuidGen returns new UUID v7 string. Falls back to v4 if v7 is unavailable.
func uuidGen() string {
	if u, err := uuid.NewV7(); err == nil {
		return u.String()
	}
	return uuid.New().String()
}