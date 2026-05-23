// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package manager_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/manager"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/state"
)

type fakeDNSBackend struct {
	ensured map[string][]manager.DNSZoneConfig
	listen  map[string]string
	stopped []string
}

func newFakeDNSBackend() *fakeDNSBackend {
	return &fakeDNSBackend{ensured: map[string][]manager.DNSZoneConfig{}, listen: map[string]string{}}
}

func (f *fakeDNSBackend) EnsureResolver(_ context.Context, vpcID, listenAddr string, zones []manager.DNSZoneConfig) error {
	f.ensured[vpcID] = zones
	f.listen[vpcID] = listenAddr
	return nil
}

func (f *fakeDNSBackend) StopResolver(_ context.Context, vpcID string) error {
	f.stopped = append(f.stopped, vpcID)
	return nil
}

type dnsEnv struct {
	zones   *manager.DNSZoneRegistry
	records *manager.DNSRecordRegistry
	vpcs    *manager.VPCRegistry
}

func newDNSEnv(t *testing.T) *dnsEnv {
	t.Helper()
	store := state.NewFileStore(t.TempDir())
	return &dnsEnv{
		zones:   registry.New[resource.DNSZoneSpec, resource.DNSZoneStatus](store, resource.KindDNSZone),
		records: registry.New[resource.DNSRecordSpec, resource.DNSRecordStatus](store, resource.KindDNSRecord),
		vpcs:    registry.New[resource.VPCSpec, resource.VPCStatus](store, resource.KindVPC),
	}
}

func (env *dnsEnv) putVPC(t *testing.T, uid, cidr, bridge string) {
	t.Helper()
	v := &resource.VPC{Metadata: resource.ObjectMeta{UID: uid, Generation: 1}, Spec: resource.VPCSpec{CIDR: cidr}}
	v.Status.BridgeName = bridge
	if err := env.vpcs.Put(v); err != nil {
		t.Fatalf("seed vpc: %v", err)
	}
}

func (env *dnsEnv) putZone(t *testing.T, uid, domain string, vpcIDs ...string) {
	t.Helper()
	if err := env.zones.Put(&resource.DNSZone{
		Metadata: resource.ObjectMeta{UID: uid, Generation: 1},
		Spec:     resource.DNSZoneSpec{Domain: domain, Visibility: "private", VPCIDs: vpcIDs},
	}); err != nil {
		t.Fatalf("seed zone: %v", err)
	}
}

func (env *dnsEnv) putRecord(t *testing.T, uid, zoneID, name, typ string, values ...string) {
	t.Helper()
	if err := env.records.Put(&resource.DNSRecord{
		Metadata: resource.ObjectMeta{UID: uid, Generation: 1},
		Spec:     resource.DNSRecordSpec{ZoneID: zoneID, Name: name, Type: typ, Records: values},
	}); err != nil {
		t.Fatalf("seed record: %v", err)
	}
}

func TestDNSReconcileServesZones(t *testing.T) {
	t.Parallel()

	env := newDNSEnv(t)
	env.putVPC(t, "vpc-1", "10.0.0.0/16", "br-vpc1")
	env.putZone(t, "z-1", "example.internal", "vpc-1")
	env.putRecord(t, "r-1", "z-1", "web", "A", "10.0.1.10")
	env.putRecord(t, "r-2", "z-1", "db", "AAAA", "fd00::1")
	env.putRecord(t, "r-cname", "z-1", "alias", "CNAME", "web.example.internal") // not in hosts

	be := newFakeDNSBackend()
	rec := manager.NewDNSReconciler(env.zones, env.records, env.vpcs, be)
	if err := rec.ReconcileAll(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	cfgs, ok := be.ensured["vpc-1"]
	if !ok || len(cfgs) != 1 {
		t.Fatalf("vpc-1 not served: %+v", be.ensured)
	}
	if be.listen["vpc-1"] != "10.0.0.1" {
		t.Fatalf("listen addr = %q, want 10.0.0.1", be.listen["vpc-1"])
	}
	if cfgs[0].Domain != "example.internal" {
		t.Fatalf("domain = %q", cfgs[0].Domain)
	}
	joined := strings.Join(cfgs[0].Hosts, "\n")
	if !strings.Contains(joined, "10.0.1.10 web.example.internal") || !strings.Contains(joined, "fd00::1 db.example.internal") {
		t.Fatalf("host lines missing: %v", cfgs[0].Hosts)
	}
	if strings.Contains(joined, "alias") {
		t.Fatalf("CNAME should not appear in hosts: %v", cfgs[0].Hosts)
	}

	z, _ := env.zones.Get("z-1")
	if !z.Status.IsReady() {
		t.Fatalf("zone phase = %q, want Ready", z.Status.Phase)
	}
	r, _ := env.records.Get("r-1")
	if !r.Status.IsReady() {
		t.Fatalf("record phase = %q, want Ready", r.Status.Phase)
	}
	if rec.Name() != resource.KindDNSZone {
		t.Fatalf("name = %q", rec.Name())
	}
}

func TestDNSPendingWhenBridgeMissing(t *testing.T) {
	t.Parallel()

	env := newDNSEnv(t)
	env.putVPC(t, "vpc-1", "10.0.0.0/16", "") // no bridge yet
	env.putZone(t, "z-1", "example.internal", "vpc-1")

	be := newFakeDNSBackend()
	rec := manager.NewDNSReconciler(env.zones, env.records, env.vpcs, be)
	if err := rec.ReconcileAll(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(be.ensured) != 0 {
		t.Fatalf("no resolver should be ensured: %+v", be.ensured)
	}
	z, _ := env.zones.Get("z-1")
	if z.Status.Phase != resource.PhasePending {
		t.Fatalf("zone phase = %q, want Pending", z.Status.Phase)
	}
}

func TestDNSStopsResolverWhenNoZones(t *testing.T) {
	t.Parallel()

	env := newDNSEnv(t)
	env.putVPC(t, "vpc-1", "10.0.0.0/16", "br-vpc1") // no zones attached

	be := newFakeDNSBackend()
	rec := manager.NewDNSReconciler(env.zones, env.records, env.vpcs, be)
	if err := rec.ReconcileAll(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(be.stopped) != 1 || be.stopped[0] != "vpc-1" {
		t.Fatalf("expected vpc-1 resolver stopped, got %v", be.stopped)
	}
}

func TestDNSPublicZoneNeedsNoResolver(t *testing.T) {
	t.Parallel()

	env := newDNSEnv(t)
	env.putZone(t, "z-1", "public.example.com") // no VPCs

	be := newFakeDNSBackend()
	rec := manager.NewDNSReconciler(env.zones, env.records, env.vpcs, be)
	if err := rec.ReconcileAll(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(be.ensured) != 0 {
		t.Fatalf("public zone needs no resolver: %+v", be.ensured)
	}
	z, _ := env.zones.Get("z-1")
	if !z.Status.IsReady() {
		t.Fatalf("public zone phase = %q, want Ready", z.Status.Phase)
	}
}

func TestExecDNSWritesHostsAndStarts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rec := &fwRecorder{}
	available := func(string) (string, error) { return "/usr/sbin/dnsmasq", nil }
	be := manager.NewExecDNSWithRunner(dir, rec.run, available)

	zones := []manager.DNSZoneConfig{{Domain: "example.internal", Hosts: []string{"10.0.1.10 web.example.internal"}}}
	if err := be.EnsureResolver(context.Background(), "vpc-1", "10.0.0.1", zones); err != nil {
		t.Fatalf("ensure resolver: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "vpc-1.hosts"))
	if err != nil {
		t.Fatalf("read hosts: %v", err)
	}
	if !strings.Contains(string(data), "10.0.1.10 web.example.internal") {
		t.Fatalf("hosts file missing record: %q", data)
	}
	for _, want := range []string{"--local=/example.internal/", "--no-resolv", "--listen-address=10.0.0.1"} {
		if !anyCallHas(rec.calls, want) {
			t.Fatalf("missing %q in dnsmasq args: %v", want, rec.calls)
		}
	}

	if err := be.StopResolver(context.Background(), "vpc-1"); err != nil {
		t.Fatalf("stop resolver: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "vpc-1.hosts")); !os.IsNotExist(err) {
		t.Fatalf("hosts file should be removed, err = %v", err)
	}
}
