// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewID(t *testing.T) {
	t.Parallel()

	id, err := newID("vpc")
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	if !strings.HasPrefix(id, "vpc-") {
		t.Fatalf("id %q missing kind prefix", id)
	}
	if len(id) != len("vpc-")+12 { // 6 random bytes -> 12 hex chars
		t.Fatalf("id %q has unexpected length %d", id, len(id))
	}
	if other, _ := newID("vpc"); id == other {
		t.Fatal("newID should not repeat")
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
	for _, computed := range []string{"id", "bridge_name", "phase"} {
		if !attrs[computed].IsComputed() {
			t.Fatalf("%s should be computed", computed)
		}
	}
}
