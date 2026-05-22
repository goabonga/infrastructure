// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	infra "github.com/goabonga/infrastructure/internal/domain/resource"
)

type ipAddressModel struct {
	ID        types.String `tfsdk:"id"`
	Type      types.String `tfsdk:"type"`
	SubnetID  types.String `tfsdk:"subnet_id"`
	VPCID     types.String `tfsdk:"vpc_id"`
	ComputeID types.String `tfsdk:"compute_id"`
	Address   types.String `tfsdk:"address"`
	Phase     types.String `tfsdk:"phase"`
}

// NewIPAddressResource is the infra_ip_address resource factory.
func NewIPAddressResource() resource.Resource {
	return newGeneric(resourceDef[ipAddressModel, infra.IPAddressSpec, infra.IPAddressStatus]{
		kind: infra.KindIPAddress,
		schema: schema.Schema{
			MarkdownDescription: "A reserved IP address.",
			Attributes: map[string]schema.Attribute{
				"id":         idAttribute(),
				"type":       schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "private or public."},
				"subnet_id":  schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Subnet to allocate from."},
				"vpc_id":     schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "VPC scope."},
				"compute_id": schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Compute to bind to."},
				"address":    schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Requested or resolved address."},
				"phase":      phaseAttribute(),
			},
		},
		toSpec: func(_ context.Context, m ipAddressModel) (infra.IPAddressSpec, diag.Diagnostics) {
			return infra.IPAddressSpec{
				Type:      m.Type.ValueString(),
				SubnetID:  m.SubnetID.ValueString(),
				VPCID:     m.VPCID.ValueString(),
				ComputeID: m.ComputeID.ValueString(),
				Address:   m.Address.ValueString(),
			}, nil
		},
		toModel: func(_ context.Context, r *infra.IPAddress) (ipAddressModel, diag.Diagnostics) {
			address := r.Status.Address
			if address == "" {
				address = r.Spec.Address
			}
			return ipAddressModel{
				ID:        types.StringValue(r.Metadata.UID),
				Type:      types.StringValue(r.Spec.Type),
				SubnetID:  types.StringValue(r.Spec.SubnetID),
				VPCID:     types.StringValue(r.Spec.VPCID),
				ComputeID: types.StringValue(r.Spec.ComputeID),
				Address:   types.StringValue(address),
				Phase:     types.StringValue(string(r.Status.Phase)),
			}, nil
		},
		id: func(m ipAddressModel) string { return m.ID.ValueString() },
	})
}

type igwModel struct {
	ID        types.String `tfsdk:"id"`
	VPCID     types.String `tfsdk:"vpc_id"`
	HostIface types.String `tfsdk:"host_iface"`
	Bridge    types.String `tfsdk:"bridge"`
	Phase     types.String `tfsdk:"phase"`
}

// NewIGWResource is the infra_igw resource factory.
func NewIGWResource() resource.Resource {
	return newGeneric(resourceDef[igwModel, infra.IGWSpec, infra.IGWStatus]{
		kind: infra.KindIGW,
		schema: schema.Schema{
			MarkdownDescription: "An internet gateway for a VPC.",
			Attributes: map[string]schema.Attribute{
				"id":         idAttribute(),
				"vpc_id":     schema.StringAttribute{Required: true, MarkdownDescription: "Parent VPC id."},
				"host_iface": schema.StringAttribute{Computed: true, MarkdownDescription: "Host interface used for egress."},
				"bridge":     schema.StringAttribute{Computed: true, MarkdownDescription: "Bridge backing the gateway."},
				"phase":      phaseAttribute(),
			},
		},
		toSpec: func(_ context.Context, m igwModel) (infra.IGWSpec, diag.Diagnostics) {
			return infra.IGWSpec{VPCID: m.VPCID.ValueString()}, nil
		},
		toModel: func(_ context.Context, r *infra.IGW) (igwModel, diag.Diagnostics) {
			return igwModel{
				ID:        types.StringValue(r.Metadata.UID),
				VPCID:     types.StringValue(r.Spec.VPCID),
				HostIface: types.StringValue(r.Status.HostIface),
				Bridge:    types.StringValue(r.Status.Bridge),
				Phase:     types.StringValue(string(r.Status.Phase)),
			}, nil
		},
		id: func(m igwModel) string { return m.ID.ValueString() },
	})
}

type routeModel struct {
	ID          types.String `tfsdk:"id"`
	VPCID       types.String `tfsdk:"vpc_id"`
	SubnetID    types.String `tfsdk:"subnet_id"`
	Destination types.String `tfsdk:"destination"`
	Gateway     types.String `tfsdk:"gateway"`
	Phase       types.String `tfsdk:"phase"`
}

// NewRouteResource is the infra_route resource factory.
func NewRouteResource() resource.Resource {
	return newGeneric(resourceDef[routeModel, infra.RouteSpec, infra.RouteStatus]{
		kind: infra.KindRoute,
		schema: schema.Schema{
			MarkdownDescription: "A static route within a VPC.",
			Attributes: map[string]schema.Attribute{
				"id":          idAttribute(),
				"vpc_id":      schema.StringAttribute{Required: true, MarkdownDescription: "Parent VPC id."},
				"subnet_id":   schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Subnet scope (optional)."},
				"destination": schema.StringAttribute{Required: true, MarkdownDescription: "Destination CIDR."},
				"gateway":     schema.StringAttribute{Required: true, MarkdownDescription: "Target (igw id, local, or peer)."},
				"phase":       phaseAttribute(),
			},
		},
		toSpec: func(_ context.Context, m routeModel) (infra.RouteSpec, diag.Diagnostics) {
			return infra.RouteSpec{
				VPCID:       m.VPCID.ValueString(),
				SubnetID:    m.SubnetID.ValueString(),
				Destination: m.Destination.ValueString(),
				Gateway:     m.Gateway.ValueString(),
			}, nil
		},
		toModel: func(_ context.Context, r *infra.Route) (routeModel, diag.Diagnostics) {
			return routeModel{
				ID:          types.StringValue(r.Metadata.UID),
				VPCID:       types.StringValue(r.Spec.VPCID),
				SubnetID:    types.StringValue(r.Spec.SubnetID),
				Destination: types.StringValue(r.Spec.Destination),
				Gateway:     types.StringValue(r.Spec.Gateway),
				Phase:       types.StringValue(string(r.Status.Phase)),
			}, nil
		},
		id: func(m routeModel) string { return m.ID.ValueString() },
	})
}
