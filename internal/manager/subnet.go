// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package manager

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/state"
)

// SubnetRegistry is the typed store the subnet reconciler reads and writes.
type SubnetRegistry = registry.Registry[resource.SubnetSpec, resource.SubnetStatus]

// SubnetReconciler realizes a subnet by assigning its gateway address to the
// parent VPC's bridge.
type SubnetReconciler struct {
	reg  *SubnetRegistry
	vpcs *VPCRegistry
	net  NetworkBackend
}

// NewSubnetReconciler returns a reconciler backed by reg, the VPC store and net.
func NewSubnetReconciler(reg *SubnetRegistry, vpcs *VPCRegistry, net NetworkBackend) *SubnetReconciler {
	return &SubnetReconciler{reg: reg, vpcs: vpcs, net: net}
}

// Name identifies the reconcile pass.
func (r *SubnetReconciler) Name() string { return resource.KindSubnet }

// ReconcileAll reconciles every subnet, collecting per-subnet errors.
func (r *SubnetReconciler) ReconcileAll(ctx context.Context) error {
	subnets, err := r.reg.List()
	if err != nil {
		return fmt.Errorf("manager: list subnets: %w", err)
	}
	var errs []error
	for i := range subnets {
		uid := subnets[i].Metadata.UID
		if err := r.Reconcile(ctx, uid); err != nil {
			errs = append(errs, fmt.Errorf("subnet %s: %w", uid, err))
		}
	}
	return errors.Join(errs...)
}

// Reconcile brings the subnet identified by uid in line with its spec.
func (r *SubnetReconciler) Reconcile(ctx context.Context, uid string) error {
	sn, err := r.reg.Get(uid)
	if errors.Is(err, state.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("manager: load subnet %q: %w", uid, err)
	}
	if sn.Metadata.IsDeleting() {
		return r.finalize(ctx, sn)
	}
	return r.ensure(ctx, sn)
}

func (r *SubnetReconciler) ensure(ctx context.Context, sn *resource.Subnet) error {
	if !sn.Metadata.HasFinalizer(resource.SubnetFinalizer) {
		sn.Metadata.AddFinalizer(resource.SubnetFinalizer)
	}

	bridge, ok, err := r.vpcBridge(sn.Spec.VPCID)
	if err != nil {
		sn.Status.SetPhase(resource.PhaseError, "VPCError", err.Error())
		_ = r.reg.Put(sn)
		return err
	}
	if !ok {
		// The VPC bridge is not provisioned yet; retry on the next pass.
		sn.Status.SetPhase(resource.PhasePending, "WaitingForVPC", "vpc bridge not ready")
		return r.reg.Put(sn)
	}

	gwCIDR, err := gatewayCIDR(sn.Spec.CIDR)
	if err != nil {
		sn.Status.SetPhase(resource.PhaseError, "BadCIDR", err.Error())
		_ = r.reg.Put(sn)
		return err
	}
	sn.Status.SetPhase(resource.PhaseReconciling, "Reconciling", "assigning gateway")
	if err := r.net.EnsureAddress(ctx, bridge, gwCIDR); err != nil {
		sn.Status.SetPhase(resource.PhaseError, "AddressError", err.Error())
		_ = r.reg.Put(sn)
		return err
	}

	sn.Status.Gateway = hostOf(gwCIDR)
	sn.Status.MarkReconciled(sn.Metadata.Generation)
	sn.Status.SetPhase(resource.PhaseReady, "Reconciled", "gateway assigned")
	if err := r.reg.Put(sn); err != nil {
		return fmt.Errorf("manager: save subnet %q: %w", sn.Metadata.UID, err)
	}
	return nil
}

func (r *SubnetReconciler) finalize(ctx context.Context, sn *resource.Subnet) error {
	if sn.Metadata.HasFinalizer(resource.SubnetFinalizer) {
		if bridge, ok, err := r.vpcBridge(sn.Spec.VPCID); err == nil && ok {
			if gwCIDR, gwErr := gatewayCIDR(sn.Spec.CIDR); gwErr == nil {
				if err := r.net.DeleteAddress(ctx, bridge, gwCIDR); err != nil {
					sn.Status.SetPhase(resource.PhaseError, "AddressError", err.Error())
					_ = r.reg.Put(sn)
					return err
				}
			}
		}
		sn.Metadata.RemoveFinalizer(resource.SubnetFinalizer)
		sn.Status.SetPhase(resource.PhaseDeleting, "Deleting", "gateway removed")
		if err := r.reg.Put(sn); err != nil {
			return fmt.Errorf("manager: save subnet %q: %w", sn.Metadata.UID, err)
		}
	}
	if len(sn.Metadata.Finalizers) == 0 {
		if err := r.reg.Delete(sn.Metadata.UID); err != nil {
			return fmt.Errorf("manager: delete subnet %q: %w", sn.Metadata.UID, err)
		}
	}
	return nil
}

// vpcBridge returns the bridge name of a VPC and whether it is provisioned.
func (r *SubnetReconciler) vpcBridge(vpcID string) (string, bool, error) {
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

// gatewayCIDR returns the first usable address of cidr in CIDR form, e.g.
// "10.0.1.0/24" -> "10.0.1.1/24".
func gatewayCIDR(cidr string) (string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid cidr %q: %w", cidr, err)
	}
	gw := make(net.IP, len(ipnet.IP))
	copy(gw, ipnet.IP)
	gw[len(gw)-1]++
	prefix, _ := ipnet.Mask.Size()
	return fmt.Sprintf("%s/%d", gw.String(), prefix), nil
}

// hostOf returns the address portion of an addr/prefix string.
func hostOf(addrCIDR string) string {
	if ip, _, err := net.ParseCIDR(addrCIDR); err == nil {
		return ip.String()
	}
	return addrCIDR
}
