// Command chat-engine is the RINCO chat-engine binary.
//
// It serves a Connect-RPC surface (ChatService / GroupService /
// ConversationService) and a REST fallback under /v1/*.
//
// The binary is designed to start in three modes:
//
//  1. Production mode: ScyllaDB + Valkey + NATS endpoints configured via
//     the CHAT_* environment variables.
//  2. Hybrid mode: the same as production but the ScyllaDB / Valkey
//     connections may be unreachable; the engine falls back to the
//     in-process memory store so the binary still serves traffic.
//  3. Development mode: CHAT_IN_MEMORY_MODE=true forces the in-process
//     store regardless of any external configuration.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/itdoanh/rinco/services/chat-engine/internal/chat"
	"github.com/itdoanh/rinco/services/chat-engine/internal/config"
	"github.com/itdoanh/rinco/services/chat-engine/internal/conversation"
	"github.com/itdoanh/rinco/services/chat-engine/internal/group"
	"github.com/itdoanh/rinco/services/chat-engine/internal/nats"
	"github.com/itdoanh/rinco/services/chat-engine/internal/presence"
	"github.com/itdoanh/rinco/services/chat-engine/internal/server"
	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/storage/memory"
	"github.com/itdoanh/rinco/services/chat-engine/internal/storage/scylla"
	valkeystore "github.com/itdoanh/rinco/services/chat-engine/internal/storage/valkey"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "chat-engine: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.FromEnv()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("starting chat-engine",
		"config", cfg.String(),
		"version", "0.3.0",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var store storage.Aggregator
	var presenceStore storage.PresenceStore

	if cfg.EnableInMemoryMode || (len(cfg.ScyllaURL) == 0 && cfg.ValkeyURL == "") {
		logger.Warn("running in in-memory mode (no ScyllaDB / Valkey configured)")
		mem := memory.New()
		store = mem
		valkey := valkeystore.Open(ctx, cfg.ValkeyURL)
		presenceStore = valkey
	} else {
		// ScyllaDB
		sc, err := scylla.Open(ctx, cfg.ScyllaURL, cfg.ScyllaKeyspace)
		if err != nil {
			logger.Warn("scylla open failed; falling back to memory store", "error", err)
			store = memory.New()
		} else {
			store = sc
		}
		valkey := valkeystore.Open(ctx, cfg.ValkeyURL)
		presenceStore = valkey
	}

	pub := nats.New(cfg.NATSURL)

	typingTTL := int(cfg.TypingTTL.Seconds())
	if typingTTL <= 0 {
		typingTTL = 5
	}

	chatSvc := chat.New(store.Messages(), store.Conversations(), presenceStore, pub, typingTTL, cfg.MaxHistoryLimit)
	groupSvc := group.New(store.Messages(), store.Groups(), store.Conversations(), presenceStore, pub, cfg.MaxHistoryLimit)
	convSvc := conversation.New(store.Conversations(), presenceStore)
	presenceSvc := presence.New(presenceStore, pub, int(cfg.PresenceTTL.Seconds()))

	deps := server.Deps{
		Chat:     chatSvc,
		Group:    groupSvc,
		Conv:     convSvc,
		Presence: presenceSvc,
		NATS:     pub,
	}

	srvCfg := server.Config{
		HTTPAddr:        cfg.HTTPAddr,
		RPCAddr:         cfg.ConnectRPCAddr,
		ShutdownTimeout: cfg.ShutdownTimeout,
	}

	if err := server.Run(ctx, srvCfg, deps, logger); err != nil {
		return err
	}
	// Brief sleep to allow graceful shutdown to drain.
	time.Sleep(50 * time.Millisecond)
	return nil
}
