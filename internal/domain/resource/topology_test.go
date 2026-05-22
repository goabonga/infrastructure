// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package resource_test

import (
	"testing"

	"github.com/goabonga/infrastructure/internal/domain/resource"
)

func TestTopologySpecsSatisfyValidator(t *testing.T) {
	t.Parallel()

	var (
		_ resource.Validator = resource.SubnetSpec{}
		_ resource.Validator = resource.SecurityGroupSpec{}
		_ resource.Validator = resource.SecurityGroupRuleSpec{}
		_ resource.Validator = resource.IGWSpec{}
		_ resource.Validator = resource.RouteSpec{}
		_ resource.Validator = resource.KMSKeyringSpec{}
		_ resource.Validator = resource.KMSKeySpec{}
		_ resource.Validator = resource.DiskSpec{}
		_ resource.Validator = resource.DiskFileSpec{}
		_ resource.Validator = resource.ComputeSpec{}
	)
}

func TestTopologyValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    resource.Validator
		wantErr bool
	}{
		{"subnet ok", resource.SubnetSpec{VPCID: "vpc-1", CIDR: "10.0.1.0/24", Type: "public"}, false},
		{"subnet no vpc", resource.SubnetSpec{CIDR: "10.0.1.0/24"}, true},
		{"subnet bad cidr", resource.SubnetSpec{VPCID: "vpc-1", CIDR: "x"}, true},
		{"subnet bad type", resource.SubnetSpec{VPCID: "vpc-1", CIDR: "10.0.1.0/24", Type: "dmz"}, true},
		{"sg ok", resource.SecurityGroupSpec{VPCID: "vpc-1"}, false},
		{"sg no vpc", resource.SecurityGroupSpec{}, true},
		{"rule ok", resource.SecurityGroupRuleSpec{SecurityGroupID: "sg-1", Direction: "ingress", Protocol: "tcp", Port: 443}, false},
		{"rule bad dir", resource.SecurityGroupRuleSpec{SecurityGroupID: "sg-1", Direction: "x", Protocol: "tcp"}, true},
		{"igw ok", resource.IGWSpec{VPCID: "vpc-1"}, false},
		{"route ok", resource.RouteSpec{VPCID: "vpc-1", Destination: "0.0.0.0/0", Gateway: "igw-1"}, false},
		{"route bad dest", resource.RouteSpec{VPCID: "vpc-1", Destination: "x", Gateway: "igw-1"}, true},
		{"keyring ok", resource.KMSKeyringSpec{Name: "prod"}, false},
		{"key no ring", resource.KMSKeySpec{Name: "k"}, true},
		{"disk ok", resource.DiskSpec{SizeMB: 1024, KMSKeyID: "key-1"}, false},
		{"disk bad size", resource.DiskSpec{SizeMB: 0}, true},
		{"diskfile ok", resource.DiskFileSpec{DiskID: "disk-1", Path: "/etc/x"}, false},
		{"compute ok", resource.ComputeSpec{SubnetID: "sn-1", Image: "nginx:latest"}, false},
		{"compute no subnet", resource.ComputeSpec{Image: "nginx"}, true},
		{"compute no image", resource.ComputeSpec{SubnetID: "sn-1"}, true},
		{"compute bad disk", resource.ComputeSpec{SubnetID: "sn-1", Image: "nginx", Disks: []resource.ComputeDiskRef{{DiskID: "d"}}}, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.spec.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
