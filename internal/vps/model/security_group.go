// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package model

import "github.com/hashicorp/terraform-plugin-framework/types"

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
