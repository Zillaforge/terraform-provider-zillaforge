// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	sgmodels "github.com/Zillaforge/cloud-sdk/models/vps/securitygroups"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/helper"
	resourcemodels "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/validators"
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
	var plan resourcemodels.SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating security group", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	// Build rule list from ingress and egress
	rules, diags := helper.BuildSecurityGroupRules(ctx, plan)
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
	apiIngressRules, apiEgressRules, diags := helper.MapSDKRulesToTerraform(ctx, securityGroup.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder API rules to match plan order
	plan.IngressRule = helper.ReorderRulesToMatchPlan(ctx, plan.IngressRule, apiIngressRules)
	plan.EgressRule = helper.ReorderRulesToMatchPlan(ctx, plan.EgressRule, apiEgressRules)

	tflog.Debug(ctx, "Created security group", map[string]interface{}{
		"id":   securityGroup.ID,
		"name": securityGroup.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourcemodels.SecurityGroupResourceModel
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
	apiIngressRules, apiEgressRules, diags := helper.MapSDKRulesToTerraform(ctx, securityGroup.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder API rules to match current state order to prevent phantom changes
	state.IngressRule = helper.ReorderRulesToMatchPlan(ctx, state.IngressRule, apiIngressRules)
	state.EgressRule = helper.ReorderRulesToMatchPlan(ctx, state.EgressRule, apiEgressRules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecurityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourcemodels.SecurityGroupResourceModel
	var state resourcemodels.SecurityGroupResourceModel

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
			Description: plan.Description.ValueStringPointer(),
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
	rules, diags := helper.BuildSecurityGroupRules(ctx, plan)
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
	apiIngressRules, apiEgressRules, diags := helper.MapSDKRulesToTerraform(ctx, updatedGroup.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder API rules to match plan order if possible
	plan.IngressRule = helper.ReorderRulesToMatchPlan(ctx, plan.IngressRule, apiIngressRules)
	plan.EgressRule = helper.ReorderRulesToMatchPlan(ctx, plan.EgressRule, apiEgressRules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourcemodels.SecurityGroupResourceModel
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
	var state resourcemodels.SecurityGroupResourceModel
	state.ID = types.StringValue(securityGroup.ID)
	state.Name = types.StringValue(securityGroup.Name)
	if securityGroup.Description != "" {
		state.Description = types.StringValue(securityGroup.Description)
	} else {
		state.Description = types.StringNull()
	}

	// Map rules from API
	apiIngressRules, apiEgressRules, diags := helper.MapSDKRulesToTerraform(ctx, securityGroup.Rules)
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
