// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

//go:build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goabonga/infrastructure/internal/manager"
)

// TestExecDNSResolver starts a real dnsmasq for a VPC on a private loopback
// address and tears it down. It needs root and dnsmasq and skips otherwise.
func TestExecDNSResolver(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root")
	}
	if _, err := exec.LookPath("dnsmasq"); err != nil {
		t.Skip("dnsmasq not available")
	}

	ctx := context.Background()
	dir := t.TempDir()
	be := manager.NewExecDNS(dir)
	const vpc = "vpc-itest"
	t.Cleanup(func() { _ = be.StopResolver(ctx, vpc) })

	zones := []manager.DNSZoneConfig{{
		Domain: "itest.internal",
		Hosts:  []string{"10.0.1.10 web.itest.internal"},
	}}
	// 127.0.0.99 avoids the systemd-resolved stub on 127.0.0.53.
	if err := be.EnsureResolver(ctx, vpc, "127.0.0.99", zones); err != nil {
		t.Fatalf("ensure resolver: %v", err)
	}

	pidFile := filepath.Join(dir, vpc+".pid")
	if _, err := os.Stat(pidFile); err != nil {
		t.Fatalf("dnsmasq pid file missing: %v", err)
	}

	if err := be.StopResolver(ctx, vpc); err != nil {
		t.Fatalf("stop resolver: %v", err)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("pid file should be removed, err = %v", err)
	}
}
