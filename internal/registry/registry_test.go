// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package registry_test

import (
	"errors"
	"sort"
	"testing"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/state"
)

type vpcSpec struct {
	CIDR string `json:"cidr"`
}

type vpcStatus struct {
	resource.StatusBase
	AssignedCIDR string `json:"assignedCidr"`
}

func newRegistry(t *testing.T) *registry.Registry[vpcSpec, vpcStatus] {
	t.Helper()
	return registry.New[vpcSpec, vpcStatus](state.NewFileStore(t.TempDir()), "vpc")
}

func TestRegistryPutGetRoundtrip(t *testing.T) {
	t.Parallel()

	r := newRegistry(t)
	in := &resource.Resource[vpcSpec, vpcStatus]{
		Metadata: resource.ObjectMeta{UID: "vpc-1", Name: "prod"},
		Spec:     vpcSpec{CIDR: "10.0.0.0/16"},
	}
	in.Status.SetPhase(resource.PhaseReady, "Created", "ok")

	if err := r.Put(in); err != nil {
		t.Fatalf("put: %v", err)
	}

	out, err := r.Get("vpc-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if out.APIVersion != resource.APIVersion {
		t.Fatalf("APIVersion = %q, want %q", out.APIVersion, resource.APIVersion)
	}
	if out.Kind != "vpc" {
		t.Fatalf("Kind = %q, want vpc", out.Kind)
	}
	if out.Spec.CIDR != "10.0.0.0/16" {
		t.Fatalf("Spec.CIDR = %q", out.Spec.CIDR)
	}
	if !out.Status.IsReady() {
		t.Fatal("expected Ready status to round-trip")
	}
}

func TestRegistryPutValidation(t *testing.T) {
	t.Parallel()

	r := newRegistry(t)
	if err := r.Put(nil); err == nil {
		t.Fatal("expected error for nil resource")
	}
	if err := r.Put(&resource.Resource[vpcSpec, vpcStatus]{}); err == nil {
		t.Fatal("expected error for empty UID")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	t.Parallel()

	r := newRegistry(t)
	if _, err := r.Get("nope"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRegistryList(t *testing.T) {
	t.Parallel()

	r := newRegistry(t)
	for _, uid := range []string{"vpc-1", "vpc-2"} {
		res := &resource.Resource[vpcSpec, vpcStatus]{
			Metadata: resource.ObjectMeta{UID: uid},
			Spec:     vpcSpec{CIDR: "10.0.0.0/16"},
		}
		if err := r.Put(res); err != nil {
			t.Fatalf("put %s: %v", uid, err)
		}
	}

	items, err := r.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	uids := make([]string, 0, len(items))
	for _, it := range items {
		uids = append(uids, it.Metadata.UID)
	}
	sort.Strings(uids)
	if len(uids) != 2 || uids[0] != "vpc-1" || uids[1] != "vpc-2" {
		t.Fatalf("unexpected uids: %v", uids)
	}
}

func TestRegistryDelete(t *testing.T) {
	t.Parallel()

	r := newRegistry(t)
	res := &resource.Resource[vpcSpec, vpcStatus]{
		Metadata: resource.ObjectMeta{UID: "vpc-1"},
	}
	if err := r.Put(res); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := r.Delete("vpc-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := r.Get("vpc-1"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
