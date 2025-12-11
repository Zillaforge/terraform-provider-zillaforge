// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	networksmodels "github.com/Zillaforge/cloud-sdk/models/vps/networks"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &NetworkDataSource{}

func NewNetworkDataSource() datasource.DataSource { return &NetworkDataSource{} }

type NetworkDataSource struct {
	client *cloudsdk.ProjectClient
}

type NetworkDataSourceModel struct {
	Name     types.String   `tfsdk:"name"`
	Status   types.String   `tfsdk:"status"`
	Networks []NetworkModel `tfsdk:"networks"`
}

type NetworkModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	CIDR        types.String `tfsdk:"cidr"`
	Status      types.String `tfsdk:"status"`
	Description types.String `tfsdk:"description"`
}

func (d *NetworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *NetworkDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query available networks in Zillaforge VPS service.",
		Attributes: map[string]schema.Attribute{
			"name":   schema.StringAttribute{MarkdownDescription: "Exact name match", Optional: true},
			"status": schema.StringAttribute{MarkdownDescription: "Exact status match", Optional: true},
			"networks": schema.ListNestedAttribute{MarkdownDescription: "List of matching networks", Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":          schema.StringAttribute{MarkdownDescription: "Network id", Computed: true},
				"name":        schema.StringAttribute{MarkdownDescription: "Network name", Computed: true},
				"cidr":        schema.StringAttribute{MarkdownDescription: "CIDR block", Computed: true},
				"status":      schema.StringAttribute{MarkdownDescription: "Network status", Computed: true},
				"description": schema.StringAttribute{MarkdownDescription: "Optional description", Computed: true},
			}}},
		},
	}
}

func (d *NetworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		d.client = projectClient
	}
}

func (d *NetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		// return empty list
		data.Networks = []NetworkModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	nets, err := listNetworksWithSDK(ctx, d.client, data)
	if err != nil {
		resp.Diagnostics.AddError("Networks list error", fmt.Sprintf("Failed to list networks using SDK: %s", err))
		data.Networks = []NetworkModel{}
	} else {
		data.Networks = nets
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, "Read zillaforge_networks data source")
}

func listNetworksWithSDK(ctx context.Context, projectClient *cloudsdk.ProjectClient, filters NetworkDataSourceModel) ([]NetworkModel, error) {
	if projectClient == nil {
		return nil, fmt.Errorf("no project client available")
	}
	vpsClient := projectClient.VPS()
	opts := &networksmodels.ListNetworksOptions{}
	if !filters.Name.IsNull() {
		opts.Name = filters.Name.ValueString()
	}
	if !filters.Status.IsNull() {
		opts.Status = filters.Status.ValueString()
	}
	networkList, err := vpsClient.Networks().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("sdk Network List() error: %w", err)
	}
	results := []NetworkModel{}
	for _, nr := range networkList {
		// Apply exact name filter: if provided and mismatched -> skip
		if !filters.Name.IsNull() && nr.Network.Name != filters.Name.ValueString() {
			continue
		}
		if !filters.Status.IsNull() && nr.Network.Status != filters.Status.ValueString() {
			continue
		}
		nm := NetworkModel{
			ID:          types.StringValue(nr.Network.ID),
			Name:        types.StringValue(nr.Network.Name),
			CIDR:        types.StringValue(nr.Network.CIDR),
			Status:      types.StringValue(nr.Network.Status),
			Description: types.StringValue(nr.Network.Description),
		}
		results = append(results, nm)
	}
	return results, nil
}
