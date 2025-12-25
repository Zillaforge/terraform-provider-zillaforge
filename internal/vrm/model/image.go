// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package model

import "github.com/hashicorp/terraform-plugin-framework/types"

// ImagesDataSourceModel describes the data source config and filters.
type ImagesDataSourceModel struct {
	Repository types.String `tfsdk:"repository"`  // Optional filter
	Tag        types.String `tfsdk:"tag"`         // Optional filter (mutually exclusive with tag_pattern)
	TagPattern types.String `tfsdk:"tag_pattern"` // Optional filter (mutually exclusive with tag)
	Images     []ImageModel `tfsdk:"images"`      // Computed results
}

// ImageModel represents a single image (tag) in the results list.
type ImageModel struct {
	ID              types.String `tfsdk:"id"`
	RepositoryName  types.String `tfsdk:"repository_name"`
	TagName         types.String `tfsdk:"tag_name"`
	Size            types.Int64  `tfsdk:"size"`
	OperatingSystem types.String `tfsdk:"operating_system"`
	Description     types.String `tfsdk:"description"`
	Type            types.String `tfsdk:"type"`
	Status          types.String `tfsdk:"status"`
}
