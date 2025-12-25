// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	sgmodels "github.com/Zillaforge/cloud-sdk/models/vps/securitygroups"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/helper"
	resourcemodel "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SecurityGroupsDataSource{}

// NewSecurityGroupsDataSource creates a new instance of the security groups data source.
func NewSecurityGroupsDataSource() datasource.DataSource {
	return &SecurityGroupsDataSource{}
}

// SecurityGroupsDataSource defines the data source implementation.
type SecurityGroupsDataSource struct {
	client *cloudsdk.ProjectClient
}

func (d *SecurityGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_groups"
}

func (d *SecurityGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries existing security groups from ZillaForge VPS. Use this data source to retrieve security group details by ID or name, or to list all security groups in your project. Security groups returned include all ingress/egress rules and can be referenced in other resources.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Optional filter to query a specific security group by ID (UUID format). Returns a single security group or errors if not found. **Mutually exclusive with `name`**.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Optional filter to query security groups by exact name (case-sensitive match). Returns a list of matching security groups (typically 0 or 1 since names are unique per project). **Mutually exclusive with `id`**.",
				Optional:            true,
			},
			"security_groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of security group objects matching the filter criteria. Empty list if no matches found (for name filter). Contains all security group attributes including ingress/egress rules.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the security group (UUID format). Use this value to reference the security group in other resources.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Human-readable name for the security group. Unique within the project.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Optional description providing context about the security group's purpose. Empty string if not set.",
							Computed:            true,
						},
						"ingress_rule": schema.ListNestedAttribute{
							MarkdownDescription: "Inbound firewall rules that control traffic TO instances attached to this security group.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"protocol": schema.StringAttribute{
										MarkdownDescription: "Network protocol for this rule. Valid values: `tcp`, `udp`, `icmp`, `any`.",
										Computed:            true,
									},
									"port_range": schema.StringAttribute{
										MarkdownDescription: "Port specification. Formats: single port (`22`), port range (`8000-8100`), or `all` (equivalent to `1-65535`).",
										Computed:            true,
									},
									"source_cidr": schema.StringAttribute{
										MarkdownDescription: "Source CIDR block for allowed inbound traffic. Examples: `0.0.0.0/0` (all IPv4), `192.168.1.0/24` (subnet).",
										Computed:            true,
									},
									"destination_cidr": schema.StringAttribute{
										MarkdownDescription: "Not used for ingress rules. Always null.",
										Computed:            true,
									},
								},
							},
						},
						"egress_rule": schema.ListNestedAttribute{
							MarkdownDescription: "Outbound firewall rules that control traffic FROM instances attached to this security group.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"protocol": schema.StringAttribute{
										MarkdownDescription: "Network protocol for this rule. Valid values: `tcp`, `udp`, `icmp`, `any`.",
										Computed:            true,
									},
									"port_range": schema.StringAttribute{
										MarkdownDescription: "Port specification. Formats: single port (`22`), port range (`8000-8100`), or `all` (equivalent to `1-65535`).",
										Computed:            true,
									},
									"source_cidr": schema.StringAttribute{
										MarkdownDescription: "Not used for egress rules. Always null.",
										Computed:            true,
									},
									"destination_cidr": schema.StringAttribute{
										MarkdownDescription: "Destination CIDR block for allowed outbound traffic. Examples: `0.0.0.0/0` (all IPv4), `10.0.0.0/8` (private network).",
										Computed:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *SecurityGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		d.client = projectClient
	}
}

func (d *SecurityGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config resourcemodel.SecurityGroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate mutually exclusive filters
	if !config.ID.IsNull() && !config.Name.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Filter Attributes",
			"The 'id' and 'name' attributes are mutually exclusive. Please specify only one filter attribute, or omit both to list all security groups.",
		)
		return
	}

	vpsClient := d.client.VPS()
	var securityGroups []resourcemodel.SecurityGroupDataModel

	// Filter by ID
	if !config.ID.IsNull() {
		tflog.Debug(ctx, "Querying security group by ID", map[string]interface{}{
			"id": config.ID.ValueString(),
		})

		securityGroupResource, err := vpsClient.SecurityGroups().Get(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Query Security Group by ID",
				fmt.Sprintf("Unable to read security group with ID '%s': %s", config.ID.ValueString(), err.Error()),
			)
			return
		}

		sg := securityGroupResource.SecurityGroup
		sgModel, diags := helper.MapSDKSecurityGroupToModel(*sg)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		securityGroups = []resourcemodel.SecurityGroupDataModel{sgModel}
	} else if !config.Name.IsNull() {
		// Filter by name
		tflog.Debug(ctx, "Querying security groups by name", map[string]interface{}{
			"name": config.Name.ValueString(),
		})

		// Request detailed information including rules
		allGroups, err := vpsClient.SecurityGroups().List(ctx, &sgmodels.ListSecurityGroupsOptions{
			Detail: true,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to List Security Groups",
				fmt.Sprintf("Unable to list security groups: %s", err.Error()),
			)
			return
		}

		// Client-side name filtering
		for _, sg := range allGroups {
			if sg.SecurityGroup.Name == config.Name.ValueString() {
				sgModel, diags := helper.MapSDKSecurityGroupToModel(*sg.SecurityGroup)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				securityGroups = append(securityGroups, sgModel)
			}
		}

		tflog.Debug(ctx, "Name filter results", map[string]interface{}{
			"name":  config.Name.ValueString(),
			"count": len(securityGroups),
		})
	} else {
		// List all security groups
		tflog.Debug(ctx, "Listing all security groups")

		// Request detailed information including rules
		allGroups, err := vpsClient.SecurityGroups().List(ctx, &sgmodels.ListSecurityGroupsOptions{
			Detail: true,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to List Security Groups",
				fmt.Sprintf("Unable to list security groups: %s", err.Error()),
			)
			return
		}

		for _, sg := range allGroups {
			sgModel, diags := helper.MapSDKSecurityGroupToModel(*sg.SecurityGroup)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			securityGroups = append(securityGroups, sgModel)
		}

		tflog.Debug(ctx, "Listed all security groups", map[string]interface{}{
			"count": len(securityGroups),
		})
	}

	// Ensure deterministic order by ID
	helper.SortSecurityGroupsByID(securityGroups)

	// Set state
	config.SecurityGroups = securityGroups

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
