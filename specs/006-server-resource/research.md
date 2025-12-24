# Research: Server Resource (VPS Virtual Machine)

**Feature**: Server Resource  
**Branch**: `006-server-resource`  
**Date**: 2025-12-17

## Overview

This document captures research findings for implementing the `zillaforge_server` resource using `github.com/Zillaforge/cloud-sdk`. Research focuses on cloud-SDK API structure, Terraform Plugin Framework best practices for complex resources, validation patterns for nested blocks, and state management for long-running operations.

---

## Research Tasks

### 1. Cloud-SDK Server/Instance API Structure

**Question**: What is the exact API structure for VPS server instances in `github.com/Zillaforge/cloud-sdk`?

**Findings**:
Based on analysis of existing code patterns (keypairs, networks, flavors, security groups), the expected structure is:

```go
// Import pattern
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    servermodels "github.com/Zillaforge/cloud-sdk/models/vps/servers"
)

// Client access chain
vpsClient := projectClient.VPS()
serverClient := vpsClient.Servers()

// Expected API methods
server, err := serverClient.Get(ctx, id)
servers, err := serverClient.List(ctx, &servermodels.ListOptions{Name: "filter"})
server, err := serverClient.Create(ctx, &servermodels.CreateRequest{...})
server, err := serverClient.Update(ctx, id, &servermodels.UpdateRequest{...})
err := serverClient.Delete(ctx, id)

// Status checking for polling
server, err := serverClient.Get(ctx, id)
// server.Status == "active" | "building" | "error" | "deleted"
```

**Expected Model Structures**:

```go
package servermodels

// Server represents a VPS instance
type Server struct {
    ID               string
    Name             string
    Description      string
    FlavorID         string
    ImageID          string
    Status           string // "building", "active", "error", "deleted"
    NetworkPorts     []NetworkPort
    KeypairID        string
    Password         string // Base64 encoded (if set by autoscale group)
    UserData         string
    IPAddresses      []string // Computed IP addresses
    CreatedAt        time.Time
}

// NetworkPort represents a network interface attachment
type NetworkPort struct {
    NetworkID string
    IPAddress string   // Empty for DHCP, set for fixed IP
    IsPrimary bool
    SGIDs     []string // Security group IDs attached to this NIC
}

// CreateRequest for server creation
type CreateRequest struct {
    Name             string
    Description      string
    FlavorID         string
    ImageID          string
    NetworkPorts     []NetworkPortSpec
    KeypairID        string
    Password         string // Base64 encoded
    UserData         string // Base64 encoded cloud-init
}

// NetworkPortSpec for creation
type NetworkPortSpec struct {
    NetworkID string
    IPAddress string   // Optional fixed IP
    IsPrimary bool
    SGIDs     []string // Security group IDs to apply to this NIC
}

// UpdateRequest for in-place updates
type UpdateRequest struct {
    Name         *string
    Description  *string
    NetworkPorts []NetworkPortSpec // Update network attachments (each NIC's SGIDs can be changed)
}

// ListOptions for filtering
type ListOptions struct {
    Name   string
    Status string
}
```

**Decision**: Use cloud-sdk server client following established VPS resource patterns. API structure follows OpenStack Nova conventions (server = instance).

**Alternatives Considered**:
- Direct HTTP client: Rejected - violates consistency with existing resources
- Custom SDK wrapper: Rejected - cloud-sdk is the official client

---

### 2. Nested Block Validation for network_attachment

**Question**: How to validate `network_attachment` nested block constraints (at most one `primary=true`, valid IP format)?

**Findings**:

Terraform Plugin Framework supports custom validators at the attribute and block level:

```go
// In schema definition
Schema: schema.Schema{
    Attributes: map[string]schema.Attribute{
        "network_attachment": schema.ListNestedAttribute{
            NestedObject: schema.NestedAttributeObject{
                Attributes: map[string]schema.Attribute{
                    "network_id": schema.StringAttribute{Required: true},
                    "ip_address": schema.StringAttribute{
                        Optional: true,
                        Validators: []validator.String{
                            validators.IPv4Address(), // Custom validator
                        },
                    },
                    "primary": schema.BoolAttribute{Optional: true},
                },
            },
            Required: true,
            Validators: []validator.List{
                validators.NetworkAttachmentPrimaryConstraint(), // Custom validator
            },
        },
    },
}
```

**Custom Validator Implementation**:

```go
// File: internal/vps/validators/network_attachment_validator.go
package validators

import (
    "context"
    "fmt"

    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// NetworkAttachmentPrimaryConstraint validates at most one primary=true
type NetworkAttachmentPrimaryConstraint struct{}

func (v NetworkAttachmentPrimaryConstraint) Description(ctx context.Context) string {
    return "ensures at most one network attachment has primary=true"
}

func (v NetworkAttachmentPrimaryConstraint) MarkdownDescription(ctx context.Context) string {
    return "ensures at most one network attachment has `primary=true`"
}

func (v NetworkAttachmentPrimaryConstraint) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
    if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
        return
    }

    var attachments []NetworkAttachmentModel
    diags := req.ConfigValue.ElementsAs(ctx, &attachments, false)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    primaryCount := 0
    for _, att := range attachments {
        if !att.Primary.IsNull() && att.Primary.ValueBool() {
            primaryCount++
        }
    }

    if primaryCount > 1 {
        resp.Diagnostics.AddAttributeError(
            req.Path,
            "Multiple Primary Network Attachments",
            fmt.Sprintf("Only one network attachment can have primary=true, found %d", primaryCount),
        )
    }
}
```

**Decision**: Use custom list validator for `network_attachment` to enforce primary constraint. Use built-in string validators for IP address format.

**Alternatives Considered**:
- Plan modifier validation: Rejected - validators are the correct framework pattern
- Manual validation in Create/Update: Rejected - duplicates logic, poor UX (late error detection)

---

### 3. State Polling for "active" Status

**Question**: Best practice for waiting until instance reaches "active" state during Create operation, and should this be configurable?

**Findings**:

**Option 1: Always Wait for Active (Traditional Approach)**

Terraform Plugin Testing framework and common provider patterns use a retry loop with context timeout:

```go
func (r *ServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // ... create API call ...

    // Poll for active status
    err := retry.RetryContext(ctx, 10*time.Minute, func() *retry.RetryError {
        server, err := serverClient.Get(ctx, serverID)
        if err != nil {
            return retry.NonRetryableError(err)
        }

        switch server.Status {
        case "active":
            return nil // Success
        case "error":
            return retry.NonRetryableError(fmt.Errorf("instance entered error state"))
        case "building":
            return retry.RetryableError(fmt.Errorf("instance still building"))
        default:
            return retry.RetryableError(fmt.Errorf("instance in transitional state: %s", server.Status))
        }
    })
}
```

**Option 2: Configurable Wait Behavior (Recommended)**

Allow users to control whether to wait for active status via `wait_for_active` attribute:

```go
func (r *ServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // ... create API call ...
    
    // Check if should wait for active status
    if plan.WaitForActive.ValueBool() {
        // Use cloud-sdk vps core waiter to poll for active status
        waitCtx, cancel := context.WithTimeout(ctx, timeout)
        defer cancel()

        err := vps.WaitForServerStatus(waitCtx, vps.ServerWaiterConfig{
            Client:       serverClient,
            ServerID:     serverID,
            TargetStatus: servermodels.ServerStatusActive,
        })
        if err != nil {
            return fmt.Errorf("waiting for server to become active: %w", err)
        }
    }
    // Otherwise, return immediately with current status (may be "building")
}
```

Benefits of configurable wait:
- **Faster deployments**: For batch creation, users can skip waiting and verify externally
- **Flexibility**: Autoscaling groups or external orchestration can manage readiness
- **Backward compatibility**: Default to `true` maintains traditional behavior

**Use cloud-sdk waiter helper**

The `github.com/Zillaforge/cloud-sdk` `vps` package provides a built-in waiter helper for waiting on server state transitions; prefer using the SDK waiter to reduce duplication and ensure consistent behavior (it respects context cancellation and supports timeouts).

Example usage (illustrative - adapt to the SDK's exact method name):

```go
// Use the cloud-sdk vps core waiter to wait for the "active" state.
if plan.WaitForActive.ValueBool() {
    waitCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    err := vps.WaitForServerStatus(waitCtx, vps.ServerWaiterConfig{
        Client:       serverClient,
        ServerID:     serverID,
        TargetStatus: servermodels.ServerStatusActive,
    })
    if err != nil {
        return fmt.Errorf("waiting for server to become active: %w", err)
    }
}
```

**Decision**: Add `wait_for_active` (Boolean, default: `true`) attribute to control waiting behavior. Use the cloud-sdk `vps` package waiter helper when waiting is enabled. Respect context timeout (default 10 minutes, configurable via timeouts block).

**Alternatives Considered**:
- Always wait: Rejected - inflexible for batch operations and external orchestration scenarios
- terraform-plugin-sdk retry package: Rejected - use SDK helper for consistency
- Fixed interval polling: Rejected - inefficient, wastes API calls

---

### 4. In-Place vs ForceReplace Update Strategy

**Question**: How to correctly mark attributes as updateable vs requiring replacement?

**Findings**:

Terraform Plugin Framework uses `PlanModifier` to control update behavior:

```go
// Attribute requires replacement when changed
"flavor": schema.StringAttribute{
    Required: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),
    },
},

// Attribute updateable in-place (default behavior)
"name": schema.StringAttribute{
    Required: true,
    // No RequiresReplace modifier = in-place update
},

// Computed attribute uses state for unknown values
"id": schema.StringAttribute{
    Computed: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},
```

**For network_attachment (including per-NIC `sg_ids`)** (updateable):
- No `RequiresReplace` modifier
- Implement diff logic in `Update()` method
- Use cloud-SDK update API to modify associations

**For flavor and image** (immutable):
- Add `RequiresReplace()` plan modifier
- Terraform will destroy + recreate automatically
- No update logic needed in `Update()` method

**Decision**: Use `RequiresReplace()` for flavor and image. Allow in-place updates for name, description, network_attachment (including `sg_ids`).

**Alternatives Considered**:
- Custom plan modifier for resize warning: Rejected - out of scope, too complex for MVP
- Force replacement for network changes: Rejected - spec requires in-place network updates

---

### 5. Import Functionality Best Practices

**Question**: How to implement import for server resource with nested network_attachment blocks?

**Findings**:

Import implementation requires:
1. `ResourceWithImportState` interface
2. Map API response to Terraform state
3. Handle nested blocks correctly

```go
func (r *ServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // ID from user: "srv-12345"
    serverID := req.ID

    // Call Read to populate state
    server, err := r.client.VPS().Servers().Get(ctx, serverID)
    if err != nil {
        resp.Diagnostics.AddError("Import Failed", err.Error())
        return
    }

    // Map to state model
    state := ServerResourceModel{
        ID:           types.StringValue(server.ID),
        Name:         types.StringValue(server.Name),
        FlavorID:     types.StringValue(server.FlavorID),
        // ... map all attributes ...
    }

    // Map network_attachment nested blocks
    var networkAttachments []NetworkAttachmentModel
    for _, port := range server.NetworkPorts {
        networkAttachments = append(networkAttachments, NetworkAttachmentModel{
            NetworkID: types.StringValue(port.NetworkID),
            IPAddress: types.StringPointerValue(port.IPAddress),
            Primary:   types.BoolValue(port.IsPrimary),
        })
    }
    state.NetworkAttachment, diags = types.ListValueFrom(ctx, types.ObjectType{...}, networkAttachments)
    resp.Diagnostics.Append(diags...)

    // Set state
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

**Testing Import**:

```go
resource.TestStep{
    Config: testAccServerConfig,
    ResourceName: "zillaforge_server.test",
    ImportState: true,
    ImportStateVerify: true,
    ImportStateVerifyIgnore: []string{"user_data"}, // Sensitive, not returned by API
}
```

**Decision**: Implement `ImportState` using Read logic. Map all attributes including nested blocks. Mark `user_data` as `ImportStateVerifyIgnore` since API doesn't return it for security.

**Alternatives Considered**:
- Separate import model: Rejected - adds complexity, Read logic sufficient
- Skip import for MVP: Rejected - import is P3 requirement in spec

---

## Best Practices Summary

### Cloud-SDK Integration
- Use `projectClient.VPS().Servers()` for all operations
- Expected methods: `Get()`, `List()`, `Create()`, `Update()`, `Delete()`
- Model structure follows OpenStack Nova patterns
- Status polling required for create (wait for "active")

### Schema Design
- Required attributes: name, flavor, image, network_attachment (each NIC must include `security_group_ids` for security groups)
- Optional attributes: description, keypair, user_data
- Computed attributes: id, status, ip_addresses, created_at
- Use `ListNestedAttribute` for network_attachment blocks
- Assign security groups per `network_attachment` using `security_group_ids` (list of string); each NIC should include at least one security group ID

### Validation
- Custom list validator for `network_attachment` primary constraint
- Ensure each `network_attachment` includes at least one `security_group_ids` entry (custom nested-block validator)
- IPv4 validator for `ip_address` field
- Leverage data sources for flavor/image/network/security group existence validation

### State Management
- Use cloud-sdk `vps` package waiter helper to wait for `active` status (respects context timeout; default 10 minutes, configurable via timeouts block)
- Respect context.Done() in all polling loops
- Handle "error" status gracefully with actionable diagnostics

### Update Strategy
- In-place: name, description, network_attachment (including `sg_ids` per NIC)
- Force replace: flavor, image
- Use `RequiresReplace()` plan modifier for immutable attributes
- Implement diff logic in Update() for mutable attributes

### Import
- Use Read logic to populate state from API response
- Map nested blocks (network_attachment) correctly
- Ignore `user_data` and `password` in import verification (sensitive, not returned by API)
- Test import with `ImportStateVerify: true`

### Testing
- Write acceptance tests FIRST (TDD)
- Test CRUD operations independently
- Test in-place updates vs force replacement separately
- Test import with drift detection
- Use `PreCheck` to validate environment variables
- Clean up resources in test teardown

---

## Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| API Client | `github.com/Zillaforge/cloud-sdk` | Official client, consistent with provider |
| Server API | `vpsClient.Servers()` | Follows VPS resource pattern |
| Framework | Terraform Plugin Framework v1.14.1 | Modern framework, better type safety |
| Testing | terraform-plugin-testing v1.11.0 | Official testing framework |
| Validation | Custom validators + built-in | Primary constraint, IP format validation |
| Polling | cloud-sdk `vps` waiter helper | Efficient, SDK-provided polling that respects timeouts |

---

## Open Questions

*All questions resolved during research phase. Ready to proceed to Phase 1 (Design).*

---

## References

- [Terraform Plugin Framework - Resources](https://developer.hashicorp.com/terraform/plugin/framework/resources)
- [Terraform Plugin Framework - Nested Attributes](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes/list-nested)
- [Terraform Plugin Framework - Validators](https://developer.hashicorp.com/terraform/plugin/framework/validation)
- [Terraform Plugin Framework - Plan Modifiers](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
- Existing implementations: `internal/vps/resource/keypair_resource.go`, `internal/vps/resource/security_group_resource.go`
