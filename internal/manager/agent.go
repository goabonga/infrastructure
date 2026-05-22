// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package manager

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Agent reconciles every VPC in the store on a fixed interval. A single bad
// resource is logged and skipped so it cannot stall the others.
type Agent struct {
	reg      *VPCRegistry
	rec      *VPCReconciler
	interval time.Duration
	logger   *slog.Logger
}

// NewAgent wires an Agent over reg and net. A nil logger uses slog.Default.
func NewAgent(reg *VPCRegistry, net NetworkBackend, interval time.Duration, logger *slog.Logger) *Agent {
	if logger == nil {
		logger = slog.Default()
	}
	return &Agent{
		reg:      reg,
		rec:      NewVPCReconciler(reg, net),
		interval: interval,
		logger:   logger,
	}
}

// ReconcileOnce reconciles every VPC currently in the store exactly once.
func (a *Agent) ReconcileOnce(ctx context.Context) error {
	vpcs, err := a.reg.List()
	if err != nil {
		return fmt.Errorf("manager: list vpcs: %w", err)
	}
	for i := range vpcs {
		uid := vpcs[i].Metadata.UID
		if err := a.rec.Reconcile(ctx, uid); err != nil {
			a.logger.ErrorContext(ctx, "reconcile vpc", "uid", uid, "err", err)
		}
	}
	return nil
}

// Run reconciles immediately and then on every tick until ctx is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	a.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			a.tick(ctx)
		}
	}
}

func (a *Agent) tick(ctx context.Context) {
	if err := a.ReconcileOnce(ctx); err != nil {
		a.logger.ErrorContext(ctx, "reconcile pass", "err", err)
	}
}
