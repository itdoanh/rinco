// Package auth - crypto helpers nội bộ (không export).
package auth

import "crypto/sha256"

// sha256sumImpl implementation local.
func sha256sumImpl(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}