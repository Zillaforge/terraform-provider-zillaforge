// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sgmodels "github.com/Zillaforge/cloud-sdk/models/vps/securitygroups"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
