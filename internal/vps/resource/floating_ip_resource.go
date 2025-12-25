// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/helper"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &FloatingIPResource{}
	_ resource.ResourceWithImportState = &FloatingIPResource{}
)

// NewFloatingIPResource creates a new instance of the floating IP resource.
func NewFloatingIPResource() resource.Resource {
	return &FloatingIPResource{}
}

// FloatingIPResource defines the floating IP resource implementation.
type FloatingIPResource struct {
	client *cloudsdk.ProjectClient
}

func (r *FloatingIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_floating_ip"
}

func (r *FloatingIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages floating IP addresses in ZillaForge. Allocates, updates, and releases public IPv4 addresses from a shared pool. Floating IPs can be associated with VPS instances for public internet access (association management is out of scope for this resource).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the floating IP (UUID format). Assigned by the API upon allocation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the floating IP. Optional but recommended for identification in large deployments. Can be updated in-place without releasing the IP address.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description providing context about the floating IP's purpose or usage. Can be updated in-place without releasing the IP address.",
				Optional:            true,
			},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "The allocated IPv4 address in dotted-decimal notation (e.g., 203.0.113.42). Assigned automatically from the pool during creation and cannot be changed.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Current operational status of the floating IP. Possible values: `ACTIVE` (ready for use), `DOWN` (unreachable), `PENDING` (allocation in progress), `REJECTED` (allocation failed). All status values are informational only and do not block Terraform operations.",
				Computed:            true,
			},
			"device_id": schema.StringAttribute{
				MarkdownDescription: "ID of the VPS instance this floating IP is associated with. Null when unassociated. This field is read-only; association management is out of scope for this resource.",
				Computed:            true,
			},
		},
	}
}

func (r *FloatingIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		r.client = projectClient
	}
}

func (r *FloatingIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating floating IP", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	// Build create request using model helper
	createReq := helper.BuildCreateRequest(&plan)

	// Call API
	vpsClient := r.client.VPS()
	floatingIP, err := vpsClient.FloatingIPs().Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Create Error",
			fmt.Sprintf("Unable to create floating IP: %s", err),
		)
		return
	}

	// Map response to state using model helper
	var state model.FloatingIPResourceModel
	helper.MapFloatingIPToResourceModel(ctx, floatingIP, &state)

	tflog.Debug(ctx, "Created floating IP", map[string]interface{}{
		"id":         state.ID.ValueString(),
		"ip_address": state.IPAddress.ValueString(),
		"status":     state.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FloatingIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading floating IP", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Call API
	vpsClient := r.client.VPS()
	floatingIP, err := vpsClient.FloatingIPs().Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Read Error",
			fmt.Sprintf("Unable to read floating IP %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	// Map response to state using model helper
	helper.MapFloatingIPToResourceModel(ctx, floatingIP, &state)

	tflog.Debug(ctx, "Read floating IP", map[string]interface{}{
		"id":         state.ID.ValueString(),
		"ip_address": state.IPAddress.ValueString(),
		"status":     state.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FloatingIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating floating IP", map[string]interface{}{
		"id": plan.ID.ValueString(),
	})

	// Build update request using model helper
	updateReq := helper.BuildUpdateRequest(&plan)

	// Call API
	vpsClient := r.client.VPS()
	floatingIP, err := vpsClient.FloatingIPs().Update(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update Error",
			fmt.Sprintf("Unable to update floating IP %s: %s", plan.ID.ValueString(), err),
		)
		return
	}

	// Map response to state using model helper
	var state model.FloatingIPResourceModel
	helper.MapFloatingIPToResourceModel(ctx, floatingIP, &state)

	tflog.Debug(ctx, "Updated floating IP", map[string]interface{}{
		"id":   state.ID.ValueString(),
		"name": state.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FloatingIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model.FloatingIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting floating IP", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Call API
	vpsClient := r.client.VPS()
	err := vpsClient.FloatingIPs().Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete Error",
			fmt.Sprintf("Unable to delete floating IP %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	tflog.Debug(ctx, "Deleted floating IP", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *FloatingIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the ID provided by the user as the floating IP ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, "Importing floating IP", map[string]interface{}{
		"id": req.ID,
	})
}
