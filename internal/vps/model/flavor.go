// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package model

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
