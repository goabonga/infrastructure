// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Package resources implements the Terraform resources exposed by the provider.
// Each resource maps a Terraform schema onto the control-plane API client.
package resources

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/goabonga/infrastructure/internal/client"
	infra "github.com/goabonga/infrastructure/internal/domain/resource"
)

// vpcClient is the API client specialized to the VPC kind.
type vpcClient = client.Client[infra.VPCSpec, infra.VPCStatus]

// vpcModel is the Terraform state/plan shape for an infra_vpc resource.
type vpcModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CIDR       types.String `tfsdk:"cidr"`
	BridgeName types.String `tfsdk:"bridge_name"`
	Phase      types.String `tfsdk:"phase"`
}

// vpcResource is the infra_vpc Terraform resource.
type vpcResource struct {
	client *vpcClient
}

// Compile-time interface checks.
var (
	_ resource.Resource                = (*vpcResource)(nil)
	_ resource.ResourceWithConfigure   = (*vpcResource)(nil)
	_ resource.ResourceWithImportState = (*vpcResource)(nil)
)

// NewVPCResource is the resource factory registered with the provider.
func NewVPCResource() resource.Resource {
	return &vpcResource{}
}

func (r *vpcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

func (r *vpcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A virtual private cloud: an isolated Linux bridge fabric.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "System-assigned unique identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Display name.",
			},
			"cidr": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Address range in CIDR notation, e.g. 10.0.0.0/16.",
			},
			"bridge_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Linux bridge backing the VPC, set by the agent.",
			},
			"phase": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Lifecycle phase reported by the control plane.",
			},
		},
	}
}

func (r *vpcResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return // provider not configured yet
	}
	cfg, ok := req.ProviderData.(ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("expected ProviderConfig, got %T", req.ProviderData),
		)
		return
	}
	var opts []client.Option
	if cfg.Token != "" {
		opts = append(opts, client.WithToken(cfg.Token))
	}
	r.client = client.New[infra.VPCSpec, infra.VPCStatus](cfg.Endpoint, infra.KindVPC, opts...)
}

func (r *vpcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vpcModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := newID()
	if err != nil {
		resp.Diagnostics.AddError("Generate id", err.Error())
		return
	}
	out, err := r.client.Put(ctx, &infra.VPC{
		Metadata: infra.ObjectMeta{UID: id, Name: plan.Name.ValueString()},
		Spec:     infra.VPCSpec{CIDR: plan.CIDR.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError("Create VPC", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromVPC(out))...)
}

func (r *vpcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vpcModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.Get(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read VPC", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromVPC(out))...)
}

func (r *vpcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vpcModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.Put(ctx, &infra.VPC{
		Metadata: infra.ObjectMeta{UID: plan.ID.ValueString(), Name: plan.Name.ValueString()},
		Spec:     infra.VPCSpec{CIDR: plan.CIDR.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError("Update VPC", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromVPC(out))...)
}

func (r *vpcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vpcModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete VPC", err.Error())
	}
}

func (r *vpcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelFromVPC maps an API resource onto the Terraform model.
func modelFromVPC(v *infra.VPC) vpcModel {
	m := vpcModel{
		ID:         types.StringValue(v.Metadata.UID),
		CIDR:       types.StringValue(v.Spec.CIDR),
		BridgeName: types.StringValue(v.Status.BridgeName),
		Phase:      types.StringValue(string(v.Status.Phase)),
	}
	if v.Metadata.Name == "" {
		m.Name = types.StringNull()
	} else {
		m.Name = types.StringValue(v.Metadata.Name)
	}
	return m
}

// newID returns a fresh "vpc-<hex>" identifier.
func newID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("resources: generate id: %w", err)
	}
	return "vpc-" + hex.EncodeToString(b), nil
}
