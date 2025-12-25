# Research: Floating IP Association with Network Attachments

**Feature**: 008-floating-ip-association  
**Date**: 2025-12-25

## Overview

This document contains technical research for implementing floating IP association/disassociation with network attachments in the zillaforge_server resource.

---

## 1. Cloud-SDK Floating IP Association API

**Question**: What SDK methods are available for associating/disassociating floating IPs with servers?

**Research Approach**: Examine cloud-sdk documentation and existing code patterns.

**Findings**:

Based on the cloud-sdk and existing implementations in `internal/vps/helper/server.go`, the SDK provides:

```go
// Actual SDK methods (confirmed from codebase)
import (
    servermodels "github.com/Zillaforge/cloud-sdk/models/vps/servers"
    serversdk "github.com/Zillaforge/cloud-sdk/modules/vps/servers"
    floatingipmodels "github.com/Zillaforge/cloud-sdk/models/vps/floatingips"
    floatingipsdk "github.com/Zillaforge/cloud-sdk/modules/vps/floatingips"
    vpscore "github.com/Zillaforge/cloud-sdk/modules/vps/core"
)

// 1. Wait for server ACTIVE (required before NIC operations)
err := vpscore.WaitForServerStatus(ctx, vpscore.ServerWaiterConfig{
    Client:       serversClient,
    ServerID:     serverID,
    TargetStatus: servermodels.ServerStatusActive,
})

// 2. Get server resource for NIC operations
server, err := serversClient.Get(ctx, serverID)

// 3. Associate floating IP via server NIC
req := servermodels.ServerNICAssociateFloatingIPRequest{
    FloatingIPID: floatingIPID,
}
err := server.NICs().AssociateFloatingIP(ctx, nicID, req)

// 4. Wait for floating IP to be associated (SDK waiter)
err := vpscore.WaitForFloatingIPStatus(ctx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusActive,
    TargetDeviceID: serverID,
})

// 5. Disassociate floating IP via Delete (does not delete resource)
err := floatingIPClient.Delete(ctx, floatingIPID)

// 6. Wait for disassociation (SDK waiter)
err := vpscore.WaitForFloatingIPStatus(ctx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusDown,
    TargetDeviceID: "",
})
```

**Decision**: 
1. **CRITICAL**: Must wait for server ACTIVE before floating IP association (NICs not ready during BUILD)
2. Association via `server.NICs().AssociateFloatingIP()` with NIC ID (not port_id)
3. Disassociation via `floatingIPClient.Delete()` (disassociates only, doesn't delete resource)
4. Use SDK waiter helpers (`vpscore.WaitForFloatingIPStatus`) like existing server waiter pattern
5. Map `network_id` from Terraform to NIC ID from server resource

**Rationale**: 
- Server ACTIVE status ensures NICs are ready for floating IP operations
- SDK provides purpose-built waiters for consistent polling behavior
- NIC-based association aligns with server's network interface model
- Delete for disassociation is SDK's semantic (separation of concerns)

---

## 2. Terraform Framework: Optional vs Computed Attributes

**Question**: How should `floating_ip_id` (user-provided) and `floating_ip` (computed) be modeled in the schema?

**Research Approach**: Review Terraform Plugin Framework documentation and existing server resource patterns.

**Findings**:

```go
// Schema definition pattern
"network_attachment": schema.ListNestedAttribute{
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            // ... existing attributes ...
            
            "floating_ip_id": schema.StringAttribute{
                MarkdownDescription: "UUID of the floating IP to associate with this network interface. When specified, the floating IP will be associated with this network attachment. Remove this attribute to disassociate the floating IP.",
                Optional: true,
                Validators: []validator.String{
                    stringvalidator.RegexMatches(
                        regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
                        "must be a valid UUID",
                    ),
                },
            },
            
            "floating_ip": schema.StringAttribute{
                MarkdownDescription: "The public IP address associated with this network interface. This is a read-only attribute that displays the IP address of the floating IP specified in floating_ip_id.",
                Computed: true,
            },
        },
    },
},
```

**Decision**: 
- `floating_ip_id`: Optional StringAttribute with UUID validation
- `floating_ip`: Computed StringAttribute (read-only, populated from SDK response)

**Rationale**: This pattern separates user intent (floating_ip_id) from infrastructure state (floating_ip), making it clear when users are configuring vs observing. The computed attribute provides immediate visibility of the associated IP address without requiring a separate data source query.

---

## 3. State Management: Handling Null and Unknown Values

**Question**: How should we handle floating_ip_id when it's null (not specified) vs when it's being removed (changed from value to null)?

**Research Approach**: Review Terraform Plugin Framework state diff semantics and Plan Modifiers.

**Findings**:

The framework distinguishes between:
- **Null**: Attribute not set (user didn't specify it)
- **Unknown**: Attribute value not yet determined (during plan phase)
- **Known**: Attribute has a concrete value

For our use case:
- **Create**: `floating_ip_id.IsNull()` → no association needed
- **Create**: `floating_ip_id.ValueString()` != "" → associate during/after creation
- **Update**: Plan has null, State has value → disassociate (user removed attribute)
- **Update**: Plan has value, State has different value → swap (disassociate old, associate new)
- **Update**: Plan has value, State has same value → no-op
- **Delete**: Automatically disassociate (handled in server Delete method)

```go
// Pseudo-code for Update logic
func (r *ServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan, state ServerResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    
    // Extract network attachments
    var planAttachments, stateAttachments []NetworkAttachmentModel
    resp.Diagnostics.Append(plan.NetworkAttachment.ElementsAs(ctx, &planAttachments, false)...)
    resp.Diagnostics.Append(state.NetworkAttachment.ElementsAs(ctx, &stateAttachments, false)...)
    
    // For each network attachment, check floating IP changes
    for i := range planAttachments {
        planFIPID := planAttachments[i].FloatingIPID
        stateFIPID := stateAttachments[i].FloatingIPID
        
        if !planFIPID.Equal(stateFIPID) {
            // Change detected
            if !stateFIPID.IsNull() {
                // Disassociate old floating IP
                DisassociateFloatingIP(ctx, stateFIPID.ValueString())
            }
            if !planFIPID.IsNull() {
                // Associate new floating IP
                portID := getPortIDForAttachment(i) // from server details
                AssociateFloatingIP(ctx, planFIPID.ValueString(), portID)
            }
        }
    }
}
```

**Decision**: Use `types.String.Equal()` to compare plan vs state values. Handle null-to-value (associate), value-to-null (disassociate), and value-to-different-value (swap) transitions explicitly.

**Rationale**: Terraform's state diff semantics provide clear signals for user intent. Explicit handling of each transition ensures correct behavior and clear error messages.

---

## 4. Polling and Synchronous Operations

**Question**: How should we implement "wait with polling" for floating IP operations given the SDK might return immediately?

**Research Approach**: Review existing server resource wait logic for active status.

**Findings**:

The server resource already implements polling for server status:

```go
// Existing pattern in server_resource.go
func waitForServerActive(ctx context.Context, client *vps.Client, serverID string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for server to become active")
        case <-ticker.C:
            server, err := client.GetServer(ctx, serverID)
            if err != nil {
                return err
            }
            if server.Status == "active" {
                return nil
            }
            if server.Status == "error" {
                return fmt.Errorf("server entered error state")
            }
        }
    }
}
```

**Decision**: Use SDK-provided waiter helpers similar to the existing server waiter pattern in `internal/vps/helper/server.go`:

```go
// WaitForFloatingIPAssociated follows the pattern from WaitForServerActive
func WaitForFloatingIPAssociated(
    ctx context.Context,
    floatingIPClient *floatingipsdk.Client,
    floatingIPID string,
    serverID string,
    timeout time.Duration,
) (*floatingipmodels.FloatingIP, error) {
    waitCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    if err := vpscore.WaitForFloatingIPStatus(waitCtx, vpscore.FloatingIPWaiterConfig{
        Client:         floatingIPClient,
        FloatingIPID:   floatingIPID,
        TargetStatus:   floatingipmodels.FloatingIPStatusActive,
        TargetDeviceID: serverID, // Wait for association to specific server
    }); err != nil {
        return nil, fmt.Errorf("waiting for floating IP to be associated: %w", err)
    }

    // After waiter completes, fetch the latest floating IP
    return floatingIPClient.Get(ctx, floatingIPID)
}

func WaitForFloatingIPDisassociated(
    ctx context.Context,
    floatingIPClient *floatingipsdk.Client,
    floatingIPID string,
    timeout time.Duration,
) (*floatingipmodels.FloatingIP, error) {
    waitCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    if err := vpscore.WaitForFloatingIPStatus(waitCtx, vpscore.FloatingIPWaiterConfig{
        Client:         floatingIPClient,
        FloatingIPID:   floatingIPID,
        TargetStatus:   floatingipmodels.FloatingIPStatusDown,
        TargetDeviceID: "", // Wait for empty device_id (disassociated)
    }); err != nil {
        return nil, fmt.Errorf("waiting for floating IP to be disassociated: %w", err)
    }

    // After waiter completes, fetch the latest floating IP
    return floatingIPClient.Get(ctx, floatingIPID)
}
```

Timeouts:
- Association: 30 seconds (per SC-001)
- Disassociation: 15 seconds (per SC-002)

**Rationale**: Consistent with existing server resource patterns. Polling every 2 seconds provides responsive feedback without excessive API calls. Timeouts enforce performance requirements from success criteria.

---

## 5. Error Handling and Validation

**Question**: What validation should occur before making SDK calls, and how should SDK errors be handled?

**Research Approach**: Review Terraform Plugin Framework validator patterns and existing resource error handling.

**Findings**:

**Pre-SDK Validations**:
1. UUID format for floating_ip_id (schema validator)
2. Floating IP exists (call GetFloatingIP before associate)
3. Floating IP not already in use (check DeviceID is empty)
4. Server is in ACTIVE status (check before association)

**SDK Error Handling**:
```go
// Fail immediately without retries (per clarification)
if err := client.AssociateFloatingIP(ctx, floatingIPID, req); err != nil {
    resp.Diagnostics.AddError(
        "Failed to Associate Floating IP",
        fmt.Sprintf("Could not associate floating IP %s with network interface: %s. "+
            "Verify the floating IP exists and is not already in use. "+
            "Ensure the server is in ACTIVE status.", floatingIPID, err.Error()),
    )
    return
}
```

**Decision**: 
- Use schema validators for format validation (UUID regex)
- Perform existence/state checks before operations
- Fail immediately on SDK errors with actionable messages
- Include specific failure reasons in diagnostics

**Rationale**: Pre-validation catches user errors early with clear messages. Immediate failure (no retries) matches clarified requirement and provides fast feedback. Actionable error messages guide users to resolution.

---

## 6. Import Support

**Question**: How should floating IP associations be handled during Terraform import?

**Research Approach**: Review existing server resource import logic.

**Findings**:

```go
// Existing import implementation
func (r *ServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

The ImportStatePassthroughID uses the server ID to fetch the full server state via Read method. Since Read will populate all attributes including floating IPs from the SDK response, import will automatically work.

**Decision**: No changes needed to import logic. The Read method will be updated to populate floating_ip_id and floating_ip from the server's NetworkPort details.

**Rationale**: Terraform's import mechanism calls Read after setting the ID. Since floating IP associations are part of the server state returned by the SDK, they'll be automatically populated during import.

---

## 7. Testing Strategy

**Question**: What test scenarios are needed for comprehensive coverage?

**Research Approach**: Map acceptance scenarios from spec to test cases, following TDD principles.

**Findings**:

**Test Cases** (all written BEFORE implementation):

1. **TestAccServerResource_FloatingIP_Associate**
   - Create server with floating_ip_id in network_attachment
   - Verify floating IP is associated
   - Verify floating_ip computed attribute shows correct address

2. **TestAccServerResource_FloatingIP_AssociateExisting**
   - Create server without floating IP
   - Update to add floating_ip_id
   - Verify association occurs

3. **TestAccServerResource_FloatingIP_Disassociate**
   - Create server with floating IP
   - Update to remove floating_ip_id
   - Verify disassociation occurs
   - Verify floating_ip computed attribute is empty

4. **TestAccServerResource_FloatingIP_Swap**
   - Create server with floating IP A
   - Update to use floating IP B
   - Verify A is disassociated, B is associated
   - Verify sequential operation (A disassociated before B associated)

5. **TestAccServerResource_FloatingIP_Multiple**
   - Create server with 2 network attachments
   - Associate different floating IPs to each
   - Verify both associations work independently

6. **TestAccServerResource_FloatingIP_InvalidUUID**
   - Attempt to create server with invalid floating_ip_id format
   - Verify validation error

7. **TestAccServerResource_FloatingIP_AlreadyInUse**
   - Attempt to associate floating IP that's already associated elsewhere
   - Verify clear error message

8. **TestAccServerResource_FloatingIP_Import**
   - Create server with floating IP via API
   - Import into Terraform
   - Verify floating_ip_id and floating_ip are populated correctly

**Test Execution**:
```bash
make testacc TESTARGS='-run=TestAccServerResource_FloatingIP' PARALLEL=1
```

**Decision**: Write all tests FIRST following TDD red-green-refactor cycle. Use PreCheck to verify floating IPs are available in test environment. Use CheckFunctions to validate all state attributes.

**Rationale**: TDD ensures behavior is specified before implementation. Comprehensive test coverage validates all user stories and edge cases from the spec. Constitution requires tests before code.

---

## Summary

### Key Technical Decisions

1. **SDK Integration**: Use FloatingIP Associate/Disassociate methods with port_id mapping
2. **Schema Design**: floating_ip_id (Optional) + floating_ip (Computed) in NetworkAttachmentModel
3. **State Management**: Explicit handling of null/value transitions for associate/disassociate/swap
4. **Synchronous Operations**: Polling with 2s intervals, 30s/15s timeouts per success criteria
5. **Error Handling**: Fail-fast with actionable messages, pre-validation before SDK calls
6. **Import**: Automatic via Read method, no special handling needed
7. **Testing**: 8 comprehensive acceptance tests written BEFORE implementation

### Unknowns Resolved

- ✅ SDK API methods for floating IP association
- ✅ Schema attribute types and validation
- ✅ State diff handling for updates
- ✅ Polling implementation pattern
- ✅ Error handling approach
- ✅ Import behavior
- ✅ Test coverage requirements

### Next Phase

Proceed to Phase 1: Generate data-model.md with detailed Go struct definitions and SDK mappings.
