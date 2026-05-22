// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Command infra-agent reconciles the resources in the local store against the
// host kernel. It requires root / CAP_NET_ADMIN to manage bridges.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/manager"
	"github.com/goabonga/infrastructure/internal/meta"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/state"
)

func main() {
	if err := run(); err != nil {
		slog.Error("infra-agent stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	stateDir := flag.String("state-dir", envOr("GOA_STATE_DIR", "./state"), "state directory")
	interval := flag.Duration("interval", 5*time.Second, "reconcile interval")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info(meta.Line("infra-agent", Version), "stateDir", *stateDir, "interval", interval.String())

	store := state.NewFileStore(*stateDir)
	vpcs := registry.New[resource.VPCSpec, resource.VPCStatus](store, resource.KindVPC)
	agent := manager.NewAgent(vpcs, manager.NewExecBackend(), *interval, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := agent.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
