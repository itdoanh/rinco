// Command webrtc-sfu is the RINCO WebRTC SFU binary.
//
// The SFU exposes REST endpoints under /v1/* and Connect-RPC procedures
// under /rinco.sfu.v1.*. The transport is pluggable: the binary ships with
// an in-process forwarder, but a Pion-based pipeline can be swapped in by
// adding a PeerConnection manager that pipes the RTP packets from the
// connect-go Subscribe stream into Pion tracks.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/config"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "webrtc-sfu: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.FromEnv()
	logger := defaultLogger(cfg)
	logger.Info("starting webrtc-sfu", "config", cfg.String(), "version", "0.3.0")
	srv := server.New(cfg, logger)
	return srv.Run(context.Background())
}
