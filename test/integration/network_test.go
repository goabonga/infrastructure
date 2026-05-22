// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

//go:build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/goabonga/infrastructure/internal/manager"
)

// TestExecBackendRealBridge exercises the real iproute2 backend against the
// host kernel. It needs root / CAP_NET_ADMIN and the `ip` command, and skips
// cleanly when either is missing.
func TestExecBackendRealBridge(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root / CAP_NET_ADMIN")
	}
	if _, err := exec.LookPath("ip"); err != nil {
		t.Skip("iproute2 (ip) not available")
	}

	ctx := context.Background()
	be := manager.NewExecBackend()
	const name = "br-itest0"
	t.Cleanup(func() { _ = be.DeleteBridge(ctx, name) })

	if err := be.EnsureBridge(ctx, manager.Bridge{Name: name, CIDR: "10.255.0.0/24"}); err != nil {
		t.Fatalf("ensure bridge: %v", err)
	}
	ok, err := be.BridgeExists(ctx, name)
	if err != nil || !ok {
		t.Fatalf("bridge should exist: ok=%v err=%v", ok, err)
	}

	if err := be.DeleteBridge(ctx, name); err != nil {
		t.Fatalf("delete bridge: %v", err)
	}
	ok, err = be.BridgeExists(ctx, name)
	if err != nil || ok {
		t.Fatalf("bridge should be gone: ok=%v err=%v", ok, err)
	}
}
