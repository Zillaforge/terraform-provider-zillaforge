// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package model

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
