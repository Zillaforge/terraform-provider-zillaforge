// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"context"
	"fmt"
	"sort"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	sgmodels "github.com/Zillaforge/cloud-sdk/models/vps/securitygroups"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// SecurityGroupsDataSourceModel describes the data source data model.
type SecurityGroupsDataSourceModel struct {
	ID             types.String             `tfsdk:"id"`
	Name           types.String             `tfsdk:"name"`
	SecurityGroups []SecurityGroupDataModel `tfsdk:"security_groups"`
}

// SecurityGroupDataModel represents a single security group in the results.
type SecurityGroupDataModel struct {
	ID          types.String        `tfsdk:"id"`
	Name        types.String        `tfsdk:"name"`
	Description types.String        `tfsdk:"description"`
	IngressRule []SecurityRuleModel `tfsdk:"ingress_rule"`
	EgressRule  []SecurityRuleModel `tfsdk:"egress_rule"`
}

// SecurityRuleModel represents a firewall rule (same as resource model).
type SecurityRuleModel struct {
	Protocol        types.String `tfsdk:"protocol"`
	PortRange       types.String `tfsdk:"port_range"`
	SourceCIDR      types.String `tfsdk:"source_cidr"`
	DestinationCIDR types.String `tfsdk:"destination_cidr"`
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
	var config SecurityGroupsDataSourceModel
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
	var securityGroups []SecurityGroupDataModel

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
		sgModel, diags := mapSDKSecurityGroupToModel(ctx, *sg)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		securityGroups = []SecurityGroupDataModel{sgModel}
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
				sgModel, diags := mapSDKSecurityGroupToModel(ctx, *sg.SecurityGroup)
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
			sgModel, diags := mapSDKSecurityGroupToModel(ctx, *sg.SecurityGroup)
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

	// Set state
	config.SecurityGroups = securityGroups

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// mapSDKSecurityGroupToModel converts an SDK security group to the data source model.
//
//nolint:unparam // ctx parameter kept for consistency with other mapper functions
func mapSDKSecurityGroupToModel(ctx context.Context, sg sgmodels.SecurityGroup) (SecurityGroupDataModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := SecurityGroupDataModel{
		ID:          types.StringValue(sg.ID),
		Name:        types.StringValue(sg.Name),
		Description: types.StringValue(sg.Description),
	}

	// Map rules - initialize as empty slices to ensure they're never nil
	type tmpRule struct {
		model SecurityRuleModel
		min   int
		max   int
	}

	ingressTmp := make([]tmpRule, 0)
	egressTmp := make([]tmpRule, 0)

	for _, sdkRule := range sg.Rules {
		rule := SecurityRuleModel{
			Protocol:  types.StringValue(string(sdkRule.Protocol)),
			PortRange: types.StringValue(formatPortRange(sdkRule.PortMin, sdkRule.PortMax)),
		}

		if sdkRule.Direction == sgmodels.DirectionIngress {
			rule.SourceCIDR = types.StringValue(sdkRule.RemoteCIDR)
			rule.DestinationCIDR = types.StringNull()
			ingressTmp = append(ingressTmp, tmpRule{model: rule, min: sdkRule.PortMin, max: sdkRule.PortMax})
		} else {
			rule.SourceCIDR = types.StringNull()
			rule.DestinationCIDR = types.StringValue(sdkRule.RemoteCIDR)
			egressTmp = append(egressTmp, tmpRule{model: rule, min: sdkRule.PortMin, max: sdkRule.PortMax})
		}
	}

	// Sort rules deterministically: by port min, then port max, then protocol, then cidr
	sort.SliceStable(ingressTmp, func(i, j int) bool {
		if ingressTmp[i].min != ingressTmp[j].min {
			return ingressTmp[i].min < ingressTmp[j].min
		}
		if ingressTmp[i].max != ingressTmp[j].max {
			return ingressTmp[i].max < ingressTmp[j].max
		}
		if ingressTmp[i].model.Protocol.ValueString() != ingressTmp[j].model.Protocol.ValueString() {
			return ingressTmp[i].model.Protocol.ValueString() < ingressTmp[j].model.Protocol.ValueString()
		}
		return ingressTmp[i].model.SourceCIDR.ValueString() < ingressTmp[j].model.SourceCIDR.ValueString()
	})

	sort.SliceStable(egressTmp, func(i, j int) bool {
		if egressTmp[i].min != egressTmp[j].min {
			return egressTmp[i].min < egressTmp[j].min
		}
		if egressTmp[i].max != egressTmp[j].max {
			return egressTmp[i].max < egressTmp[j].max
		}
		if egressTmp[i].model.Protocol.ValueString() != egressTmp[j].model.Protocol.ValueString() {
			return egressTmp[i].model.Protocol.ValueString() < egressTmp[j].model.Protocol.ValueString()
		}
		return egressTmp[i].model.DestinationCIDR.ValueString() < egressTmp[j].model.DestinationCIDR.ValueString()
	})

	ingressRules := make([]SecurityRuleModel, 0, len(ingressTmp))
	for _, t := range ingressTmp {
		ingressRules = append(ingressRules, t.model)
	}

	egressRules := make([]SecurityRuleModel, 0, len(egressTmp))
	for _, t := range egressTmp {
		egressRules = append(egressRules, t.model)
	}

	model.IngressRule = ingressRules
	model.EgressRule = egressRules

	return model, diags
}

// formatPortRange converts min/max port integers to string format.
func formatPortRange(portMin, portMax int) string {
	if portMin == 0 && portMax == 0 {
		return "all"
	}
	if portMin == 1 && portMax == 65535 {
		return "all"
	}
	if portMin == portMax {
		return fmt.Sprintf("%d", portMin)
	}
	return fmt.Sprintf("%d-%d", portMin, portMax)
}
