// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Package manager reconciles declarative resources against Linux kernel
// primitives. Kernel access sits behind interfaces so the reconciliation logic
// is testable without root; the real backends shell out to iproute2 and only
// run on a privileged host.
package manager

import (
	"context"
	"fmt"
	"hash/fnv"
	"os/exec"
	"strings"
)

// maxIfaceName is the Linux interface name length limit (IFNAMSIZ - 1).
const maxIfaceName = 15

// Bridge describes the desired state of a Linux bridge.
type Bridge struct {
	// Name is the kernel interface name (<= 15 chars).
	Name string
	// CIDR is the VPC address range the bridge fabricates (informational at the
	// bridge level; subnets assign addresses).
	CIDR string
}

// NetworkBackend abstracts the kernel network operations the agent needs.
type NetworkBackend interface {
	// EnsureBridge creates the bridge if absent and brings it up. Idempotent.
	EnsureBridge(ctx context.Context, b Bridge) error
	// DeleteBridge removes the bridge. Deleting an absent bridge is not an error.
	DeleteBridge(ctx context.Context, name string) error
	// BridgeExists reports whether the named bridge is present.
	BridgeExists(ctx context.Context, name string) (bool, error)
}

// bridgeName derives a valid, deterministic bridge interface name from a UID,
// hashing when a sanitized "br-<uid>" would exceed the kernel length limit.
func bridgeName(uid string) string {
	var b strings.Builder
	for _, r := range uid {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	name := "br-" + b.String()
	if len(name) <= maxIfaceName {
		return name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(uid))
	return fmt.Sprintf("br-%08x", h.Sum32())
}

// Runner runs an external command and returns its combined output.
type Runner func(ctx context.Context, name string, args ...string) (string, error)

func defaultRun(ctx context.Context, name string, args ...string) (string, error) {
	// #nosec G204 -- the agent intentionally shells out to iproute2 with a
	// fixed command name and arguments it controls, not user input.
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// ExecBackend is a NetworkBackend that shells out to iproute2 (`ip`). It
// requires root / CAP_NET_ADMIN at run time.
type ExecBackend struct {
	run Runner
}

// NewExecBackend returns an ExecBackend using the real `ip` command.
func NewExecBackend() *ExecBackend {
	return &ExecBackend{run: defaultRun}
}

// NewExecBackendWithRunner returns an ExecBackend driven by a custom runner,
// used in tests to assert the issued commands without touching the kernel.
func NewExecBackendWithRunner(run Runner) *ExecBackend {
	return &ExecBackend{run: run}
}

// BridgeExists reports whether `ip link show <name>` finds the bridge.
func (b *ExecBackend) BridgeExists(ctx context.Context, name string) (bool, error) {
	out, err := b.run(ctx, "ip", "link", "show", name)
	if err == nil {
		return true, nil
	}
	if strings.Contains(out, "does not exist") || strings.Contains(out, "Cannot find device") {
		return false, nil
	}
	return false, fmt.Errorf("manager: bridge exists %q: %w: %s", name, err, strings.TrimSpace(out))
}

// EnsureBridge creates the bridge if needed and brings it up.
func (b *ExecBackend) EnsureBridge(ctx context.Context, br Bridge) error {
	exists, err := b.BridgeExists(ctx, br.Name)
	if err != nil {
		return err
	}
	if !exists {
		if out, err := b.run(ctx, "ip", "link", "add", "name", br.Name, "type", "bridge"); err != nil {
			return fmt.Errorf("manager: add bridge %q: %w: %s", br.Name, err, strings.TrimSpace(out))
		}
	}
	if out, err := b.run(ctx, "ip", "link", "set", br.Name, "up"); err != nil {
		return fmt.Errorf("manager: set bridge %q up: %w: %s", br.Name, err, strings.TrimSpace(out))
	}
	return nil
}

// DeleteBridge removes the bridge if present.
func (b *ExecBackend) DeleteBridge(ctx context.Context, name string) error {
	exists, err := b.BridgeExists(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if out, err := b.run(ctx, "ip", "link", "del", name); err != nil {
		return fmt.Errorf("manager: delete bridge %q: %w: %s", name, err, strings.TrimSpace(out))
	}
	return nil
}
