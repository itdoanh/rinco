// Package config loads environment-driven configuration for chat-engine.
//
// All knobs are read from environment variables with safe defaults so the
// binary can be started in development without any extra setup.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the runtime configuration for chat-engine.
type Config struct {
	HTTPAddr           string
	ConnectRPCAddr     string
	ScyllaURL          []string
	ScyllaKeyspace     string
	ValkeyURL          string
	NATSURL            string
	OTLPEndpoint       string
	ShutdownTimeout    time.Duration
	HeartbeatInterval  time.Duration
	TypingTTL          time.Duration
	PresenceTTL        time.Duration
	MaxHistoryLimit    int
	EnableConnectRPC   bool
	EnableRESTFallback bool
	EnableInMemoryMode bool
}

// FromEnv builds a Config from the process environment.
func FromEnv() Config {
	return Config{
		HTTPAddr:           getenv("CHAT_HTTP_ADDR", ":8080"),
		ConnectRPCAddr:     getenv("CHAT_RPC_ADDR", ":8082"),
		ScyllaURL:          splitCSV(getenv("CHAT_SCYLLA_URL", "")),
		ScyllaKeyspace:     getenv("CHAT_SCYLLA_KEYSPACE", "rinco_chat"),
		ValkeyURL:          getenv("CHAT_VALKEY_URL", ""),
		NATSURL:            getenv("CHAT_NATS_URL", ""),
		OTLPEndpoint:       getenv("CHAT_OTLP_ENDPOINT", ""),
		ShutdownTimeout:    getenvDuration("CHAT_SHUTDOWN_TIMEOUT_SECS", 30*time.Second),
		HeartbeatInterval:  getenvDuration("CHAT_HEARTBEAT_INTERVAL_SECS", 30*time.Second),
		TypingTTL:          getenvDuration("CHAT_TYPING_TTL_SECS", 5*time.Second),
		PresenceTTL:        getenvDuration("CHAT_PRESENCE_TTL_SECS", 90*time.Second),
		MaxHistoryLimit:    getenvInt("CHAT_MAX_HISTORY_LIMIT", 200),
		EnableConnectRPC:   getenvBool("CHAT_ENABLE_CONNECT_RPC", true),
		EnableRESTFallback: getenvBool("CHAT_ENABLE_REST_FALLBACK", true),
		EnableInMemoryMode: getenvBool("CHAT_IN_MEMORY_MODE", false),
	}
}

// String returns a redacted summary suitable for logging on startup.
func (c Config) String() string {
	return fmt.Sprintf(
		"http=%s rpc=%s scylla=%v keyspace=%s valkey=%q nats=%q in_memory=%t",
		c.HTTPAddr, c.ConnectRPCAddr, c.ScyllaURL, c.ScyllaKeyspace,
		c.ValkeyURL, c.NATSURL, c.EnableInMemoryMode,
	)
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
