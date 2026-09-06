// Package auth - UUID helper nội bộ.
package auth

import "github.com/google/uuid"

// uuidGen returns new UUID v7 string.
func uuidGen() string {
	return uuid.NewV7().String()
}