# Data Model: Server Resource

**Feature**: zillaforge_server  
**Date**: 2025-12-17

## Overview

This document defines the data structures for the `zillaforge_server` resource, including Terraform state models, cloud-SDK API models, and validation logic.

---

## Terraform Resource Model

**File**: `internal/vps/resource/server_resource.go`

```go
package resource

import (
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// ServerResourceModel represents the Terraform state for a server resource
type ServerResourceModel struct {
    // Required user-provided attributes
    Name              types.String `tfsdk:"name"`
    FlavorID          types.String `tfsdk:"flavor"` // Flavor ID (must be the resource ID; use data.zillaforge_flavors.*.flavors[0].id), not a human-readable name
    ImageID           types.String `tfsdk:"image"`  // Image ID (must be the resource ID; use data.zillaforge_images.*.images[0].id), not a human-readable name
    NetworkAttachment types.List   `tfsdk:"network_attachment"` // List of NetworkAttachmentModel

    // Optional user-provided attributes
    Description      types.String `tfsdk:"description"`
    Keypair          types.String `tfsdk:"keypair"` // Keypair ID used when launching servers; required if no password provided
    Password         types.String `tfsdk:"password"`
    UserData         types.String `tfsdk:"user_data"`
    WaitForActive    types.Bool   `tfsdk:"wait_for_active"` // Whether to wait for server to reach active status (default: true)
    WaitForDeleted   types.Bool   `tfsdk:"wait_for_deleted"` // Whether to wait for server to be fully deleted (default: true)

    // Computed attributes (read-only)
    ID          types.String `tfsdk:"id"`
    Status      types.String `tfsdk:"status"`
    IPAddresses types.List   `tfsdk:"ip_addresses"` // List of types.String
    CreatedAt   types.String `tfsdk:"created_at"`

    // Timeouts configuration
    Timeouts types.Object `tfsdk:"timeouts"` // TimeoutsModel
}

// NetworkAttachmentModel represents a network interface attachment
type NetworkAttachmentModel struct {
    NetworkID          types.String `tfsdk:"network_id"`
    IPAddress          types.String `tfsdk:"ip_address"` // Optional fixed IP
    Primary            types.Bool   `tfsdk:"primary"`    // Optional, at most one true
    SecurityGroupIDs   types.List   `tfsdk:"security_group_ids"`     // List of security group IDs applied to this NIC
}

// TimeoutsModel for configurable operation timeouts
type TimeoutsModel struct {
    Create types.String `tfsdk:"create"` // Default: "10m"
    Update types.String `tfsdk:"update"` // Default: "10m"
    Delete types.String `tfsdk:"delete"` // Default: "10m"
}
```

---

## Cloud-SDK API Models

**Import Path**: `github.com/Zillaforge/cloud-sdk/models/vps/servers`

### Server Entity

```go
package servermodels

import "time"

// Server represents a VPS instance in the ZillaForge platform
type Server struct {
    ID               string        `json:"id"`
    Name             string        `json:"name"`
    Description      string        `json:"description"`
    FlavorID         string        `json:"flavor_id"`
    ImageID          string        `json:"image_id"`
    Status           string        `json:"status"` // "building", "active", "error", "deleted"
    NetworkPorts     []NetworkPort `json:"network_ports"`
    KeypairID        string        `json:"keypair_id,omitempty"` // Keypair ID used when launching servers
    Password         string        `json:"password,omitempty"`    // Base64 encoded password, optional (used for autoscale-launched instances)
    UserData         string        `json:"user_data,omitempty"` // Not returned by GET for security
    IPAddresses      []string      `json:"ip_addresses"`
    CreatedAt        time.Time     `json:"created_at"`
}

// NetworkPort represents a network interface attachment
type NetworkPort struct {
    PortID            string   `json:"port_id"`     // Platform-generated port UUID
    NetworkID         string   `json:"network_id"`  // User-specified network
    IPAddress         string   `json:"ip_address"`  // Assigned IP (DHCP or fixed)
    IsPrimary         bool     `json:"is_primary"`  // Primary interface flag
    SecurityGroupIDs  []string `json:"security_group_ids,omitempty"` // Security group IDs attached to this NIC
}
```

### Create Request

```go
// CreateRequest for creating a new server instance
type CreateRequest struct {
    Name             string             `json:"name"`
    Description      string             `json:"description,omitempty"`
    FlavorID         string             `json:"flavor_id"`
    ImageID          string             `json:"image_id"`
    NetworkPorts     []NetworkPortSpec  `json:"network_ports"`
    KeypairID        string             `json:"keypair_id,omitempty"`
    Password         string             `json:"password,omitempty"`  // Base64 encoded
    UserData         string             `json:"user_data,omitempty"` // Base64 encoded
}

// NetworkPortSpec for specifying network interface during creation
type NetworkPortSpec struct {
    NetworkID         string   `json:"network_id"`
    IPAddress         string   `json:"ip_address,omitempty"` // Empty for DHCP, set for fixed IP
    IsPrimary         bool     `json:"is_primary"`
    SecurityGroupIDs  []string `json:"security_group_ids,omitempty"`
}

// Validation rules:
// - Name: 1-255 characters
// - Description: 0-1000 characters
// - FlavorID, ImageID, NetworkPorts: Required, non-empty
// - NetworkPorts: At least 1 port, at most 1 primary=true; each port must include at least one security group ID (`sg_ids`)
// - UserData: Max 64KB, base64 encoded
// - Password: Max 64KB, base64 encoded (optional)
// - Keypair: This is the Keypair ID used when launching servers; required if no password is provided
```

### Update Request

```go
// UpdateRequest for in-place updates to a server
type UpdateRequest struct {
    Name           *string            `json:"name,omitempty"`
    Description    *string            `json:"description,omitempty"`
    NetworkPorts   []NetworkPortSpec  `json:"network_ports,omitempty"`
    SecurityGroups []string           `json:"security_groups,omitempty"`
}

// Validation rules:
// - Only non-nil fields are updated
// - FlavorID, ImageID: NOT updateable (immutable)
// - NetworkPorts: Full replacement (cannot update individual ports)
// - SecurityGroups: Full replacement list
```

### List Options

```go
// ListOptions for filtering server list queries
type ListOptions struct {
    Name   string `json:"name,omitempty"`   // Filter by name (substring match)
    Status string `json:"status,omitempty"` // Filter by status
}
```

---

## Mapping Logic

### Terraform → Cloud-SDK (Create)

```go
func buildCreateRequest(ctx context.Context, plan ServerResourceModel) (*servermodels.CreateRequest, diag.Diagnostics) {
    var diags diag.Diagnostics

    // Map network_attachment blocks
    var networkAttachments []NetworkAttachmentModel
    diags.Append(plan.NetworkAttachment.ElementsAs(ctx, &networkAttachments, false)...)

    networkPorts := make([]servermodels.NetworkPortSpec, len(networkAttachments))
    for i, att := range networkAttachments {
        // Extract security_group_ids from nested block (security_group_ids is a list of strings)
        var sgList []types.String
        diags.Append(att.SecurityGroupIDs.ElementsAs(ctx, &sgList, false)...)

        securityGroupIDs := make([]string, len(sgList))
        for j, sg := range sgList {
            securityGroupIDs[j] = sg.ValueString()
        }

        networkPorts[i] = servermodels.NetworkPortSpec{
            NetworkID:        att.NetworkID.ValueString(),
            IPAddress:        att.IPAddress.ValueString(), // Empty if null
            IsPrimary:        att.Primary.ValueBool(),     // False if null
            SecurityGroupIDs: securityGroupIDs,
        }
    }

    req := &servermodels.CreateRequest{
        Name:             plan.Name.ValueString(),
        Description:      plan.Description.ValueString(),
        FlavorID:         plan.FlavorID.ValueString(),
        ImageID:          plan.ImageID.ValueString(),
        NetworkPorts:     networkPorts,
        KeypairID:        plan.Keypair.ValueString(),
        Password:         plan.Password.ValueString(),
        UserData:         plan.UserData.ValueString(),
    }

    return req, diags
}
```

### Cloud-SDK → Terraform (Read)

```go
func mapServerToState(ctx context.Context, server *servermodels.Server) (ServerResourceModel, diag.Diagnostics) {
    var diags diag.Diagnostics

    // Map network ports to network_attachment blocks
    networkAttachments := make([]NetworkAttachmentModel, len(server.NetworkPorts))
    for i, port := range server.NetworkPorts {
        // Map SecurityGroupIDs to types.List
        sgVals := make([]types.String, 0, len(port.SecurityGroupIDs))
        for _, sg := range port.SecurityGroupIDs {
            sgVals = append(sgVals, types.StringValue(sg))
        }
        sgList, d := types.ListValueFrom(ctx, types.StringType, sgVals)
        diags.Append(d...)

        networkAttachments[i] = NetworkAttachmentModel{
            NetworkID:        types.StringValue(port.NetworkID),
            IPAddress:        types.StringPointerValue(&port.IPAddress), // Null if empty
            Primary:          types.BoolValue(port.IsPrimary),
            SecurityGroupIDs: sgList,
        }
    }

    networkAttachmentList, diags := types.ListValueFrom(ctx, 
        types.ObjectType{AttrTypes: networkAttachmentAttrTypes},
        networkAttachments,
    )
    diags.Append(d...)

    // Map IP addresses to list
    ipAddressesList, d := types.ListValueFrom(ctx, types.StringType, server.IPAddresses)
    diags.Append(d...)

    state := ServerResourceModel{
        ID:                types.StringValue(server.ID),
        Name:              types.StringValue(server.Name),
        Description:       types.StringPointerValue(&server.Description),
        FlavorID:          types.StringValue(server.FlavorID),
        ImageID:           types.StringValue(server.ImageID),
        Status:            types.StringValue(server.Status),
        NetworkAttachment: networkAttachmentList,
        SecurityGroups:    securityGroupsList,
        Keypair:           types.StringPointerValue(&server.KeypairID),
        Password:          types.StringNull(), // Not returned by API
        UserData:          types.StringNull(), // Not returned by API
        IPAddresses:       ipAddressesList,
        CreatedAt:         types.StringValue(server.CreatedAt.Format(time.RFC3339)),
    }

    return state, diags
}
```

### Terraform → Cloud-SDK (Update)

```go
func buildUpdateRequest(ctx context.Context, plan, state ServerResourceModel) (*servermodels.UpdateRequest, diag.Diagnostics) {
    var diags diag.Diagnostics
    req := &servermodels.UpdateRequest{}

    // Update name if changed
    if !plan.Name.Equal(state.Name) {
        name := plan.Name.ValueString()
        req.Name = &name
    }

    // Update description if changed
    if !plan.Description.Equal(state.Description) {
        desc := plan.Description.ValueString()
        req.Description = &desc
    }

    // Disallow changing flavor/image in-place (resize/reprovision)
    if !plan.FlavorID.Equal(state.FlavorID) {
        diags.AddError("Unsupported Change: flavor_id", "Changing 'flavor_id' is a platform resize operation and is out of scope for in-place updates. Please recreate the instance or perform a manual resize in the ZillaForge platform.")
        return req, diags
    }
    if !plan.ImageID.Equal(state.ImageID) {
        diags.AddError("Unsupported Change: image_id", "Changing 'image_id' is not supported by in-place updates. This operation requires recreating the instance (replacement).")
        return req, diags
    }

    // TODO: network_attachment and security group updates are planned but not yet implemented.
    // For now, changes to these fields will be ignored and a TODO warning will be surfaced.

    return req, diags
}
```

---

## Validation Rules

### Schema-Level Validation

1. **network_attachment** (custom list validator):
   - At most one block has `primary=true`
   - At least one network_attachment block required

2. **ip_address** (built-in string validator):
   - Must be valid IPv4 address format
   - Validator: `stringvalidator.RegexMatches()` or custom IPv4 validator

3. **security_groups** (built-in list validator):
   - At least one security group required
   - Validator: `listvalidator.SizeAtLeast(1)`

### API-Level Validation

Handled by cloud-SDK:
- FlavorID exists and available
- ImageID exists and compatible with flavor
- NetworkID exists in project
- Security group IDs exist in project
- Availability zone valid for project
- Quota limits (max instances, max networks)

### State Validation

During Create/Update operations:
- UserData must be base64-encoded (if provided)
- Timeout values must be valid duration strings ("10m", "1h")

---

## State Transitions

### Server Status States

| Status | Description | Transition From | Next States |
|--------|-------------|-----------------|-------------|
| `building` | Instance provisioning in progress | (initial) | `active`, `error` |
| `active` | Instance running and ready | `building` | `deleted`, `error` |
| `error` | Instance entered error state | `building`, `active` | `deleted` |
| `deleted` | Instance deleted | `active`, `error` | (terminal) |

### Resource Lifecycle

```
terraform apply (create)
    ↓
[API] POST /servers → Status: "building"
    ↓
[Poll] GET /servers/{id} → Status: "building" (retry)
    ↓
[Poll] GET /servers/{id} → Status: "active" ✓
    ↓
Terraform State: Created

terraform apply (update in-place)
    ↓
[API] PATCH /servers/{id} → Status: "active"
    ↓
Terraform State: Updated

terraform destroy
    ↓
[API] DELETE /servers/{id}
    ↓
[Poll] GET /servers/{id} → 404 Not Found ✓
    ↓
Terraform State: Destroyed
```

---

## Entity Relationships

```
ServerResource
    ├── flavor (zillaforge_flavors data source)
    │   └── Validation: FlavorID must exist
    ├── image (zillaforge_images data source)
    │   └── Validation: ImageID must exist
    ├── network_attachment[]
    │   ├── network_id (zillaforge_networks data source)
    │   │   └── Validation: NetworkID must exist in project
    │   ├── ip_address (optional fixed IP)
    │   │   └── Validation: Must be valid IPv4 within network CIDR
    │   └── primary (boolean)
    │       └── Constraint: At most one true across all attachments
    ├── security_groups[] (zillaforge_security_groups data source)
    │   └── Validation: All IDs must exist in project
    └── keypair (zillaforge_keypairs data source, optional)
        └── Validation: Keypair name must exist in project
```

---

## Error Handling

### API Error Responses

| HTTP Code | Error Type | Handling Strategy |
|-----------|------------|-------------------|
| 400 | Bad Request | Return diagnostic with API error message |
| 404 | Not Found | Return "resource not found" diagnostic |
| 409 | Conflict | Return diagnostic (e.g., name already exists) |
| 422 | Validation Error | Return diagnostic with validation details |
| 429 | Rate Limit | Retry with exponential backoff |
| 500 | Server Error | Return diagnostic, suggest retry |

### Timeout Handling

```go
func getTimeout(ctx context.Context, timeouts TimeoutsModel, operation string) (time.Duration, error) {
    var timeoutStr string
    switch operation {
    case "create":
        timeoutStr = timeouts.Create.ValueString()
    case "update":
        timeoutStr = timeouts.Update.ValueString()
    case "delete":
        timeoutStr = timeouts.Delete.ValueString()
    default:
        return 10 * time.Minute, nil // Default
    }

    if timeoutStr == "" {
        return 10 * time.Minute, nil
    }

    return time.ParseDuration(timeoutStr)
}
```

---

## Testing Considerations

### Test Data Models

```go
// Test fixture for basic server
var testServerBasic = ServerResourceModel{
    Name:     types.StringValue("test-server"),
    FlavorID: types.StringValue("flavor-small"),
    ImageID:  types.StringValue("ubuntu-22.04"),
    NetworkAttachment: types.ListValueMust(types.ObjectType{...}, []attr.Value{
        types.ObjectValueMust(networkAttachmentAttrTypes, map[string]attr.Value{
            "network_id": types.StringValue("net-default"),
            "ip_address": types.StringNull(),
            "primary":    types.BoolValue(true),
        }),
    }),
    SecurityGroups: types.ListValueMust(types.StringType, []attr.Value{
        types.StringValue("sg-default"),
    }),
}
```

### Test Cases

1. **Create with minimum required fields**
2. **Create with all optional fields**
3. **Create with multiple network attachments**
4. **Update name (in-place)**
5. **Update network attachments (in-place)**
6. **Update security groups (in-place)**
7. **Update flavor (force replacement)** ← Should trigger destroy+create
8. **Import existing server**
9. **Validation: Multiple primary network attachments** ← Should fail
10. **Validation: Invalid IP address format** ← Should fail
11. **Create with wait_for_active=false** ← Should return immediately
12. **Delete with wait_for_deleted=true (default)** ← Should wait for deletion
13. **Delete with wait_for_deleted=false** ← Should return immediately after delete API call

---

## Behavior Flags

### wait_for_active

Controls whether Terraform waits for the server to reach `active` status after creation:
- **Default**: `true`
- **When true**: Terraform polls server status until it reaches `active` or timeout is exceeded
- **When false**: Terraform returns immediately after create API call, state may show `building` status

### wait_for_deleted

Controls whether Terraform waits for the server to be fully deleted:
- **Default**: `true`
- **When true**: Terraform polls server status until it is fully deleted or timeout is exceeded
- **When false**: Terraform returns immediately after delete API call, without confirming deletion completion
- **Use case**: Batch deletions, external orchestration, or when deletion confirmation is handled elsewhere

---

## Summary

- **Terraform Model**: `ServerResourceModel` with nested `NetworkAttachmentModel`, `TimeoutsModel`
- **Cloud-SDK Models**: `Server`, `CreateRequest`, `UpdateRequest`, `NetworkPort`, `NetworkPortSpec`
- **Mapping**: Bidirectional conversion with diagnostic handling
- **Validation**: Custom validators for network_attachment, built-in for IP format
- **State**: Poll for "active" status after create, handle timeouts
- **Errors**: Map HTTP errors to actionable Terraform diagnostics

Ready for Phase 1 contract generation.
