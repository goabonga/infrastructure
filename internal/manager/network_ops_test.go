// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package manager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/goabonga/infrastructure/internal/manager"
)

// netRecorder records commands; iptables -C checks report "not present" so the
// add path runs, and `ip route show default` returns a canned line.
type netRecorder struct {
	calls    [][]string
	routeOut string
}

func (r *netRecorder) run(_ context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if name == "ip" && len(args) >= 3 && args[0] == "route" && args[1] == "show" && args[2] == "default" {
		return r.routeOut, nil
	}
	for _, a := range args {
		if a == "-C" {
			return "", errors.New("rule does not exist")
		}
	}
	return "", nil
}

func TestExecBackendNetworkOps(t *testing.T) {
	t.Parallel()

	rec := &netRecorder{routeOut: "default via 192.168.1.1 dev wan0 proto dhcp src 192.168.1.50"}
	be := manager.NewExecBackendWithRunner(rec.run)
	ctx := context.Background()

	ifc, err := be.DefaultInterface(ctx)
	if err != nil || ifc != "wan0" {
		t.Fatalf("DefaultInterface = %q, %v", ifc, err)
	}
	if err := be.EnsureAddress(ctx, "br0", "10.0.1.1/24"); err != nil {
		t.Fatalf("EnsureAddress: %v", err)
	}
	if err := be.EnableForwarding(ctx); err != nil {
		t.Fatalf("EnableForwarding: %v", err)
	}
	if err := be.EnsureNAT(ctx, "10.0.0.0/16", "wan0"); err != nil {
		t.Fatalf("EnsureNAT: %v", err)
	}

	for _, want := range [][]string{
		{"ip", "addr", "add", "10.0.1.1/24", "dev", "br0"},
		{"sysctl", "-w", "net.ipv4.ip_forward=1"},
		{"iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "10.0.0.0/16", "-o", "wan0", "-j", "MASQUERADE"},
	} {
		if !sawCall(rec.calls, want...) {
			t.Fatalf("missing call %v in %v", want, rec.calls)
		}
	}
}
