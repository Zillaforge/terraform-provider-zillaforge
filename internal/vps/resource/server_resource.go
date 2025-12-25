// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"sort"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	servermodels "github.com/Zillaforge/cloud-sdk/models/vps/servers"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/helper"
	resourcemodels "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/modifiers"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/validators"
)

// ServerResource defines the server resource implementation.
type ServerResource struct {
	client *cloudsdk.ProjectClient
}

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ServerResource{}
	_ resource.ResourceWithImportState = &ServerResource{}
)

// NewServerResource creates a new instance of the server resource.
func NewServerResource() resource.Resource {
	return &ServerResource{}
}

func (r *ServerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *ServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a ZillaForge VPS virtual machine instance. Supports create, read, update (in-place for name, description, network attachments), delete, and import operations.",

		Blocks: map[string]schema.Block{
			"network_attachment": schema.ListNestedBlock{
				MarkdownDescription: "Network interfaces to attach to the server. Each block defines a network connection. At least one network attachment is required, and at most one can be marked as `primary=true`.",
				Validators: []validator.List{
					validators.NetworkAttachmentPrimaryConstraint(),
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the network to attach. Use the `zillaforge_networks` data source to list available networks.",
							Required:            true,
							Validators: []validator.String{
								validators.NetworkIDValidator(),
							},
						},
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "Optional fixed IPv4 address to assign to this network interface. If not specified, an IP address will be automatically assigned via DHCP. Must be a valid IPv4 address within the network's CIDR range.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								// Don't use UseStateForUnknown() here - when network_id changes,
								// ip_address must be recomputed for the new network
								modifiers.IPAddressUnknownOnNetworkChange(),
							},
						},
						"primary": schema.BoolAttribute{
							MarkdownDescription: "Whether this is the primary network interface for the server. At most one network attachment can have `primary=true`. The primary interface is used for default routing.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"security_group_ids": schema.ListAttribute{
							MarkdownDescription: "List of security group IDs to apply to this network interface. Use the `zillaforge_security_groups` data source to list available security groups.",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"timeouts": schema.SingleNestedBlock{
				MarkdownDescription: "Configurable timeouts for create, update, and delete operations.",
				Attributes: map[string]schema.Attribute{
					"create": schema.StringAttribute{
						MarkdownDescription: "Maximum time to wait for server creation to complete. Default is `10m` (10 minutes). Use Go duration syntax (e.g., `15m`, `1h`).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("10m"),
						PlanModifiers: []planmodifier.String{
							modifiers.IgnoreChangeAttributePlanModifierString("timeouts.create"),
						},
					},
					"update": schema.StringAttribute{
						MarkdownDescription: "Maximum time to wait for server update to complete. Default is `10m` (10 minutes). Use Go duration syntax (e.g., `15m`, `1h`).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("10m"),
						PlanModifiers: []planmodifier.String{
							modifiers.IgnoreChangeAttributePlanModifierString("timeouts.update"),
						},
					},
					"delete": schema.StringAttribute{
						MarkdownDescription: "Maximum time to wait for server deletion to complete. Default is `10m` (10 minutes). Use Go duration syntax (e.g., `15m`, `1h`).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("10m"),
						PlanModifiers: []planmodifier.String{
							modifiers.IgnoreChangeAttributePlanModifierString("timeouts.delete"),
						},
					},
				},
			},
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the server instance. Generated by the platform.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the server instance. Must be unique within the project and between 1-255 characters.",
				Required:            true,
			},
			"flavor_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the flavor (instance type) to use for this server. Defines the virtual CPU count, memory, and root disk size. **Changing this attribute is not supported and will be rejected at plan time.** Use the `zillaforge_flavors` data source to list available flavors.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					modifiers.ImmutableAttributePlanModifier("flavor_id"),
				},
				Validators: []validator.String{
					validators.FlavorIDValidator(),
				},
			},
			"image_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the image to use for the server's operating system. **Changing this attribute is not supported and will be rejected at plan time.** Use the `zillaforge_images` data source to list available images.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					modifiers.ImmutableAttributePlanModifier("image_id"),
				},
				Validators: []validator.String{
					validators.ImageIDValidator(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A human-readable description of the server. Maximum 1000 characters.",
				Optional:            true,
			},
			"keypair": schema.StringAttribute{
				MarkdownDescription: "The name of the SSH keypair to inject into the server for authentication. **Changing this attribute is not supported and will be rejected at plan time.** Use the `zillaforge_keypairs` data source to list available keypairs or create a new one with the `zillaforge_keypair` resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					modifiers.ImmutableAttributePlanModifier("keypair"),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for the server. Must be base64-encoded. **Changing this attribute is not supported and will be rejected at plan time.** This attribute is sensitive and will not appear in logs or plan output.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					modifiers.ImmutableAttributePlanModifier("password"),
				},
			},
			"user_data": schema.StringAttribute{
				MarkdownDescription: "Cloud-init user data for configuring the server on first boot. Must be base64-encoded (use Terraform's `base64encode()` function). Maximum size 64KB. **Changing this attribute is not supported and will be rejected at plan time.** The user data is not returned by the API for security reasons, so it will not appear in state after import.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					modifiers.ImmutableAttributePlanModifier("user_data"),
				},
			},
			"wait_for_active": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the server to reach `active` status after creation. **This value is used only during create/apply and is not stored in state; changing it does not trigger resource updates.** When set to `true` (default), Terraform will poll the server status until it reaches `active` state or the timeout is exceeded. When set to `false`, Terraform will return immediately after the API responds, without waiting for the server to become active. Default is `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					modifiers.IgnoreChangeAttributePlanModifierBool("wait_for_active"),
				},
			}, "wait_for_deleted": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the server to be fully deleted. **This value is used only during delete/apply and is not stored in state; changing it does not trigger resource updates.** When set to `true` (default), Terraform will poll the server status until it is fully deleted or the timeout is exceeded. When set to `false`, Terraform will return immediately after the delete API call, without waiting for the server deletion to complete. Default is `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					modifiers.IgnoreChangeAttributePlanModifierBool("wait_for_deleted"),
				},
			}, "status": schema.StringAttribute{
				MarkdownDescription: "The current status of the server. Possible values: `building` (instance is being created), `active` (instance is running and ready), `error` (instance entered an error state), `deleted` (instance has been deleted).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_addresses": schema.ListAttribute{
				MarkdownDescription: "List of IP addresses assigned to the server. The order corresponds to the order of `network_attachment` blocks. Includes both DHCP-assigned and fixed IP addresses.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					// Mark as unknown when network_attachment changes to avoid inconsistent state errors
					modifiers.IPAddressesUnknownOnNetworkChange(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the server was created, in RFC3339 format (e.g., `2023-10-15T14:30:00Z`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *cloudsdk.ProjectClient, got something else",
		)
		return
	}

	r.client = client
}

func (r *ServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourcemodels.ServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating server", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	// Build create request
	createReq, diags := helper.BuildServerCreateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log presence of key creation fields (do not log secret contents)
	tflog.Debug(ctx, "Create request fields", map[string]interface{}{
		"has_password": createReq.Password != "",
		"has_keypair":  createReq.KeypairID != "",
	})

	// Call API
	vpsClient := r.client.VPS()
	serverRes, err := vpsClient.Servers().Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Create Error",
			fmt.Sprintf("Unable to create server: %s", err),
		)
		return
	}

	tflog.Info(ctx, "Server created", map[string]interface{}{
		"id":     serverRes.Server.ID,
		"status": serverRes.Server.Status,
	})

	// Wait for active status if requested
	waitForActive := true
	if !plan.WaitForActive.IsNull() {
		waitForActive = plan.WaitForActive.ValueBool()
	}

	if waitForActive {
		// Get timeout from config (default 10m)
		timeout := 10 * time.Minute
		var timeoutsModel resourcemodels.TimeoutsModel
		if !plan.Timeouts.IsNull() {
			resp.Diagnostics.Append(plan.Timeouts.As(ctx, &timeoutsModel, basetypes.ObjectAsOptions{})...)
			if !resp.Diagnostics.HasError() && !timeoutsModel.Create.IsNull() {
				if d, err := time.ParseDuration(timeoutsModel.Create.ValueString()); err == nil {
					timeout = d
				}
			}
		}

		tflog.Debug(ctx, "Waiting for server to become active", map[string]interface{}{
			"timeout": timeout.String(),
		})

		serverRes, err = helper.WaitForServerActive(ctx, vpsClient.Servers(), serverRes.Server.ID, timeout)
		if err != nil {
			resp.Diagnostics.AddError(
				"Create Error",
				fmt.Sprintf("Server created but failed to reach active state: %s", err),
			)
			return
		}
	}

	// Map response to state
	state, diags := helper.MapServerToState(ctx, serverRes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder network_attachment to match user-provided plan order (preserve which block was marked primary)
	// We prefer plan's order/primary selection since the API does not expose a stable primary flag.
	var planNetworkAttachments []resourcemodels.NetworkAttachmentModel
	planDiags := plan.NetworkAttachment.ElementsAs(ctx, &planNetworkAttachments, false)
	// Only attempt reorder if we can successfully read the plan attachments
	if len(planDiags) == 0 {
		// Build a map from network_id -> ServerNIC for quick lookup
		nics, err := serverRes.NICs().List(ctx)
		if err == nil {
			nicMap := make(map[string]*servermodels.ServerNIC, len(nics))
			for _, nic := range nics {
				nicMap[nic.NetworkID] = nic
			}

			networkAttachmentAttrTypes := map[string]attr.Type{
				"network_id":         types.StringType,
				"ip_address":         types.StringType,
				"primary":            types.BoolType,
				"security_group_ids": types.ListType{ElemType: types.StringType},
			}

			ordered := make([]attr.Value, 0, len(nics))
			// Add attachments in plan order
			for _, p := range planNetworkAttachments {
				nid := p.NetworkID.ValueString()
				nic := nicMap[nid]
				// Map SecurityGroupIDs: prefer plan-specified order if present, otherwise use API NIC SGIDs (sorted deterministically)
				sgVals := make([]attr.Value, 0)
				var planSGs []types.String
				if d := p.SecurityGroupIDs.ElementsAs(ctx, &planSGs, false); len(d) == 0 && len(planSGs) > 0 {
					for _, sg := range planSGs {
						sgVals = append(sgVals, types.StringValue(sg.ValueString()))
					}
				} else if nic != nil {
					sgIDs := make([]string, len(nic.SGIDs))
					copy(sgIDs, nic.SGIDs)
					sort.Strings(sgIDs)
					for _, sg := range sgIDs {
						sgVals = append(sgVals, types.StringValue(sg))
					}
				}
				sgList, d := types.ListValue(types.StringType, sgVals)
				diags.Append(d...)

				// IP address from NIC if available
				ipAddress := types.StringNull()
				if nic != nil && len(nic.Addresses) > 0 {
					ipAddress = types.StringValue(nic.Addresses[0])
				}

				attObj, d := types.ObjectValue(networkAttachmentAttrTypes, map[string]attr.Value{
					"network_id":         types.StringValue(nid),
					"ip_address":         ipAddress,
					"primary":            types.BoolValue(p.Primary.ValueBool()),
					"security_group_ids": sgList,
				})
				diags.Append(d...)
				ordered = append(ordered, attObj)
				// Remove from nicMap so remaining NICs can be appended later
				delete(nicMap, nid)
			}

			// Append any NICs not present in the plan (sorted by NetworkID)
			remaining := make([]*servermodels.ServerNIC, 0, len(nicMap))
			for _, nic := range nicMap {
				remaining = append(remaining, nic)
			}
			// Deterministic sort
			if len(remaining) > 1 {
				sort.SliceStable(remaining, func(i, j int) bool {
					return remaining[i].NetworkID < remaining[j].NetworkID
				})
			}
			for _, nic := range remaining {
				sgIDs := make([]string, len(nic.SGIDs))
				copy(sgIDs, nic.SGIDs)
				sort.Strings(sgIDs)
				sgVals := make([]attr.Value, 0, len(sgIDs))
				for _, sg := range sgIDs {
					sgVals = append(sgVals, types.StringValue(sg))
				}
				sgList, d := types.ListValue(types.StringType, sgVals)
				diags.Append(d...)

				ipAddress := types.StringNull()
				if len(nic.Addresses) > 0 {
					ipAddress = types.StringValue(nic.Addresses[0])
				}

				attObj, d := types.ObjectValue(networkAttachmentAttrTypes, map[string]attr.Value{
					"network_id":         types.StringValue(nic.NetworkID),
					"ip_address":         ipAddress,
					"primary":            types.BoolValue(false),
					"security_group_ids": sgList,
				})
				diags.Append(d...)
				ordered = append(ordered, attObj)
			}

			// Build final list and set on state
			networkAttachmentList, d := types.ListValue(
				types.ObjectType{AttrTypes: networkAttachmentAttrTypes},
				ordered,
			)
			diags.Append(d...)
			if !d.HasError() {
				state.NetworkAttachment = networkAttachmentList
			}
		}
	}

	// Preserve user-provided values that aren't returned by API
	// NOTE: wait_for_active, wait_for_deleted and timeouts are runtime-only
	// Preserve user-provided values that aren't returned by API
	state.UserData = plan.UserData // API doesn't return user_data for security
	state.Password = plan.Password // API doesn't return password for security
	state.Keypair = plan.Keypair

	// Store runtime-only config in state during Create (they will be ignored during updates)
	state.WaitForActive = plan.WaitForActive
	state.WaitForDeleted = plan.WaitForDeleted
	state.Timeouts = plan.Timeouts

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourcemodels.ServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading server", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Call API
	vpsClient := r.client.VPS()
	server, err := vpsClient.Servers().Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Read Error",
			fmt.Sprintf("Unable to read server: %s", err),
		)
		return
	}

	// Map response to state
	newState, diags := helper.MapServerToState(ctx, server)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reorder network_attachment to prefer existing state order (stable across reads)
	var prevNetworkAttachments []resourcemodels.NetworkAttachmentModel
	if d := state.NetworkAttachment.ElementsAs(ctx, &prevNetworkAttachments, false); len(d) == 0 && len(prevNetworkAttachments) > 0 {
		var apiNetworkAttachments []resourcemodels.NetworkAttachmentModel
		if d2 := newState.NetworkAttachment.ElementsAs(ctx, &apiNetworkAttachments, false); len(d2) == 0 {
			// Build map from network_id -> attachment
			nicMap := make(map[string]resourcemodels.NetworkAttachmentModel, len(apiNetworkAttachments))
			for _, a := range apiNetworkAttachments {
				nicMap[a.NetworkID.ValueString()] = a
			}

			networkAttachmentAttrTypes := map[string]attr.Type{
				"network_id":         types.StringType,
				"ip_address":         types.StringType,
				"primary":            types.BoolType,
				"security_group_ids": types.ListType{ElemType: types.StringType},
			}

			ordered := make([]attr.Value, 0, len(apiNetworkAttachments))
			// Add attachments in previous state order when possible
			for _, p := range prevNetworkAttachments {
				nid := p.NetworkID.ValueString()
				if nic, ok := nicMap[nid]; ok {
					// Prefer previous state's security group ordering when possible to avoid spurious diffs.
					// Use previous state's ordering filtered by what's actually present in the API; append any
					// remaining API SGs deterministically (sorted).
					sgVals := make([]attr.Value, 0)
					var prevSGs []types.String
					if d := p.SecurityGroupIDs.ElementsAs(ctx, &prevSGs, false); len(d) == 0 && len(prevSGs) > 0 {
						// Build set of SG IDs returned by API for this NIC
						var apiSGs []types.String
						resp.Diagnostics.Append(nic.SecurityGroupIDs.ElementsAs(ctx, &apiSGs, false)...) // append errors if any
						apiSet := make(map[string]struct{}, len(apiSGs))
						for _, s := range apiSGs {
							apiSet[s.ValueString()] = struct{}{}
						}
						// Add in previous state's order when present in the API
						for _, s := range prevSGs {
							if _, ok := apiSet[s.ValueString()]; ok {
								sgVals = append(sgVals, types.StringValue(s.ValueString()))
								delete(apiSet, s.ValueString())
							}
						}
						// Append any remaining API SGs deterministically (sorted)
						if len(apiSet) > 0 {
							remaining := make([]string, 0, len(apiSet))
							for s := range apiSet {
								remaining = append(remaining, s)
							}
							sort.Strings(remaining)
							for _, s := range remaining {
								sgVals = append(sgVals, types.StringValue(s))
							}
						}
					} else {
						// Fallback: use API SGs sorted deterministically
						var apiSGs []types.String
						resp.Diagnostics.Append(nic.SecurityGroupIDs.ElementsAs(ctx, &apiSGs, false)...) // append errors if any
						sgStrings := make([]string, len(apiSGs))
						for i, sg := range apiSGs {
							sgStrings[i] = sg.ValueString()
						}
						sort.Strings(sgStrings)
						for _, sg := range sgStrings {
							sgVals = append(sgVals, types.StringValue(sg))
						}
					}
					sgList, d := types.ListValue(types.StringType, sgVals)
					resp.Diagnostics.Append(d...)

					ipAddress := types.StringNull()
					if !nic.IPAddress.IsNull() {
						ipAddress = nic.IPAddress
					}

					attObj, d := types.ObjectValue(networkAttachmentAttrTypes, map[string]attr.Value{
						"network_id":         types.StringValue(nid),
						"ip_address":         ipAddress,
						"primary":            types.BoolValue(nic.Primary.ValueBool()),
						"security_group_ids": sgList,
					})
					resp.Diagnostics.Append(d...)
					ordered = append(ordered, attObj)
					delete(nicMap, nid)
				}
			}

			// Append any NICs not present in the previous state (sorted by NetworkID)
			if len(nicMap) > 0 {
				remaining := make([]resourcemodels.NetworkAttachmentModel, 0, len(nicMap))
				for _, nic := range nicMap {
					remaining = append(remaining, nic)
				}
				if len(remaining) > 1 {
					sort.SliceStable(remaining, func(i, j int) bool {
						return remaining[i].NetworkID.ValueString() < remaining[j].NetworkID.ValueString()
					})
				}
				for _, nic := range remaining {
					var sgListVals []types.String
					resp.Diagnostics.Append(nic.SecurityGroupIDs.ElementsAs(ctx, &sgListVals, false)...) // append errors if any
					// Sort SG IDs to ensure deterministic ordering
					sgStrings := make([]string, len(sgListVals))
					for i, sg := range sgListVals {
						sgStrings[i] = sg.ValueString()
					}
					sort.Strings(sgStrings)
					sgVals := make([]attr.Value, 0, len(sgStrings))
					for _, sg := range sgStrings {
						sgVals = append(sgVals, types.StringValue(sg))
					}
					sgList, d := types.ListValue(types.StringType, sgVals)
					resp.Diagnostics.Append(d...)

					ipAddress := types.StringNull()
					if !nic.IPAddress.IsNull() {
						ipAddress = nic.IPAddress
					}

					attObj, d := types.ObjectValue(networkAttachmentAttrTypes, map[string]attr.Value{
						"network_id":         types.StringValue(nic.NetworkID.ValueString()),
						"ip_address":         ipAddress,
						"primary":            types.BoolValue(nic.Primary.ValueBool()),
						"security_group_ids": sgList,
					})
					resp.Diagnostics.Append(d...)
					ordered = append(ordered, attObj)
				}
			}

			// Build final list and set on newState
			if len(ordered) > 0 {
				networkAttachmentList, d := types.ListValue(
					types.ObjectType{AttrTypes: networkAttachmentAttrTypes},
					ordered,
				)
				resp.Diagnostics.Append(d...)
				if !d.HasError() {
					newState.NetworkAttachment = networkAttachmentList
				}
			}
		}
	}

	// Preserve user-provided values that aren't returned by API
	newState.UserData = state.UserData
	newState.Password = state.Password
	newState.Keypair = state.Keypair

	// Preserve runtime-only config from existing state
	newState.WaitForActive = state.WaitForActive
	newState.WaitForDeleted = state.WaitForDeleted
	newState.Timeouts = state.Timeouts

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourcemodels.ServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting server", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Call API
	vpsClient := r.client.VPS()
	err := vpsClient.Servers().Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete Error",
			fmt.Sprintf("Unable to delete server: %s", err),
		)
		return
	}

	// Check if we should wait for deletion to complete
	waitForDeleted := true // default value
	if !state.WaitForDeleted.IsNull() {
		waitForDeleted = state.WaitForDeleted.ValueBool()
	}

	tflog.Debug(ctx, "Delete wait configuration", map[string]interface{}{
		"wait_for_deleted":         waitForDeleted,
		"wait_for_deleted_is_null": state.WaitForDeleted.IsNull(),
	})

	if !waitForDeleted {
		tflog.Info(ctx, "Server delete initiated (wait_for_deleted=false)", map[string]interface{}{
			"id": state.ID.ValueString(),
		})
		return
	}

	// Get timeout from config (default 10m)
	timeout := 10 * time.Minute
	var timeoutsModel resourcemodels.TimeoutsModel
	if !state.Timeouts.IsNull() {
		resp.Diagnostics.Append(state.Timeouts.As(ctx, &timeoutsModel, basetypes.ObjectAsOptions{})...)
		if !resp.Diagnostics.HasError() && !timeoutsModel.Delete.IsNull() {
			if d, err := time.ParseDuration(timeoutsModel.Delete.ValueString()); err == nil {
				timeout = d
			}
		}
	}

	tflog.Debug(ctx, "Waiting for server to be deleted", map[string]interface{}{
		"timeout": timeout.String(),
	})

	err = helper.WaitForServerDeleted(ctx, vpsClient.Servers(), state.ID.ValueString(), timeout)
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete Error",
			fmt.Sprintf("Server delete initiated but failed to confirm deletion: %s", err),
		)
		return
	}

	tflog.Info(ctx, "Server deleted successfully", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *ServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourcemodels.ServerResourceModel
	var state resourcemodels.ServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating server", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Build update request with only changed fields
	updateCtx, diags := helper.BuildServerUpdateRequest(ctx, plan, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only call Update APIs if there are changes to apply
	if updateCtx.HasChanges {
		vpsClient := r.client.VPS()

		// Update server attributes if needed
		if updateCtx.ServerUpdate.Name != "" || updateCtx.ServerUpdate.Description != "" {
			_, err := vpsClient.Servers().Update(ctx, state.ID.ValueString(), updateCtx.ServerUpdate)
			if err != nil {
				resp.Diagnostics.AddError(
					"Update Error",
					fmt.Sprintf("Unable to update server: %s", err),
				)
				return
			}

			tflog.Info(ctx, "Server update API called", map[string]interface{}{
				"id":                  state.ID.ValueString(),
				"name_changed":        updateCtx.ServerUpdate.Name != "",
				"description_changed": updateCtx.ServerUpdate.Description != "",
			})
		}

		// Handle network attachment changes (delete, create, update)
		if len(updateCtx.NetworksToDelete) > 0 || len(updateCtx.NetworksToCreate) > 0 || len(updateCtx.NetworkChanges) > 0 {
			serverRes, err := vpsClient.Servers().Get(ctx, state.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Update Error",
					fmt.Sprintf("Unable to get server for NIC updates: %s", err),
				)
				return
			}

			// Get all NICs to find the NIC ID for each network
			nics, err := serverRes.NICs().List(ctx)
			if err != nil {
				resp.Diagnostics.AddError(
					"Update Error",
					fmt.Sprintf("Unable to list server NICs: %s", err),
				)
				return
			}

			// Build a map of network IDs to NIC IDs
			nicByNetwork := make(map[string]string)
			for _, nic := range nics {
				nicByNetwork[nic.NetworkID] = nic.ID
			}

			nicsClient := serverRes.NICs()

			// Step 1: Create new NICs first (before deleting old ones to ensure server always has at least one NIC)
			for _, nicCreate := range updateCtx.NetworksToCreate {
				// Retry Add operation a few times for transient failures (e.g., neutron IP allocation edge cases)
				var addErr error
				for attempt := 1; attempt <= 3; attempt++ {
					_, addErr = nicsClient.Add(ctx, &nicCreate)
					if addErr == nil {
						break
					}
					// If this looks like an IP allocation error from Neutron, retry a couple times
					if strings.Contains(addErr.Error(), "is not a valid IP for the specified subnet") || strings.Contains(addErr.Error(), "(neutron)IP address") {
						if attempt < 3 {
							tflog.Warn(ctx, "Transient IP allocation error when adding NIC — retrying", map[string]interface{}{"attempt": attempt, "err": addErr.Error(), "network_id": nicCreate.NetworkID})
							// small backoff
							time.Sleep(2 * time.Second)
							continue
						}
					}
					// Non-transient or final attempt: break and surface error
					break
				}

				if addErr != nil {
					// If the Add failed due to an IP allocation error, try to pick a candidate IP in the network CIDR and retry
					if strings.Contains(addErr.Error(), "is not a valid IP for the specified subnet") || strings.Contains(addErr.Error(), "(neutron)IP address") {
						// Attempt to get network CIDR
						netRes, err := vpsClient.Networks().Get(ctx, nicCreate.NetworkID)
						if err == nil && netRes != nil && netRes.Network.CIDR != "" {
							cidr := netRes.Network.CIDR
							if ip, ipnet, err := net.ParseCIDR(cidr); err == nil {
								// Only handle IPv4 for now
								if ip4 := ip.To4(); ip4 != nil {
									// Try a few candidate host offsets
									for _, off := range []int{10, 20, 30, 40, 50} {
										cand := make(net.IP, len(ip4))
										copy(cand, ip4)
										// add offset to last octet(s)
										c := uint32(cand[0])<<24 | uint32(cand[1])<<16 | uint32(cand[2])<<8 | uint32(cand[3])
										c += uint32(off)
										cand[0] = byte((c >> 24) & 0xff)
										cand[1] = byte((c >> 16) & 0xff)
										cand[2] = byte((c >> 8) & 0xff)
										cand[3] = byte(c & 0xff)
										if ipnet.Contains(cand) {
											// Try add with FixedIP candidate
											tryReq := servermodels.ServerNICCreateRequest{
												NetworkID: nicCreate.NetworkID,
												SGIDs:     nicCreate.SGIDs,
												FixedIP:   cand.String(),
											}
											_, err := nicsClient.Add(ctx, &tryReq)
											if err == nil {
												addErr = nil
												break
											}
											// If candidate fails, continue to next
										}
									}
								}
							}
						}
					}

					if addErr != nil {
						resp.Diagnostics.AddError(
							"Update Error",
							fmt.Sprintf("Unable to create NIC for network %s: %s", nicCreate.NetworkID, addErr),
						)
						return
					}
				}

				tflog.Info(ctx, "NIC created", map[string]interface{}{
					"id":         state.ID.ValueString(),
					"network_id": nicCreate.NetworkID,
					"sg_count":   len(nicCreate.SGIDs),
				})
			}

			// Step 2: Delete removed NICs (safe now that new NICs are created)
			for _, networkID := range updateCtx.NetworksToDelete {
				nicID, exists := nicByNetwork[networkID]
				if !exists {
					tflog.Warn(ctx, "NIC not found for deletion (may already be removed)", map[string]interface{}{
						"network_id": networkID,
					})
					continue
				}

				err := nicsClient.Delete(ctx, nicID)
				if err != nil {
					resp.Diagnostics.AddError(
						"Update Error",
						fmt.Sprintf("Unable to delete NIC for network %s: %s", networkID, err),
					)
					return
				}

				tflog.Info(ctx, "NIC deleted", map[string]interface{}{
					"id":         state.ID.ValueString(),
					"network_id": networkID,
					"nic_id":     nicID,
				})
			}

			// Step 3: Update security groups for existing networks
			for networkID, nicUpdate := range updateCtx.NetworkChanges {
				nicID, exists := nicByNetwork[networkID]
				if !exists {
					resp.Diagnostics.AddError(
						"Update Error",
						fmt.Sprintf("NIC not found for network %s", networkID),
					)
					return
				}

				_, err := nicsClient.Update(ctx, nicID, &nicUpdate)
				if err != nil {
					resp.Diagnostics.AddError(
						"Update Error",
						fmt.Sprintf("Unable to update security groups for network %s: %s", networkID, err),
					)
					return
				}

				tflog.Info(ctx, "NIC security groups updated", map[string]interface{}{
					"id":         state.ID.ValueString(),
					"network_id": networkID,
					"nic_id":     nicID,
					"sg_count":   len(nicUpdate.SGIDs),
				})
			}
		}

		// Get timeout from config (default 10m)
		timeout := 10 * time.Minute
		var timeoutsModel resourcemodels.TimeoutsModel
		if !plan.Timeouts.IsNull() {
			resp.Diagnostics.Append(plan.Timeouts.As(ctx, &timeoutsModel, basetypes.ObjectAsOptions{})...)
			if !resp.Diagnostics.HasError() && !timeoutsModel.Update.IsNull() {
				if d, err := time.ParseDuration(timeoutsModel.Update.ValueString()); err == nil {
					timeout = d
				}
			}
		}

		// Wait for server to return to active status after update
		tflog.Debug(ctx, "Waiting for server to become active after update", map[string]interface{}{
			"timeout": timeout.String(),
		})

		serverRes, err := helper.WaitForServerActive(ctx, vpsClient.Servers(), state.ID.ValueString(), timeout)
		if err != nil {
			resp.Diagnostics.AddError(
				"Update Error",
				fmt.Sprintf("Server updated but failed to return to active state: %s", err),
			)
			return
		}

		// Map updated server state
		newState, diags := helper.MapServerToState(ctx, serverRes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Reorder network_attachment to match plan order (to avoid spurious diffs)
		// This is critical after adding/removing NICs to ensure computed fields (IP addresses) are correct
		var planNetworkAttachments []resourcemodels.NetworkAttachmentModel
		planDiags := plan.NetworkAttachment.ElementsAs(ctx, &planNetworkAttachments, false)
		if len(planDiags) == 0 {
			// Fetch fresh NICs
			nics, err := serverRes.NICs().List(ctx)
			if err == nil {
				nicMap := make(map[string]*servermodels.ServerNIC, len(nics))
				for _, nic := range nics {
					nicMap[nic.NetworkID] = nic
				}

				networkAttachmentAttrTypes := map[string]attr.Type{
					"network_id":         types.StringType,
					"ip_address":         types.StringType,
					"primary":            types.BoolType,
					"security_group_ids": types.ListType{ElemType: types.StringType},
				}

				ordered := make([]attr.Value, 0, len(planNetworkAttachments))
				// Add attachments in plan order
				for _, p := range planNetworkAttachments {
					nid := p.NetworkID.ValueString()
					nic := nicMap[nid]
					if nic == nil {
						// NIC not found - this shouldn't happen after successful add, but handle gracefully
						tflog.Warn(ctx, "NIC not found for planned network after update", map[string]interface{}{"network_id": nid})
						continue
					}

					// Map SecurityGroupIDs: prefer plan-specified order if present
					sgVals := make([]attr.Value, 0)
					var planSGs []types.String
					if d := p.SecurityGroupIDs.ElementsAs(ctx, &planSGs, false); len(d) == 0 && len(planSGs) > 0 {
						for _, sg := range planSGs {
							sgVals = append(sgVals, types.StringValue(sg.ValueString()))
						}
					} else {
						sgIDs := make([]string, len(nic.SGIDs))
						copy(sgIDs, nic.SGIDs)
						sort.Strings(sgIDs)
						for _, sg := range sgIDs {
							sgVals = append(sgVals, types.StringValue(sg))
						}
					}
					sgList, d := types.ListValue(types.StringType, sgVals)
					diags.Append(d...)

					// IP address from NIC (this is the critical part - use actual assigned IP)
					ipAddress := types.StringNull()
					if len(nic.Addresses) > 0 {
						ipAddress = types.StringValue(nic.Addresses[0])
					}

					attObj, d := types.ObjectValue(networkAttachmentAttrTypes, map[string]attr.Value{
						"network_id":         types.StringValue(nid),
						"ip_address":         ipAddress,
						"primary":            types.BoolValue(p.Primary.ValueBool()),
						"security_group_ids": sgList,
					})
					diags.Append(d...)
					ordered = append(ordered, attObj)
					// Remove from nicMap so remaining NICs can be appended later (if any)
					delete(nicMap, nid)
				}

				// Build final list and set on newState
				networkAttachmentList, d := types.ListValue(
					types.ObjectType{AttrTypes: networkAttachmentAttrTypes},
					ordered,
				)
				diags.Append(d...)
				if !d.HasError() {
					newState.NetworkAttachment = networkAttachmentList
				}
			}
		}

		// Preserve user-provided values that aren't returned by API
		newState.UserData = plan.UserData
		newState.Password = plan.Password
		newState.Keypair = plan.Keypair

		// Preserve runtime-only config from plan (these can be changed without triggering server updates)
		newState.WaitForActive = plan.WaitForActive
		newState.WaitForDeleted = plan.WaitForDeleted
		newState.Timeouts = plan.Timeouts

		resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
	} else {
		// No supported changes detected - but still need to update runtime-only config from plan
		tflog.Debug(ctx, "No updatable fields changed")

		// Even though no API calls are needed, we still need to update runtime-only attributes
		// in state to match the plan (these don't trigger actual server updates)
		state.WaitForActive = plan.WaitForActive
		state.WaitForDeleted = plan.WaitForDeleted
		state.Timeouts = plan.Timeouts

		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func (r *ServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// T053: Import by server ID
	serverID := req.ID

	tflog.Debug(ctx, "Importing server", map[string]interface{}{
		"id": serverID,
	})

	// T054: Fetch server from API to validate it exists
	vpsClient := r.client.VPS()
	serverRes, err := vpsClient.Servers().Get(ctx, serverID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read server %s: %s", serverID, err),
		)
		return
	}

	// Build state from server response
	state, diags := helper.MapServerToState(ctx, serverRes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// T055: user_data and password are never available after creation (security), set to null
	// These are already set to null in mapServerToState helper
	state.UserData = types.StringNull()
	state.Password = types.StringNull()

	// Set default values for client-side flags (not stored in API)
	state.WaitForActive = types.BoolValue(true)  // Default behavior
	state.WaitForDeleted = types.BoolValue(true) // Default behavior

	// Set timeouts to null (not stored in API, user can configure in Terraform)
	timeoutsAttrTypes := map[string]attr.Type{
		"create": types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	}
	timeoutsNull := types.ObjectNull(timeoutsAttrTypes)
	state.Timeouts = timeoutsNull

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Info(ctx, "Imported server", map[string]interface{}{
		"id": serverRes.Server.ID,
	})
}
