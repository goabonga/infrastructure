// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Package provider implements the Terraform provider for the control-plane API.
// It exposes the API resources as Terraform-managed resources, sharing the same
// HTTP client the CLI uses.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/goabonga/infrastructure/internal/resources"
)

// defaultEndpoint is used when the provider block sets no endpoint.
const defaultEndpoint = "http://localhost:8080"

// infraProvider is the Terraform provider implementation.
type infraProvider struct {
	version string
}

var _ provider.Provider = (*infraProvider)(nil)

// New returns the provider factory used by providerserver.Serve.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &infraProvider{version: version}
	}
}

func (p *infraProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "infra"
	resp.Version = p.version
}

type providerModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
}

func (p *infraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage declarative infrastructure through the control-plane API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Base URL of the API (default " + defaultEndpoint + ").",
			},
		},
	}
}

func (p *infraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint := defaultEndpoint
	if !cfg.Endpoint.IsNull() && cfg.Endpoint.ValueString() != "" {
		endpoint = cfg.Endpoint.ValueString()
	}
	// Resources build their own typed client from the endpoint string.
	resp.ResourceData = endpoint
}

func (p *infraProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewVPCResource,
	}
}

func (p *infraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
