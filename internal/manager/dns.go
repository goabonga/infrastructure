// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/registry"
)

// DNSZoneConfig is the realized configuration of one zone served on a VPC: its
// domain and the host lines (A/AAAA records) for its addn-hosts file.
type DNSZoneConfig struct {
	Domain string
	Hosts  []string
}

// DNSBackend abstracts the per-VPC resolver the DNS reconciler drives.
type DNSBackend interface {
	// EnsureResolver writes the combined hosts file for a VPC and ensures a
	// dnsmasq serving listenAddr is running for the given zones, reloading it when
	// already up. Idempotent.
	EnsureResolver(ctx context.Context, vpcID, listenAddr string, zones []DNSZoneConfig) error
	// StopResolver stops the VPC's resolver and removes its files. Idempotent.
	StopResolver(ctx context.Context, vpcID string) error
}

// ExecDNS is a DNSBackend that drives a per-VPC dnsmasq. It requires root and
// dnsmasq at run time; without dnsmasq it still writes the hosts files.
type ExecDNS struct {
	dir      string
	run      Runner
	lookPath func(string) (string, error)
}

// NewExecDNS stores hosts and pid files under dir.
func NewExecDNS(dir string) *ExecDNS {
	return &ExecDNS{dir: dir, run: defaultRun, lookPath: exec.LookPath}
}

// NewExecDNSWithRunner is the test constructor: it overrides the runner and the
// dnsmasq lookup so the issued commands can be asserted without the daemon.
func NewExecDNSWithRunner(dir string, run Runner, lookPath func(string) (string, error)) *ExecDNS {
	return &ExecDNS{dir: dir, run: run, lookPath: lookPath}
}

func (d *ExecDNS) hostsPath(vpcID string) string { return filepath.Join(d.dir, vpcID+".hosts") }
func (d *ExecDNS) pidPath(vpcID string) string   { return filepath.Join(d.dir, vpcID+".pid") }

// EnsureResolver writes the hosts file then starts or reloads dnsmasq.
func (d *ExecDNS) EnsureResolver(ctx context.Context, vpcID, listenAddr string, zones []DNSZoneConfig) error {
	if err := os.MkdirAll(d.dir, 0o750); err != nil {
		return fmt.Errorf("manager: dns dir: %w", err)
	}
	var lines []string
	for _, z := range zones {
		lines = append(lines, z.Hosts...)
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(d.hostsPath(vpcID), []byte(content), 0o600); err != nil {
		return fmt.Errorf("manager: write dns hosts: %w", err)
	}

	if _, err := d.lookPath("dnsmasq"); err != nil {
		// dnsmasq is unavailable; the hosts file is written for a later start.
		return nil
	}

	pidFile := d.pidPath(vpcID)
	if data, err := os.ReadFile(pidFile); err == nil { // #nosec G304 -- agent-owned pid path
		if pid := strings.TrimSpace(string(data)); pid != "" {
			if _, err := d.run(ctx, "kill", "-0", pid); err == nil {
				_, _ = d.run(ctx, "kill", "-HUP", pid)
				return nil
			}
		}
	}

	args := []string{
		"--listen-address=" + listenAddr,
		"--bind-dynamic",
		"--addn-hosts=" + d.hostsPath(vpcID),
		"--pid-file=" + pidFile,
		"--no-resolv",
		"--server=8.8.8.8",
		"--server=1.1.1.1",
	}
	for _, z := range zones {
		args = append(args, "--domain="+z.Domain, "--local=/"+z.Domain+"/")
	}
	if out, err := d.run(ctx, "dnsmasq", args...); err != nil {
		return fmt.Errorf("manager: start dnsmasq for %s: %w: %s", vpcID, err, strings.TrimSpace(out))
	}
	return nil
}

// StopResolver kills the VPC's dnsmasq and removes its files.
func (d *ExecDNS) StopResolver(ctx context.Context, vpcID string) error {
	pidFile := d.pidPath(vpcID)
	if data, err := os.ReadFile(pidFile); err == nil { // #nosec G304 -- agent-owned pid path
		if pid := strings.TrimSpace(string(data)); pid != "" {
			_, _ = d.run(ctx, "kill", pid)
		}
	}
	_ = os.Remove(pidFile)
	_ = os.Remove(d.hostsPath(vpcID))
	return nil
}

// DNSZoneRegistry is the typed store of DNS zones.
type DNSZoneRegistry = registry.Registry[resource.DNSZoneSpec, resource.DNSZoneStatus]

// DNSRecordRegistry is the typed store of DNS records.
type DNSRecordRegistry = registry.Registry[resource.DNSRecordSpec, resource.DNSRecordStatus]

// DNSReconciler realizes DNS by running one dnsmasq per VPC, serving the zones
// attached to that VPC and their A/AAAA records. It reconciles at VPC
// granularity from live state, so deleting a zone or record (which carry no
// finalizer) simply regenerates the affected resolver on the next pass.
type DNSReconciler struct {
	zones   *DNSZoneRegistry
	records *DNSRecordRegistry
	vpcs    *VPCRegistry
	backend DNSBackend
}

// NewDNSReconciler returns a reconciler backed by the zone, record and VPC
// stores and the DNS backend.
func NewDNSReconciler(zones *DNSZoneRegistry, records *DNSRecordRegistry, vpcs *VPCRegistry, backend DNSBackend) *DNSReconciler {
	return &DNSReconciler{zones: zones, records: records, vpcs: vpcs, backend: backend}
}

// Name identifies the reconcile pass.
func (r *DNSReconciler) Name() string { return resource.KindDNSZone }

// ReconcileAll ensures every VPC's resolver matches the live zones and records.
func (r *DNSReconciler) ReconcileAll(ctx context.Context) error {
	vpcs, err := r.vpcs.List()
	if err != nil {
		return fmt.Errorf("manager: list vpcs: %w", err)
	}
	zones, err := r.zones.List()
	if err != nil {
		return fmt.Errorf("manager: list dns zones: %w", err)
	}
	records, err := r.records.List()
	if err != nil {
		return fmt.Errorf("manager: list dns records: %w", err)
	}

	hostsByZone := make(map[string][]string)
	for i := range records {
		rec := &records[i]
		if rec.Metadata.IsDeleting() {
			continue
		}
		if rec.Spec.Type != "A" && rec.Spec.Type != "AAAA" {
			continue
		}
		zone := zoneByUID(zones, rec.Spec.ZoneID)
		if zone == nil {
			continue
		}
		fqdn := recordFQDN(rec.Spec.Name, zone.Spec.Domain)
		for _, val := range rec.Spec.Records {
			hostsByZone[rec.Spec.ZoneID] = append(hostsByZone[rec.Spec.ZoneID], val+" "+fqdn)
		}
	}

	zonesByVPC := make(map[string][]*resource.DNSZone)
	for i := range zones {
		z := &zones[i]
		if z.Metadata.IsDeleting() {
			continue
		}
		for _, vpcID := range z.Spec.VPCIDs {
			zonesByVPC[vpcID] = append(zonesByVPC[vpcID], z)
		}
	}

	vpcReady := make(map[string]bool)
	var errs []error
	for i := range vpcs {
		v := &vpcs[i]
		attached := zonesByVPC[v.Metadata.UID]
		if len(attached) == 0 {
			if err := r.backend.StopResolver(ctx, v.Metadata.UID); err != nil {
				errs = append(errs, fmt.Errorf("vpc %s dns: %w", v.Metadata.UID, err))
			}
			continue
		}
		if v.Status.BridgeName == "" {
			vpcReady[v.Metadata.UID] = false
			continue
		}
		cfgs := make([]DNSZoneConfig, 0, len(attached))
		for _, z := range attached {
			cfgs = append(cfgs, DNSZoneConfig{Domain: z.Spec.Domain, Hosts: hostsByZone[z.Metadata.UID]})
		}
		if err := r.backend.EnsureResolver(ctx, v.Metadata.UID, firstHostOf(v.Spec.CIDR), cfgs); err != nil {
			errs = append(errs, fmt.Errorf("vpc %s dns: %w", v.Metadata.UID, err))
			vpcReady[v.Metadata.UID] = false
			continue
		}
		vpcReady[v.Metadata.UID] = true
	}

	r.updateStatuses(zones, records, vpcReady)
	return errors.Join(errs...)
}

// updateStatuses records each zone's and record's phase, writing only on change.
func (r *DNSReconciler) updateStatuses(zones []resource.DNSZone, records []resource.DNSRecord, vpcReady map[string]bool) {
	zonePhase := make(map[string]resource.Phase)
	for i := range zones {
		z := &zones[i]
		if z.Metadata.IsDeleting() {
			continue
		}
		phase, reason, msg := zonePhaseFor(z, vpcReady)
		zonePhase[z.Metadata.UID] = phase
		if z.Status.Phase == phase {
			continue
		}
		z.Status.SetPhase(phase, reason, msg)
		if phase == resource.PhaseReady {
			z.Status.MarkReconciled(z.Metadata.Generation)
		}
		_ = r.zones.Put(z)
	}
	for i := range records {
		rec := &records[i]
		if rec.Metadata.IsDeleting() {
			continue
		}
		phase := resource.PhasePending
		reason, msg := "WaitingForZone", "zone not ready"
		if p, ok := zonePhase[rec.Spec.ZoneID]; ok && p == resource.PhaseReady {
			phase, reason, msg = resource.PhaseReady, "Served", "record served"
		}
		if rec.Status.Phase == phase {
			continue
		}
		rec.Status.SetPhase(phase, reason, msg)
		if phase == resource.PhaseReady {
			rec.Status.MarkReconciled(rec.Metadata.Generation)
		}
		_ = r.records.Put(rec)
	}
}

// zonePhaseFor derives a zone's phase from the readiness of its VPCs.
func zonePhaseFor(z *resource.DNSZone, vpcReady map[string]bool) (resource.Phase, string, string) {
	if len(z.Spec.VPCIDs) == 0 {
		return resource.PhaseReady, "NoResolver", "public zone; no private resolver"
	}
	for _, vpcID := range z.Spec.VPCIDs {
		if !vpcReady[vpcID] {
			return resource.PhasePending, "WaitingForVPC", "vpc resolver not ready"
		}
	}
	return resource.PhaseReady, "Served", "zone served"
}

// zoneByUID finds a zone by UID, or nil.
func zoneByUID(zones []resource.DNSZone, uid string) *resource.DNSZone {
	for i := range zones {
		if zones[i].Metadata.UID == uid {
			return &zones[i]
		}
	}
	return nil
}

// recordFQDN joins a record name and zone domain into a fully qualified name.
func recordFQDN(name, domain string) string {
	switch {
	case name == "" || name == "@":
		return domain
	case strings.HasSuffix(name, "."):
		return strings.TrimSuffix(name, ".")
	default:
		return name + "." + domain
	}
}
