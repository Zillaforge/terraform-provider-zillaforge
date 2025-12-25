// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/helper"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &KeypairDataSource{}

// NewKeypairDataSource creates a new instance of the keypair data source.
func NewKeypairDataSource() datasource.DataSource {
	return &KeypairDataSource{}
}

// KeypairDataSource defines the keypair data source implementation.
type KeypairDataSource struct {
	client *cloudsdk.ProjectClient
}

func (d *KeypairDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keypairs"
}

func (d *KeypairDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query available SSH keypairs in ZillaForge VPS service. Supports individual lookup by ID or name, and listing all keypairs when no filters are specified.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Filter by specific keypair ID. Mutually exclusive with `name` filter. Returns single keypair if found, error if not found.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter by exact keypair name (case-sensitive). Mutually exclusive with `id` filter. Returns all keypairs matching the name.",
				Optional:            true,
			},
			"keypairs": schema.ListNestedAttribute{
				MarkdownDescription: "List of matching keypair objects. Empty list if no matches found (for name filter) or error (for id filter).",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the keypair (UUID format).",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Human-readable keypair name. Must be unique within the project.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Optional description providing context about the keypair's purpose or usage.",
							Computed:            true,
						},
						"public_key": schema.StringAttribute{
							MarkdownDescription: "SSH public key in OpenSSH format (e.g., ssh-rsa, ecdsa-sha2-nistp256, ssh-ed25519).",
							Computed:            true,
						},
						"fingerprint": schema.StringAttribute{
							MarkdownDescription: "Cryptographic fingerprint of the public key (SHA256 or MD5 hash).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *KeypairDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	// Provider Configure sets the SDK project client as DataSourceData
	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		d.client = projectClient
	} else {
		// Not the expected type - leave as nil to return empty result safely
		d.client = nil
	}
}

func (d *KeypairDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data model.KeypairDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// T035: Validate mutual exclusivity of id and name filters
	hasID := !data.ID.IsNull() && data.ID.ValueString() != ""
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""

	if hasID && hasName {
		resp.Diagnostics.AddError(
			"Invalid Filter Combination",
			"Cannot specify both 'id' and 'name' filters. Please use only one filter at a time.",
		)
		return
	}

	// If client not configured, return empty list (but not error) to avoid failing plan
	if d.client == nil {
		data.Keypairs = []model.KeypairModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	vpsClient := d.client.VPS()

	// T036: ID filter mode - use Get() for single keypair lookup
	if hasID {
		keypair, err := vpsClient.Keypairs().Get(ctx, data.ID.ValueString())
		if err != nil {
			// T039: Error handling for not-found by ID scenario
			resp.Diagnostics.AddError(
				"Keypair Not Found",
				fmt.Sprintf("Keypair with ID '%s' not found: %s", data.ID.ValueString(), err),
			)
			return
		}

		// Convert to model and set as single-item list
		data.Keypairs = []model.KeypairModel{helper.KeypairToModel(*keypair)}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		tflog.Trace(ctx, "Read zillaforge_keypairs data source by ID")
		return
	}

	// T037: Name filter or list-all mode - use List()
	keypairs, err := helper.ListKeypairsWithSDK(ctx, d.client, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Keypairs List Error",
			fmt.Sprintf("Failed to list keypairs using SDK: %s", err),
		)
		data.Keypairs = []model.KeypairModel{}
	} else {
		data.Keypairs = keypairs
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, "Read zillaforge_keypairs data source")
}
