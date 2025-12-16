// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	sgmodels "github.com/Zillaforge/cloud-sdk/models/vps/securitygroups"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/validators"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SecurityGroupResource{}
var _ resource.ResourceWithImportState = &SecurityGroupResource{}

// NewSecurityGroupResource creates a new instance of the security group resource.
func NewSecurityGroupResource() resource.Resource {
	return &SecurityGroupResource{}
}

// SecurityGroupResource defines the security group resource implementation.
type SecurityGroupResource struct {
	client *cloudsdk.ProjectClient
}

// SecurityGroupResourceModel describes the resource data model.
type SecurityGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IngressRule types.List   `tfsdk:"ingress_rule"`
	EgressRule  types.List   `tfsdk:"egress_rule"`
}

// SecurityRuleModel represents a firewall rule in the schema.
type SecurityRuleModel struct {
	Protocol        types.String `tfsdk:"protocol"`
	PortRange       types.String `tfsdk:"port_range"`
	SourceCIDR      types.String `tfsdk:"source_cidr"`      // For ingress only
	DestinationCIDR types.String `tfsdk:"destination_cidr"` // For egress only
}

func (r *SecurityGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (r *SecurityGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages security groups for VPS instances in ZillaForge. Security groups act as stateful virtual firewalls that control inbound and outbound traffic using protocol, port, and CIDR-based rules. Multiple security groups can be attached to a single instance, with rules evaluated using union logic (most permissive wins).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the security group (UUID format). Assigned by the API upon creation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the security group. Must be unique within the project. **Immutable** - changing this value forces resource replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description providing context about the security group's purpose. This attribute can be updated in-place without recreating the resource.",
				Optional:            true,
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"ingress_rule": schema.ListNestedBlock{
				MarkdownDescription: "Inbound firewall rules that control traffic TO instances attached to this security group. Rules specify allowed source traffic by protocol, port range, and source CIDR block. Empty list denies all inbound traffic (secure by default).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Network protocol for this rule. Valid values: `tcp`, `udp`, `icmp`, `any`. Case-insensitive.",
							Required:            true,
							Validators: []validator.String{
								validators.Protocol(),
							},
						},
						"port_range": schema.StringAttribute{
							MarkdownDescription: "Port specification. Valid formats: single port (`22`), port range (`8000-8100`), or `all` (equivalent to `1-65535` for TCP/UDP). For ICMP protocol, must be `all`.",
							Required:            true,
							Validators: []validator.String{
								validators.PortRange(),
							},
						},
						"source_cidr": schema.StringAttribute{
							MarkdownDescription: "Source CIDR block for allowed inbound traffic. Examples: `0.0.0.0/0` (all IPv4), `192.168.1.0/24` (subnet), `::/0` (all IPv6). Both IPv4 and IPv6 are supported.",
							Required:            true,
							Validators: []validator.String{
								validators.CIDR(),
							},
						},
						"destination_cidr": schema.StringAttribute{
							MarkdownDescription: "Not used for ingress rules. Must be null or empty.",
							Computed:            true,
						},
					},
				},
			},
			"egress_rule": schema.ListNestedBlock{
				MarkdownDescription: "Outbound firewall rules that control traffic FROM instances attached to this security group. Rules specify allowed destination traffic by protocol, port range, and destination CIDR block. Empty list denies all outbound traffic.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Network protocol for this rule. Valid values: `tcp`, `udp`, `icmp`, `any`. Case-insensitive.",
							Required:            true,
							Validators: []validator.String{
								validators.Protocol(),
							},
						},
						"port_range": schema.StringAttribute{
							MarkdownDescription: "Port specification. Valid formats: single port (`22`), port range (`8000-8100`), or `all` (equivalent to `1-65535` for TCP/UDP). For ICMP protocol, must be `all`.",
							Required:            true,
							Validators: []validator.String{
								validators.PortRange(),
							},
						},
						"source_cidr": schema.StringAttribute{
							MarkdownDescription: "Not used for egress rules. Must be null or empty.",
							Computed:            true,
						},
						"destination_cidr": schema.StringAttribute{
							MarkdownDescription: "Destination CIDR block for allowed outbound traffic. Examples: `0.0.0.0/0` (all IPv4), `10.0.0.0/8` (private network), `::/0` (all IPv6). Both IPv4 and IPv6 are supported.",
							Required:            true,
							Validators: []validator.String{
								validators.CIDR(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *SecurityGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		r.client = projectClient
	}
}

func (r *SecurityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating security group", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	// Build rule list from ingress and egress
	rules, diags := buildSecurityGroupRules(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create request
	createReq := sgmodels.SecurityGroupCreateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Rules:       rules,
	}

	vpsClient := r.client.VPS()
	securityGroupResource, err := vpsClient.SecurityGroups().Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Create Security Group",
			fmt.Sprintf("Unable to create security group '%s': %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}

	// Extract the security group from the resource wrapper
	securityGroup := securityGroupResource.SecurityGroup

	// Map response to state
	plan.ID = types.StringValue(securityGroup.ID)
	if securityGroup.Description != "" {
		plan.Description = types.StringValue(securityGroup.Description)
	} else {
		plan.Description = types.StringValue("")
	}

	// Map rules from API response back to state
	apiIngressRules, apiEgressRules, diags := mapSDKRulesToTerraform(ctx, securityGroup.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder API rules to match plan order
	plan.IngressRule = reorderRulesToMatchPlan(ctx, plan.IngressRule, apiIngressRules)
	plan.EgressRule = reorderRulesToMatchPlan(ctx, plan.EgressRule, apiEgressRules)

	tflog.Debug(ctx, "Created security group", map[string]interface{}{
		"id":   securityGroup.ID,
		"name": securityGroup.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpsClient := r.client.VPS()
	securityGroupResource, err := vpsClient.SecurityGroups().Get(ctx, state.ID.ValueString())
	if err != nil {
		// Check for 404
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			tflog.Warn(ctx, "Security group not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Failed to Read Security Group",
			fmt.Sprintf("Unable to read security group '%s': %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Extract the security group from the resource wrapper
	securityGroup := securityGroupResource.SecurityGroup

	// Update state from API response
	state.Name = types.StringValue(securityGroup.Name)
	if securityGroup.Description != "" {
		state.Description = types.StringValue(securityGroup.Description)
	} else {
		state.Description = types.StringNull()
	}

	// Map rules from API response
	apiIngressRules, apiEgressRules, diags := mapSDKRulesToTerraform(ctx, securityGroup.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder API rules to match current state order to prevent phantom changes
	state.IngressRule = reorderRulesToMatchPlan(ctx, state.IngressRule, apiIngressRules)
	state.EgressRule = reorderRulesToMatchPlan(ctx, state.EgressRule, apiEgressRules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecurityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SecurityGroupResourceModel
	var state SecurityGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating security group", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	vpsClient := r.client.VPS()

	// Update description if changed
	if !plan.Description.Equal(state.Description) {
		updateReq := sgmodels.SecurityGroupUpdateRequest{
			Description: stringPtr(plan.Description.ValueString()),
		}

		_, err := vpsClient.SecurityGroups().Update(ctx, state.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Update Security Group Description",
				fmt.Sprintf("Unable to update security group description: %s", err.Error()),
			)
			return
		}
	}

	// Full rule replacement strategy (delete all, add new)
	// This is simpler than diff-based updates and ensures consistency
	securityGroupResource, err := vpsClient.SecurityGroups().Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Read Security Group for Update",
			fmt.Sprintf("Unable to read current security group state: %s", err.Error()),
		)
		return
	}

	securityGroup := securityGroupResource.SecurityGroup

	// Delete all existing rules using the Rules() client
	rulesClient := securityGroupResource.Rules()
	for _, rule := range securityGroup.Rules {
		err := rulesClient.Delete(ctx, rule.ID)
		if err != nil {
			tflog.Warn(ctx, "Failed to delete rule during update", map[string]interface{}{
				"rule_id": rule.ID,
				"error":   err.Error(),
			})
			// Continue with other deletions
		}
	}

	// Add new rules
	rules, diags := buildSecurityGroupRules(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, rule := range rules {
		_, err := rulesClient.Create(ctx, rule)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Create Security Group Rule",
				fmt.Sprintf("Unable to create security group rule: %s", err.Error()),
			)
			return
		}
	}

	// Read back final state
	updatedGroupResource, err := vpsClient.SecurityGroups().Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Read Updated Security Group",
			fmt.Sprintf("Security group updated but unable to read final state: %s", err.Error()),
		)
		return
	}

	updatedGroup := updatedGroupResource.SecurityGroup

	// Map updated state
	plan.ID = types.StringValue(updatedGroup.ID)
	plan.Name = types.StringValue(updatedGroup.Name)
	if updatedGroup.Description != "" {
		plan.Description = types.StringValue(updatedGroup.Description)
	} else {
		plan.Description = types.StringValue("")
	}

	// We need to map the rules from the API response but maintain the order from the plan
	// Get rules from API
	apiIngressRules, apiEgressRules, diags := mapSDKRulesToTerraform(ctx, updatedGroup.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder API rules to match plan order if possible
	plan.IngressRule = reorderRulesToMatchPlan(ctx, plan.IngressRule, apiIngressRules)
	plan.EgressRule = reorderRulesToMatchPlan(ctx, plan.EgressRule, apiEgressRules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting security group", map[string]interface{}{
		"id":   state.ID.ValueString(),
		"name": state.Name.ValueString(),
	})

	vpsClient := r.client.VPS()
	err := vpsClient.SecurityGroups().Delete(ctx, state.ID.ValueString())
	if err != nil {
		// Check for 404 (already deleted)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			tflog.Warn(ctx, "Security group already deleted", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			return
		}

		// Check for 409 (in use by instances)
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "in use") {
			// Parse security group ID from error message if available
			sgID := state.ID.ValueString()
			errorMsg := err.Error()

			// Try to extract SG ID from error: "(neutron)Security Group {id} in use."
			sgPattern := regexp.MustCompile(`Security Group ([a-f0-9\-]+) in use`)
			if matches := sgPattern.FindStringSubmatch(errorMsg); len(matches) > 1 {
				sgID = matches[1]
			}

			resp.Diagnostics.AddError(
				"Security Group In Use",
				fmt.Sprintf("Cannot delete security group '%s' (ID: %s): it is currently in use by one or more instances.\n\n"+
					"Please detach the security group from all instances before deletion.\n\n"+
					"To find instances using this security group, check the ZillaForge console or use the CLI:\n"+
					"  zillaforge instances list --security-group %s",
					state.Name.ValueString(), sgID, sgID),
			)
			return
		}

		resp.Diagnostics.AddError(
			"Failed to Delete Security Group",
			fmt.Sprintf("Unable to delete security group '%s': %s", state.Name.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted security group", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

// ImportState imports a security group by its ID.
// T059-T062: Import implementation with UUID validation and error handling.
func (r *SecurityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// T060: Validate import ID is valid UUID format
	importID := req.ID
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(importID) {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Import ID must be a valid UUID format (e.g., '12345678-1234-1234-1234-123456789abc'). Got: '%s'", importID),
		)
		return
	}

	tflog.Debug(ctx, "Importing security group", map[string]interface{}{
		"id": importID,
	})

	// T062: Handle import errors (not found, invalid ID) with clear diagnostics
	vpsClient := r.client.VPS()
	securityGroupResource, err := vpsClient.SecurityGroups().Get(ctx, importID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read security group '%s': %s\n\nVerify the security group exists and you have permission to access it.", importID, err),
		)
		return
	}

	securityGroup := securityGroupResource.SecurityGroup

	// T061: Call Read() logic to populate state after import
	// Build state from API response
	var state SecurityGroupResourceModel
	state.ID = types.StringValue(securityGroup.ID)
	state.Name = types.StringValue(securityGroup.Name)
	if securityGroup.Description != "" {
		state.Description = types.StringValue(securityGroup.Description)
	} else {
		state.Description = types.StringNull()
	}

	// Map rules from API
	apiIngressRules, apiEgressRules, diags := mapSDKRulesToTerraform(ctx, securityGroup.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.IngressRule = apiIngressRules
	state.EgressRule = apiEgressRules

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Info(ctx, "Imported security group", map[string]interface{}{
		"id":   securityGroup.ID,
		"name": securityGroup.Name,
	})
}

// Helper functions

// buildSecurityGroupRules converts Terraform rule models to SDK rule creation requests.
func buildSecurityGroupRules(ctx context.Context, model SecurityGroupResourceModel) ([]sgmodels.SecurityGroupRuleCreateRequest, diag.Diagnostics) {
	var rules []sgmodels.SecurityGroupRuleCreateRequest
	var diags diag.Diagnostics

	// Process ingress rules
	if !model.IngressRule.IsNull() && !model.IngressRule.IsUnknown() {
		var ingressRules []SecurityRuleModel
		diags.Append(model.IngressRule.ElementsAs(ctx, &ingressRules, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for i, rule := range ingressRules {
			portMin, portMax, err := parsePortRange(rule.PortRange.ValueString())
			if err != nil {
				diags.AddAttributeError(
					path.Root("ingress_rule").AtListIndex(i).AtName("port_range"),
					"Invalid Port Range",
					fmt.Sprintf("Failed to parse port range: %s", err.Error()),
				)
				continue
			}

			sdkRule := sgmodels.SecurityGroupRuleCreateRequest{
				Direction:  sgmodels.DirectionIngress,
				Protocol:   sgmodels.Protocol(strings.ToLower(rule.Protocol.ValueString())),
				RemoteCIDR: rule.SourceCIDR.ValueString(),
			}

			// Only set ports for TCP/UDP (not ICMP/any)
			protocol := strings.ToLower(rule.Protocol.ValueString())
			if protocol == "tcp" || protocol == "udp" {
				sdkRule.PortMin = portMin
				sdkRule.PortMax = portMax
			}

			rules = append(rules, sdkRule)
		}
	}

	// Process egress rules
	if !model.EgressRule.IsNull() && !model.EgressRule.IsUnknown() {
		var egressRules []SecurityRuleModel
		diags.Append(model.EgressRule.ElementsAs(ctx, &egressRules, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for i, rule := range egressRules {
			portMin, portMax, err := parsePortRange(rule.PortRange.ValueString())
			if err != nil {
				diags.AddAttributeError(
					path.Root("egress_rule").AtListIndex(i).AtName("port_range"),
					"Invalid Port Range",
					fmt.Sprintf("Failed to parse port range: %s", err.Error()),
				)
				continue
			}

			sdkRule := sgmodels.SecurityGroupRuleCreateRequest{
				Direction:  sgmodels.DirectionEgress,
				Protocol:   sgmodels.Protocol(strings.ToLower(rule.Protocol.ValueString())),
				RemoteCIDR: rule.DestinationCIDR.ValueString(),
			}

			// Only set ports for TCP/UDP (not ICMP/any)
			protocol := strings.ToLower(rule.Protocol.ValueString())
			if protocol == "tcp" || protocol == "udp" {
				sdkRule.PortMin = portMin
				sdkRule.PortMax = portMax
			}

			rules = append(rules, sdkRule)
		}
	}

	if diags.HasError() {
		return nil, diags
	}

	return rules, diags
}

// mapSDKRulesToTerraform converts SDK rules to Terraform models, separating by direction.
func mapSDKRulesToTerraform(ctx context.Context, sdkRules []sgmodels.SecurityGroupRule) (types.List, types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ingressRules []SecurityRuleModel
	var egressRules []SecurityRuleModel

	for _, sdkRule := range sdkRules {
		tfRule := SecurityRuleModel{
			Protocol:  types.StringValue(string(sdkRule.Protocol)),
			PortRange: types.StringValue(formatPortRange(sdkRule.PortMin, sdkRule.PortMax)),
		}

		if sdkRule.Direction == sgmodels.DirectionIngress {
			tfRule.SourceCIDR = types.StringValue(sdkRule.RemoteCIDR)
			tfRule.DestinationCIDR = types.StringNull()
			ingressRules = append(ingressRules, tfRule)
		} else {
			tfRule.SourceCIDR = types.StringNull()
			tfRule.DestinationCIDR = types.StringValue(sdkRule.RemoteCIDR)
			egressRules = append(egressRules, tfRule)
		}
	}

	// Convert to types.List
	ingressList, ingressDiags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"protocol":         types.StringType,
			"port_range":       types.StringType,
			"source_cidr":      types.StringType,
			"destination_cidr": types.StringType,
		},
	}, ingressRules)
	diags.Append(ingressDiags...)

	egressList, egressDiags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"protocol":         types.StringType,
			"port_range":       types.StringType,
			"source_cidr":      types.StringType,
			"destination_cidr": types.StringType,
		},
	}, egressRules)
	diags.Append(egressDiags...)

	return ingressList, egressList, diags
}

// parsePortRange converts port range string to min/max integers.
// Formats: "all" -> (1, 65535), "80" -> (80, 80), "8000-8100" -> (8000, 8100).
func parsePortRange(portRange string) (*int, *int, error) {
	// Handle "all"
	if strings.ToLower(portRange) == "all" {
		minPort, maxPort := 1, 65535
		return &minPort, &maxPort, nil
	}

	// Handle range "start-end"
	if strings.Contains(portRange, "-") {
		parts := strings.Split(portRange, "-")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid port range format: %s", portRange)
		}

		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, nil, fmt.Errorf("invalid port numbers in range: %s", portRange)
		}

		return &start, &end, nil
	}

	// Handle single port
	port, err := strconv.Atoi(portRange)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid port number: %s", portRange)
	}

	return &port, &port, nil
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
		return strconv.Itoa(portMin)
	}
	return fmt.Sprintf("%d-%d", portMin, portMax)
}

// reorderRulesToMatchPlan reorders API rules to match the order in the plan.
// This prevents Terraform from detecting phantom changes due to API reordering.
func reorderRulesToMatchPlan(ctx context.Context, planList types.List, apiList types.List) types.List {
	// If plan is null/unknown or has different length, return API list as-is
	if planList.IsNull() || planList.IsUnknown() || apiList.IsNull() {
		return apiList
	}

	var planRules []SecurityRuleModel
	var apiRules []SecurityRuleModel

	planList.ElementsAs(ctx, &planRules, false)
	apiList.ElementsAs(ctx, &apiRules, false)

	if len(planRules) != len(apiRules) {
		return apiList
	}

	// Create a map of API rules for quick lookup
	// Key: protocol+portRange+cidr (using the non-null CIDR field)
	apiRuleMap := make(map[string]SecurityRuleModel)
	for _, rule := range apiRules {
		var cidr string
		if !rule.SourceCIDR.IsNull() && !rule.SourceCIDR.IsUnknown() {
			cidr = rule.SourceCIDR.ValueString()
		} else if !rule.DestinationCIDR.IsNull() && !rule.DestinationCIDR.IsUnknown() {
			cidr = rule.DestinationCIDR.ValueString()
		}
		key := rule.Protocol.ValueString() + "|" + rule.PortRange.ValueString() + "|" + cidr
		apiRuleMap[key] = rule
	}

	// Reorder API rules to match plan order
	var reorderedRules []SecurityRuleModel
	for _, planRule := range planRules {
		var cidr string
		if !planRule.SourceCIDR.IsNull() && !planRule.SourceCIDR.IsUnknown() {
			cidr = planRule.SourceCIDR.ValueString()
		} else if !planRule.DestinationCIDR.IsNull() && !planRule.DestinationCIDR.IsUnknown() {
			cidr = planRule.DestinationCIDR.ValueString()
		}
		key := planRule.Protocol.ValueString() + "|" + planRule.PortRange.ValueString() + "|" + cidr

		if apiRule, found := apiRuleMap[key]; found {
			// Use the API rule which has all computed fields properly set
			reorderedRules = append(reorderedRules, apiRule)
		} else {
			// If not found, this shouldn't happen in normal cases
			// Fall back to API list without reordering
			return apiList
		}
	}

	// Convert back to types.List
	reorderedList, _ := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"protocol":         types.StringType,
			"port_range":       types.StringType,
			"source_cidr":      types.StringType,
			"destination_cidr": types.StringType,
		},
	}, reorderedRules)

	return reorderedList
}

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}
