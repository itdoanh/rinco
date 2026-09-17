// Package config loads environment-driven configuration for the
// webrtc-sfu service.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config bundles every knob the SFU reads from the environment.
type Config struct {
	HTTPAddr         string
	RTCAddr          string
	NATSURL          string
	ICEServers       []string
	MaxParticipants  int
	MaxRooms         int
	RoomTTL          time.Duration
	SDPSemantics     string
	RecordingHookURL string
	EnableConnectRPC bool
	EnableREST       bool
}

// FromEnv builds a Config from the process environment.
func FromEnv() Config {
	return Config{
		HTTPAddr:         getenv("SFU_HTTP_ADDR", ":8090"),
		RTCAddr:          getenv("SFU_RTC_ADDR", ":8091"),
		NATSURL:          getenv("SFU_NATS_URL", ""),
		ICEServers:       splitCSV(getenv("SFU_ICE_SERVERS", "stun:stun.l.google.com:19302")),
		MaxParticipants:  getenvInt("SFU_MAX_PARTICIPANTS", 50),
		MaxRooms:         getenvInt("SFU_MAX_ROOMS", 1000),
		RoomTTL:          getenvDuration("SFU_ROOM_TTL_SECS", 60*time.Minute),
		SDPSemantics:     getenv("SFU_SDP_SEMANTICS", "unified-plan"),
		RecordingHookURL: getenv("SFU_RECORDING_HOOK_URL", ""),
		EnableConnectRPC: getenvBool("SFU_ENABLE_CONNECT_RPC", true),
		EnableREST:       getenvBool("SFU_ENABLE_REST", true),
	}
}

// String returns a redacted summary suitable for logging on startup.
func (c Config) String() string {
	return fmt.Sprintf(
		"http=%s rtc=%s nats=%q ice=%d max_rooms=%d max_participants=%d sdp=%s",
		c.HTTPAddr, c.RTCAddr, c.NATSURL, len(c.ICEServers), c.MaxRooms, c.MaxParticipants, c.SDPSemantics,
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
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
