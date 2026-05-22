// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	infra "github.com/goabonga/infrastructure/internal/domain/resource"
)

func TestNewID(t *testing.T) {
	t.Parallel()

	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	if !strings.HasPrefix(id, "vpc-") {
		t.Fatalf("id %q missing vpc- prefix", id)
	}
	if len(id) != len("vpc-")+12 { // 6 random bytes -> 12 hex chars
		t.Fatalf("id %q has unexpected length %d", id, len(id))
	}
	other, _ := newID()
	if id == other {
		t.Fatal("newID should not repeat")
	}
}

func TestModelFromVPC(t *testing.T) {
	t.Parallel()

	v := &infra.VPC{
		Metadata: infra.ObjectMeta{UID: "vpc-1", Name: "prod"},
		Spec:     infra.VPCSpec{CIDR: "10.0.0.0/16"},
	}
	v.Status.BridgeName = "br-1"
	v.Status.Phase = infra.PhaseReady

	m := modelFromVPC(v)
	if m.ID.ValueString() != "vpc-1" {
		t.Fatalf("ID = %q", m.ID.ValueString())
	}
	if m.Name.ValueString() != "prod" {
		t.Fatalf("Name = %q", m.Name.ValueString())
	}
	if m.CIDR.ValueString() != "10.0.0.0/16" {
		t.Fatalf("CIDR = %q", m.CIDR.ValueString())
	}
	if m.BridgeName.ValueString() != "br-1" {
		t.Fatalf("BridgeName = %q", m.BridgeName.ValueString())
	}
	if m.Phase.ValueString() != "Ready" {
		t.Fatalf("Phase = %q", m.Phase.ValueString())
	}
}

func TestModelFromVPCEmptyNameIsNull(t *testing.T) {
	t.Parallel()

	m := modelFromVPC(&infra.VPC{Metadata: infra.ObjectMeta{UID: "vpc-2"}})
	if !m.Name.IsNull() {
		t.Fatalf("empty name should map to null, got %q", m.Name.ValueString())
	}
}

func TestVPCResourceMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewVPCResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "infra"}, &resp)
	if resp.TypeName != "infra_vpc" {
		t.Fatalf("TypeName = %q, want infra_vpc", resp.TypeName)
	}
}

func TestVPCResourceSchema(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewVPCResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	attrs := resp.Schema.Attributes

	if !attrs["cidr"].IsRequired() {
		t.Fatal("cidr should be required")
	}
	if !attrs["name"].IsOptional() {
		t.Fatal("name should be optional")
	}
	for _, computed := range []string{"id", "bridge_name", "phase"} {
		if !attrs[computed].IsComputed() {
			t.Fatalf("%s should be computed", computed)
		}
	}
}
