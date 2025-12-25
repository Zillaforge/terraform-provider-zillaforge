// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	keypairsmodels "github.com/Zillaforge/cloud-sdk/models/vps/keypairs"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &KeypairResource{}
	_ resource.ResourceWithImportState = &KeypairResource{}
)

// NewKeypairResource creates a new instance of the keypair resource.
func NewKeypairResource() resource.Resource {
	return &KeypairResource{}
}

// KeypairResource defines the keypair resource implementation.
type KeypairResource struct {
	client *cloudsdk.ProjectClient
}

func (r *KeypairResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keypair"
}

func (r *KeypairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages SSH keypairs for VPS instance access in ZillaForge. Supports both user-provided public keys and system-generated keypairs. Note: keypair name and public key are immutable after creation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the keypair (UUID format). Assigned by the API upon creation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the keypair. Must be unique within the project. **Immutable** - changing this value forces resource replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description providing context about the keypair's purpose or usage. This is the only updatable attribute.",
				Optional:            true,
				Computed:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "SSH public key in OpenSSH format (ssh-rsa, ecdsa-sha2-*, ssh-ed25519). If omitted, the system generates a keypair automatically and returns both public and private keys. **Immutable** - changing this value forces resource replacement.",
				Optional:            true,
				Computed:            true, // Computed if user doesn't provide (system-generated)
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "Private key for SSH authentication. **Only available for system-generated keypairs** (when `public_key` is not provided). The private key is returned only once during creation and marked as sensitive to prevent exposure in logs or console output. For user-provided public keys, this field remains null.",
				Computed:            true,
				Sensitive:           true, // Prevents exposure in logs/plan output
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "Cryptographic fingerprint of the public key (SHA256 or MD5 hash format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *KeypairResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		r.client = projectClient
	}
}

func (r *KeypairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model.KeypairResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating keypair", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	// Build create request
	createReq := &keypairsmodels.KeypairCreateRequest{
		Name: plan.Name.ValueString(),
	}

	// Add optional fields
	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.PublicKey.IsNull() {
		createReq.PublicKey = plan.PublicKey.ValueString()
	}

	// Call API
	vpsClient := r.client.VPS()
	keypair, err := vpsClient.Keypairs().Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Create Error",
			fmt.Sprintf("Unable to create keypair: %s", err),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(keypair.ID)
	plan.Name = types.StringValue(keypair.Name)
	if keypair.Description != "" {
		plan.Description = types.StringValue(keypair.Description)
	} else {
		plan.Description = types.StringNull()
	}
	plan.PublicKey = types.StringValue(keypair.PublicKey)
	plan.Fingerprint = types.StringValue(keypair.Fingerprint)

	// Private key only available for system-generated keypairs
	if keypair.PrivateKey != "" {
		plan.PrivateKey = types.StringValue(keypair.PrivateKey)
	} else {
		plan.PrivateKey = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Info(ctx, "Created keypair", map[string]interface{}{
		"id": keypair.ID,
	})
}

func (r *KeypairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model.KeypairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading keypair", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Call API
	vpsClient := r.client.VPS()
	keypair, err := vpsClient.Keypairs().Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Read Error",
			fmt.Sprintf("Unable to read keypair: %s", err),
		)
		return
	}

	// Update state
	state.Name = types.StringValue(keypair.Name)
	if keypair.Description != "" {
		state.Description = types.StringValue(keypair.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.PublicKey = types.StringValue(keypair.PublicKey)
	state.Fingerprint = types.StringValue(keypair.Fingerprint)
	// Note: PrivateKey is never returned by Get (security), so preserve existing state

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KeypairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model.KeypairResourceModel
	var state model.KeypairResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating keypair", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Build update request (only description is updatable)
	updateReq := &keypairsmodels.KeypairUpdateRequest{}
	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueString()
	}

	// Call API
	vpsClient := r.client.VPS()
	keypair, err := vpsClient.Keypairs().Update(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update Error",
			fmt.Sprintf("Unable to update keypair: %s", err),
		)
		return
	}

	// Update state
	state.Name = types.StringValue(keypair.Name)
	if keypair.Description != "" {
		state.Description = types.StringValue(keypair.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.PublicKey = types.StringValue(keypair.PublicKey)
	state.Fingerprint = types.StringValue(keypair.Fingerprint)
	// Preserve PrivateKey from state (not returned by Update)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Info(ctx, "Updated keypair", map[string]interface{}{
		"id": keypair.ID,
	})
}

func (r *KeypairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model.KeypairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Warn(ctx, "Deleting keypair that may be in use by VPS instances", map[string]interface{}{
		"keypair_id":   state.ID.ValueString(),
		"keypair_name": state.Name.ValueString(),
	})

	// Call API
	vpsClient := r.client.VPS()
	err := vpsClient.Keypairs().Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete Error",
			fmt.Sprintf("Unable to delete keypair: %s", err),
		)
		return
	}

	tflog.Info(ctx, "Deleted keypair", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *KeypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by keypair ID
	keypairID := req.ID

	tflog.Debug(ctx, "Importing keypair", map[string]interface{}{
		"id": keypairID,
	})

	// Fetch keypair from API
	vpsClient := r.client.VPS()
	keypair, err := vpsClient.Keypairs().Get(ctx, keypairID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read keypair %s: %s", keypairID, err),
		)
		return
	}

	// Build state
	var state model.KeypairResourceModel
	state.ID = types.StringValue(keypair.ID)
	state.Name = types.StringValue(keypair.Name)
	if keypair.Description != "" {
		state.Description = types.StringValue(keypair.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.PublicKey = types.StringValue(keypair.PublicKey)
	state.Fingerprint = types.StringValue(keypair.Fingerprint)
	// PrivateKey is never available after creation (security), so set to null
	state.PrivateKey = types.StringNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Info(ctx, "Imported keypair", map[string]interface{}{
		"id": keypair.ID,
	})
}
