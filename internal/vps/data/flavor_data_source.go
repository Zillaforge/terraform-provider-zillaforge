// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/helper"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FlavorDataSource{}

func NewFlavorDataSource() datasource.DataSource { return &FlavorDataSource{} }

// FlavorDataSource defines the flavor data source implementation.
type FlavorDataSource struct {
	client *cloudsdk.ProjectClient
}

func (d *FlavorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flavors"
}

func (d *FlavorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query available compute flavors in Zillaforge VPS service.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter flavors by exact name match (case-sensitive)",
				Optional:            true,
			},
			"vcpus": schema.Int64Attribute{
				MarkdownDescription: "Filter flavors with minimum number of vCPUs",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"memory": schema.Int64Attribute{
				MarkdownDescription: "Filter flavors with minimum memory in GB",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"flavors": schema.ListNestedAttribute{
				MarkdownDescription: "List of matching flavor objects.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{MarkdownDescription: "Unique flavor id", Computed: true},
						"name":        schema.StringAttribute{MarkdownDescription: "Human readable flavor name", Computed: true},
						"vcpus":       schema.Int64Attribute{MarkdownDescription: "Virtual CPUs", Computed: true},
						"memory":      schema.Int64Attribute{MarkdownDescription: "Memory in GB", Computed: true},
						"disk":        schema.Int64Attribute{MarkdownDescription: "Root disk size in GB", Computed: true},
						"description": schema.StringAttribute{MarkdownDescription: "Optional description", Computed: true},
					},
				},
			},
		},
	}
}

func (d *FlavorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	// Provider Configure sets the SDK project client as DataSourceData
	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		d.client = projectClient
	} else {
		// Not the expected type - leave as nil to return empty result safely
		d.client = nil
	}
}

func (d *FlavorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data model.FlavorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Basic input validation for filter fields (compatibility for old Go toolchain)
	if !data.VCPUs.IsNull() && data.VCPUs.ValueInt64() < 1 {
		resp.Diagnostics.AddError("Invalid vcpus filter", "vcpus must be >= 1")
		return
	}
	if !data.Memory.IsNull() && data.Memory.ValueInt64() < 1 {
		resp.Diagnostics.AddError("Invalid memory filter", "memory must be >= 1")
		return
	}

	// If client not configured, return empty list (but not error) to avoid failing plan
	if d.client == nil {
		// Save empty list to state
		data.Flavors = []model.FlavorModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Use typed SDK calls from cloud-sdk
	fls, err := helper.ListFlavorsWithSDK(ctx, d.client, data)
	if err != nil {
		resp.Diagnostics.AddError("Flavors list error", fmt.Sprintf("Failed to list flavors using SDK: %s", err))
		data.Flavors = []model.FlavorModel{}
	} else {
		data.Flavors = fls
	}

	// Save state
	if diag := resp.State.Set(ctx, &data); diag.HasError() {
		resp.Diagnostics.AddError("Failed to set state", fmt.Sprintf("Could not set flavor state: %v", diag))
	}

	tflog.Trace(ctx, "Read zillaforge_flavors data source")
}
