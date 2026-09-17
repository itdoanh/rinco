// Package server wires together every component of the chat-engine and
// exposes Run() to start HTTP + Connect-RPC.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"

	"github.com/itdoanh/rinco/services/chat-engine/internal/chat"
	"github.com/itdoanh/rinco/services/chat-engine/internal/conversation"
	"github.com/itdoanh/rinco/services/chat-engine/internal/group"
	"github.com/itdoanh/rinco/services/chat-engine/internal/nats"
	"github.com/itdoanh/rinco/services/chat-engine/internal/presence"
)

// Deps bundles the wired-up services.
type Deps struct {
	Chat     *chat.Service
	Group    *group.Service
	Conv     *conversation.Service
	Presence *presence.Service
	NATS     nats.Publisher
}

// Config is a stripped-down configuration used by Run.
type Config struct {
	HTTPAddr        string
	RPCAddr         string
	ShutdownTimeout time.Duration
}

// Run starts every listener and blocks until ctx is cancelled or a fatal
// error occurs.
func Run(ctx context.Context, cfg Config, deps Deps, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	restHandler := newRESTHandler(deps)
	rpcHandler := newRPCHandler(deps)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           withRecover(restHandler, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}
	rpcSrv := &http.Server{
		Addr:              cfg.RPCAddr,
		Handler:           withRecover(rpcHandler, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)
	go func() {
		defer wg.Done()
		logger.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		logger.Info("connect-rpc server listening", "addr", cfg.RPCAddr)
		if err := rpcSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("rpc: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("listener error", "error", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown", "error", err)
	}
	if err := rpcSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("rpc shutdown", "error", err)
	}
	wg.Wait()
	if deps.NATS != nil {
		deps.NATS.Close()
	}
	logger.Info("shutdown complete")
	return nil
}

func withRecover(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"path", r.URL.Path,
					"method", r.Method,
					"panic", rec,
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Ensure unused imports are referenced.
var (
	_ = connect.NewResponse[struct{}]
)
