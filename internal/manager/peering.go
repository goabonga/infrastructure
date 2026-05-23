// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package manager

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/state"
)

// PeeringBackend abstracts the veth operations that link two VPC bridges.
type PeeringBackend interface {
	// EnsureLink creates a veth pair joining bridge1 and bridge2 if absent and
	// brings it up. Idempotent.
	EnsureLink(ctx context.Context, veth1, veth2, bridge1, bridge2 string) error
	// DeleteLink removes the veth pair (deleting one end removes both). Removing
	// an absent link is not an error.
	DeleteLink(ctx context.Context, veth1 string) error
}

// peeringNames derives the two veth interface names for a peering UID, within
// the kernel interface-name limit.
func peeringNames(uid string) (veth1, veth2 string) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(uid))
	s := fmt.Sprintf("%08x", h.Sum32())
	return "pa-" + s, "pb-" + s
}

// ExecPeering is a PeeringBackend that shells out to iproute2 (`ip`). It
// requires root / CAP_NET_ADMIN at run time.
type ExecPeering struct {
	run Runner
}

// NewExecPeering returns an ExecPeering using the real `ip` command.
func NewExecPeering() *ExecPeering {
	return &ExecPeering{run: defaultRun}
}

// NewExecPeeringWithRunner returns a backend driven by a custom runner, used in
// tests to assert the issued commands without touching the kernel.
func NewExecPeeringWithRunner(run Runner) *ExecPeering {
	return &ExecPeering{run: run}
}

// linkExists reports whether the named interface is present.
func (p *ExecPeering) linkExists(ctx context.Context, name string) (bool, error) {
	out, err := p.run(ctx, "ip", "link", "show", name)
	if err == nil {
		return true, nil
	}
	if strings.Contains(out, "does not exist") || strings.Contains(out, "Cannot find device") {
		return false, nil
	}
	return false, fmt.Errorf("manager: link exists %q: %w: %s", name, err, strings.TrimSpace(out))
}

// EnsureLink creates and attaches the veth pair if the first end is absent.
func (p *ExecPeering) EnsureLink(ctx context.Context, veth1, veth2, bridge1, bridge2 string) error {
	exists, err := p.linkExists(ctx, veth1)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	steps := [][]string{
		{"link", "add", veth1, "type", "veth", "peer", "name", veth2},
		{"link", "set", veth1, "master", bridge1},
		{"link", "set", veth1, "up"},
		{"link", "set", veth2, "master", bridge2},
		{"link", "set", veth2, "up"},
	}
	for _, s := range steps {
		if out, err := p.run(ctx, "ip", s...); err != nil {
			return fmt.Errorf("manager: peering link %v: %w: %s", s, err, strings.TrimSpace(out))
		}
	}
	return nil
}

// DeleteLink removes the veth pair by deleting its first end.
func (p *ExecPeering) DeleteLink(ctx context.Context, veth1 string) error {
	out, err := p.run(ctx, "ip", "link", "del", veth1)
	if err != nil && !strings.Contains(out, "does not exist") && !strings.Contains(out, "Cannot find device") {
		return fmt.Errorf("manager: delete peering link %q: %w: %s", veth1, err, strings.TrimSpace(out))
	}
	return nil
}

// PeeringRegistry is the typed store the peering reconciler reads and writes.
type PeeringRegistry = registry.Registry[resource.PeeringSpec, resource.PeeringStatus]

// PeeringReconciler realizes a peering by linking the two VPC bridges with a
// veth pair.
type PeeringReconciler struct {
	reg     *PeeringRegistry
	vpcs    *VPCRegistry
	backend PeeringBackend
}

// NewPeeringReconciler returns a reconciler backed by reg, the VPC store and the
// peering backend.
func NewPeeringReconciler(reg *PeeringRegistry, vpcs *VPCRegistry, backend PeeringBackend) *PeeringReconciler {
	return &PeeringReconciler{reg: reg, vpcs: vpcs, backend: backend}
}

// Name identifies the reconcile pass.
func (r *PeeringReconciler) Name() string { return resource.KindPeering }

// ReconcileAll reconciles every peering, collecting per-peering errors.
func (r *PeeringReconciler) ReconcileAll(ctx context.Context) error {
	peerings, err := r.reg.List()
	if err != nil {
		return fmt.Errorf("manager: list peerings: %w", err)
	}
	var errs []error
	for i := range peerings {
		uid := peerings[i].Metadata.UID
		if err := r.Reconcile(ctx, uid); err != nil {
			errs = append(errs, fmt.Errorf("peering %s: %w", uid, err))
		}
	}
	return errors.Join(errs...)
}

// Reconcile brings the peering identified by uid in line with its spec.
func (r *PeeringReconciler) Reconcile(ctx context.Context, uid string) error {
	pr, err := r.reg.Get(uid)
	if errors.Is(err, state.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("manager: load peering %q: %w", uid, err)
	}
	if pr.Metadata.IsDeleting() {
		return r.finalize(ctx, pr)
	}
	return r.ensure(ctx, pr)
}

func (r *PeeringReconciler) ensure(ctx context.Context, pr *resource.Peering) error {
	if !pr.Metadata.HasFinalizer(resource.PeeringFinalizer) {
		pr.Metadata.AddFinalizer(resource.PeeringFinalizer)
	}

	bridge1, ok1, err := r.vpcBridge(pr.Spec.VPC1ID)
	if err != nil {
		pr.Status.SetPhase(resource.PhaseError, "VPCError", err.Error())
		_ = r.reg.Put(pr)
		return err
	}
	bridge2, ok2, err := r.vpcBridge(pr.Spec.VPC2ID)
	if err != nil {
		pr.Status.SetPhase(resource.PhaseError, "VPCError", err.Error())
		_ = r.reg.Put(pr)
		return err
	}
	if !ok1 || !ok2 {
		// A VPC bridge is not provisioned yet; retry on the next pass.
		pr.Status.SetPhase(resource.PhasePending, "WaitingForVPC", "vpc bridge not ready")
		return r.reg.Put(pr)
	}

	veth1, veth2 := peeringNames(pr.Metadata.UID)
	pr.Status.SetPhase(resource.PhaseReconciling, "Reconciling", "linking bridges")
	if err := r.backend.EnsureLink(ctx, veth1, veth2, bridge1, bridge2); err != nil {
		pr.Status.SetPhase(resource.PhaseError, "LinkError", err.Error())
		_ = r.reg.Put(pr)
		return err
	}

	pr.Status.Veth1 = veth1
	pr.Status.Veth2 = veth2
	pr.Status.MarkReconciled(pr.Metadata.Generation)
	pr.Status.SetPhase(resource.PhaseReady, "Linked", "bridges linked")
	if err := r.reg.Put(pr); err != nil {
		return fmt.Errorf("manager: save peering %q: %w", pr.Metadata.UID, err)
	}
	return nil
}

func (r *PeeringReconciler) finalize(ctx context.Context, pr *resource.Peering) error {
	if pr.Metadata.HasFinalizer(resource.PeeringFinalizer) {
		veth1, _ := peeringNames(pr.Metadata.UID)
		if err := r.backend.DeleteLink(ctx, veth1); err != nil {
			pr.Status.SetPhase(resource.PhaseError, "LinkError", err.Error())
			_ = r.reg.Put(pr)
			return err
		}
		pr.Metadata.RemoveFinalizer(resource.PeeringFinalizer)
		pr.Status.SetPhase(resource.PhaseDeleting, "Deleting", "link removed")
		if err := r.reg.Put(pr); err != nil {
			return fmt.Errorf("manager: save peering %q: %w", pr.Metadata.UID, err)
		}
	}
	if len(pr.Metadata.Finalizers) == 0 {
		if err := r.reg.Delete(pr.Metadata.UID); err != nil {
			return fmt.Errorf("manager: delete peering %q: %w", pr.Metadata.UID, err)
		}
	}
	return nil
}

// vpcBridge returns the bridge name of a VPC and whether it is provisioned.
func (r *PeeringReconciler) vpcBridge(vpcID string) (string, bool, error) {
	vpc, err := r.vpcs.Get(vpcID)
	if errors.Is(err, state.ErrNotFound) {
		return "", false, fmt.Errorf("vpc %q not found", vpcID)
	}
	if err != nil {
		return "", false, err
	}
	if vpc.Status.BridgeName == "" {
		return "", false, nil
	}
	return vpc.Status.BridgeName, true, nil
}
