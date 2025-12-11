// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	flavorsmodels "github.com/Zillaforge/cloud-sdk/models/vps/flavors"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FlavorDataSource{}

func NewFlavorDataSource() datasource.DataSource { return &FlavorDataSource{} }

// FlavorDataSource defines the flavor data source implementation.
type FlavorDataSource struct {
	client *cloudsdk.ProjectClient
}

// FlavorDataSourceModel describes the data source data model.
type FlavorDataSourceModel struct {
	Name   types.String `tfsdk:"name"`
	VCPUs  types.Int64  `tfsdk:"vcpus"`
	Memory types.Int64  `tfsdk:"memory"`

	Flavors []FlavorModel `tfsdk:"flavors"`
}

// FlavorModel represents a single flavor computed in state.
type FlavorModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	VCPUs       types.Int64  `tfsdk:"vcpus"`
	Memory      types.Int64  `tfsdk:"memory"`
	Disk        types.Int64  `tfsdk:"disk"`
	Description types.String `tfsdk:"description"`
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
	var data FlavorDataSourceModel
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
		data.Flavors = []FlavorModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Use typed SDK calls from cloud-sdk
	fls, err := listFlavorsWithSDK(ctx, d.client, data)
	if err != nil {
		resp.Diagnostics.AddError("Flavors list error", fmt.Sprintf("Failed to list flavors using SDK: %s", err))
		data.Flavors = []FlavorModel{}
	} else {
		data.Flavors = fls
	}

	// Save state
	if diag := resp.State.Set(ctx, &data); diag.HasError() {
		resp.Diagnostics.AddError("Failed to set state", fmt.Sprintf("Could not set flavor state: %v", diag))
	}

	tflog.Trace(ctx, "Read zillaforge_flavors data source")
}

// func listFlavorsWithReflection() removed - using typed SDK integration

func listFlavorsWithSDK(ctx context.Context, projectClient *cloudsdk.ProjectClient, filters FlavorDataSourceModel) ([]FlavorModel, error) {
	if projectClient == nil {
		return nil, fmt.Errorf("no project client available")
	}
	vpsClient := projectClient.VPS()
	opts := &flavorsmodels.ListFlavorsOptions{}
	if !filters.Name.IsNull() && filters.Name.ValueString() != "" {
		opts.Name = filters.Name.ValueString()
	}
	flavorList, err := vpsClient.Flavors().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("sdk Flavor List() error: %w", err)
	}

	results := []FlavorModel{}
	for _, f := range flavorList {
		// Exact name match
		if !filters.Name.IsNull() && f.Name != filters.Name.ValueString() {
			continue
		}
		// min vcpus
		if !filters.VCPUs.IsNull() {
			if int64(f.VCPU) < filters.VCPUs.ValueInt64() {
				continue
			}
		}
		// min memory - SDK returns GiB
		if !filters.Memory.IsNull() {
			memoryGB := int64(f.Memory)
			if memoryGB < filters.Memory.ValueInt64() {
				continue
			}
		}

		fm := FlavorModel{
			ID:          types.StringValue(f.ID),
			Name:        types.StringValue(f.Name),
			VCPUs:       types.Int64Value(int64(f.VCPU)),
			Memory:      types.Int64Value(int64(f.Memory)),
			Disk:        types.Int64Value(int64(f.Disk)),
			Description: types.StringValue(f.Description),
		}
		results = append(results, fm)
	}
	return results, nil
}
