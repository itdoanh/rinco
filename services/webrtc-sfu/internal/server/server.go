// Package server wires together every component of the SFU.
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

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/config"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/connectrpc"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/httpapi"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/nats"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/peer"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/room"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/sfu"
)

// Server bundles every wiring of the SFU.
type Server struct {
	Config config.Config
	Logger *slog.Logger

	Rooms     *room.Manager
	Peers     *peer.Manager
	Forwarder *sfu.Forwarder
	NATS      nats.Publisher
}

// New builds a fully-wired Server.
func New(cfg config.Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return &Server{
		Config:    cfg,
		Logger:    logger,
		Rooms:     room.NewManager(cfg.MaxRooms),
		Peers:     peer.NewManager(),
		Forwarder: sfu.NewForwarder(),
		NATS:      nats.New(cfg.NATSURL),
	}
}

// Run blocks until ctx is cancelled or a fatal error occurs.
func (s *Server) Run(ctx context.Context) error {
	restMux := httpapi.New(s.Rooms, s.Peers, s.Forwarder, s.NATS).Routes()

	rpcMux := http.NewServeMux()
	sfuSvc := connectrpc.NewSFUService(s.Rooms, s.Peers, s.Forwarder, s.NATS)
	connectrpc.MountRoutes(rpcMux, sfuSvc)

	httpSrv := &http.Server{
		Addr:              s.Config.HTTPAddr,
		Handler:           withRecover(restMux, s.Logger),
		ReadHeaderTimeout: 5 * time.Second,
	}
	rpcSrv := &http.Server{
		Addr:              s.Config.RTCAddr,
		Handler:           withRecover(rpcMux, s.Logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Periodic cleanup of empty rooms.
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.Rooms.CleanupEmpty()
			}
		}
	}()

	errCh := make(chan error, 2)
	go func() {
		defer wg.Done()
		s.Logger.Info("http server listening", "addr", s.Config.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		s.Logger.Info("connect-rpc server listening", "addr", s.Config.RTCAddr)
		if err := rpcSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("rpc: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		s.Logger.Info("shutdown signal received")
	case err := <-errCh:
		s.Logger.Error("listener error", "error", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	_ = rpcSrv.Shutdown(shutdownCtx)
	wg.Wait()
	s.NATS.Close()
	s.Logger.Info("shutdown complete")
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

// Ensure connect is referenced for the import.
var _ = connect.NewResponse[struct{}]
