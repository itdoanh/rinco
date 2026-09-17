// Package integration re-exports a few helpers used across multiple test
// files in this directory.
//
//go:build integration

package integration

import "encoding/json"

// jsonUnmarshal is a tiny alias to keep test files compact.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
