// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package model

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// KeypairResourceModel describes the Terraform resource data model.
type KeypairResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PublicKey   types.String `tfsdk:"public_key"`
	PrivateKey  types.String `tfsdk:"private_key"` // Sensitive
	Fingerprint types.String `tfsdk:"fingerprint"`
}

// KeypairDataSourceModel describes the data source config and filters.
type KeypairDataSourceModel struct {
	ID       types.String   `tfsdk:"id"`       // Optional filter
	Name     types.String   `tfsdk:"name"`     // Optional filter
	Keypairs []KeypairModel `tfsdk:"keypairs"` // Computed results
}

// KeypairModel represents a single keypair in the results list.
type KeypairModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}
