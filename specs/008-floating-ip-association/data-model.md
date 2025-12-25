# Data Model: Floating IP Association with Network Attachments

**Feature**: 008-floating-ip-association  
**Date**: 2025-12-25

## Overview

This document defines the data structures required to extend the `zillaforge_server` resource with floating IP association capabilities. The changes add two new attributes to the `NetworkAttachmentModel`: `floating_ip_id` (user-provided UUID) and `floating_ip` (computed IP address).

---

## Modified Terraform Models

### NetworkAttachmentModel (MODIFIED)

**File**: `internal/vps/model/server.go`

```go
// NetworkAttachmentModel represents a network interface attachment.
type NetworkAttachmentModel struct {
    NetworkID        types.String `tfsdk:"network_id"`
    IPAddress        types.String `tfsdk:"ip_address"`
    Primary          types.Bool   `tfsdk:"primary"`
    SecurityGroupIDs types.List   `tfsdk:"security_group_ids"` // List of types.String
    
    // NEW: Floating IP association
    FloatingIPID     types.String `tfsdk:"floating_ip_id"` // Optional: UUID of floating IP to associate
    FloatingIP       types.String `tfsdk:"floating_ip"`    // Computed: Actual IP address of associated floating IP
}
```

**Attribute Details**:

| Attribute | Type | Required/Optional/Computed | Description |
|-----------|------|----------------------------|-------------|
| `network_id` | types.String | Required | Network ID (existing) |
| `ip_address` | types.String | Optional | Fixed IP address (existing) |
| `primary` | types.Bool | Optional | Primary interface flag (existing) |
| `security_group_ids` | types.List | Optional | List of security group IDs (existing) |
| `floating_ip_id` | types.String | **Optional** (NEW) | UUID of floating IP to associate with this network interface |
| `floating_ip` | types.String | **Computed** (NEW) | The public IP address of the associated floating IP (read-only) |

**Validation Rules**:
- `floating_ip_id`: Must be valid UUID format if specified (regex: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
- `floating_ip`: Cannot be set by user, populated from SDK response
- When `floating_ip_id` is null/empty, `floating_ip` should also be null/empty

---

## Schema Definition (MODIFIED)

**File**: `internal/vps/resource/server_resource.go`

```go
// In Schema() method, modify the network_attachment nested attribute
"network_attachment": schema.ListNestedAttribute{
    MarkdownDescription: "List of network interfaces to attach to the server. At least one network attachment is required. Each attachment represents a NIC with its network, IP, and security groups.",
    Required: true,
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            "network_id": schema.StringAttribute{
                MarkdownDescription: "The ID of the network to attach.",
                Required: true,
            },
            "ip_address": schema.StringAttribute{
                MarkdownDescription: "Fixed IP address to assign. Leave empty for DHCP.",
                Optional: true,
            },
            "primary": schema.BoolAttribute{
                MarkdownDescription: "Whether this is the primary network interface. At most one attachment can be primary.",
                Optional: true,
            },
            "security_group_ids": schema.ListAttribute{
                MarkdownDescription: "List of security group IDs to apply to this network interface.",
                Optional: true,
                ElementType: types.StringType,
            },
            // NEW ATTRIBUTES
            "floating_ip_id": schema.StringAttribute{
                MarkdownDescription: "UUID of the floating IP to associate with this network interface. " +
                    "When specified, the floating IP will be associated with this network attachment. " +
                    "Remove this attribute or set to null to disassociate the floating IP. " +
                    "Note: The floating IP must exist and not be associated with another server.",
                Optional: true,
                Validators: []validator.String{
                    stringvalidator.RegexMatches(
                        regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
                        "must be a valid UUID format",
                    ),
                },
            },
            "floating_ip": schema.StringAttribute{
                MarkdownDescription: "The public IP address of the floating IP associated with this network interface. " +
                    "This is a read-only attribute that displays the IP address corresponding to floating_ip_id. " +
                    "Empty when no floating IP is associated.",
                Computed: true,
            },
        },
    },
},
```

---

## SDK Integration

### Cloud-SDK Floating IP Client (Expected Interface)

**Package**: `github.com/Zillaforge/cloud-sdk/clients/vps`

```go
// VPSClient provides methods for VPS operations including floating IPs
type VPSClient interface {
    // Existing methods...
    GetServer(ctx context.Context, id string) (*servermodels.Server, error)
    CreateServer(ctx context.Context, req *servermodels.CreateRequest) (*servermodels.Server, error)
    UpdateServer(ctx context.Context, id string, req *servermodels.UpdateRequest) (*servermodels.Server, error)
    DeleteServer(ctx context.Context, id string) error
    
    // Floating IP methods
    GetFloatingIP(ctx context.Context, id string) (*floatingipmodels.FloatingIP, error)
    AssociateFloatingIP(ctx context.Context, floatingIPID string, req *floatingipmodels.AssociateRequest) error
    DisassociateFloatingIP(ctx context.Context, floatingIPID string) error
}
```

### Floating IP Association Request

**Package**: `github.com/Zillaforge/cloud-sdk/models/vps/servers`

```go
// ServerNICAssociateFloatingIPRequest for associating floating IP with a NIC
type ServerNICAssociateFloatingIPRequest struct {
    FloatingIPID string `json:"floating_ip_id"`
}

// Note: Association is via server.NICs().AssociateFloatingIP(nicID, request)
// We map network_attachment.network_id to server.NICs[i].ID to find the correct NIC ID
// See SDK_API_CORRECTIONS.md for authoritative API documentation
```

### Server NIC Structure (Existing)

**Package**: `github.com/Zillaforge/cloud-sdk/models/vps/servers`

```go
// ServerNIC represents a network interface on a server (from SDK)
type ServerNIC struct {
    ID               string   `json:"id"`          // NIC ID (used for floating IP association)
    NetworkID        string   `json:"network_id"`  // User-specified network
    Addresses        []string `json:"addresses"`   // Assigned IP addresses
    SGIDs            []string `json:"sg_ids,omitempty"` // Security group IDs
}

// Floating IP association information comes from FloatingIP.DeviceID and GetFloatingIP calls
// We need to query each floating IP to get its Address and match it to DeviceID = serverID
// See SDK_API_CORRECTIONS.md for authoritative SDK method signatures
```

---

## Mapping Logic

### 1. Terraform → SDK (Create with Floating IP)

**File**: `internal/vps/helper/server.go` (NEW FUNCTION)

```go
// AssociateFloatingIPsForServer associates floating IPs after server creation
func AssociateFloatingIPsForServer(
    ctx context.Context,
    serversClient *serversdk.Client,
    floatingIPClient *floatingipsdk.Client,
    serverID string,
    attachments []model.NetworkAttachmentModel,
) diag.Diagnostics {
    var diags diag.Diagnostics
    
    // Get server resource for NIC operations
    server, err := serversClient.Get(ctx, serverID)
    if err != nil {
        diags.AddError("Failed to Get Server", err.Error())
        return diags
    }
    
    // Map network attachments to NICs by network_id
    nicMap := make(map[string]string) // network_id -> NIC ID
    nics, err := server.NICs().List(ctx)
    if err != nil {
        diags.AddError("Failed to List NICs", err.Error())
        return diags
    }
    for _, nic := range nics {
        nicMap[nic.NetworkID] = nic.ID
    }
    
    // Associate floating IPs for each attachment
    for _, attachment := range attachments {
        if attachment.FloatingIPID.IsNull() || attachment.FloatingIPID.IsUnknown() {
            continue // No floating IP to associate
        }
        
        floatingIPID := attachment.FloatingIPID.ValueString()
        networkID := attachment.NetworkID.ValueString()
        nicID, ok := nicMap[networkID]
        if !ok {
            diags.AddError(
                "NIC Not Found",
                fmt.Sprintf("Could not find NIC for network %s", networkID),
            )
            continue
        }
        
        // Validate floating IP exists and is available
        fip, err := client.GetFloatingIP(ctx, floatingIPID)
        if err != nil {
            diags.AddError(
                "Floating IP Not Found",
                fmt.Sprintf("Could not find floating IP %s: %s", floatingIPID, err.Error()),
            )
            continue
        }
        
        if fip.DeviceID != "" {
            diags.AddError(
                "Floating IP Already In Use",
                fmt.Sprintf("Floating IP %s is already associated with device %s", floatingIPID, fip.DeviceID),
            )
            continue
        }
        
        // Associate the floating IP via server NIC
        req := servermodels.ServerNICAssociateFloatingIPRequest{
            FloatingIPID: floatingIPID,
        }
        
        if err := server.NICs().AssociateFloatingIP(ctx, nicID, req); err != nil {
            diags.AddError(
                "Failed to Associate Floating IP",
                fmt.Sprintf("Could not associate floating IP %s with NIC %s: %s", floatingIPID, nicID, err.Error()),
            )
            continue
        }
        
        // Wait for association to complete
        if err := waitForFloatingIPAssociated(ctx, client, floatingIPID, serverID, 30*time.Second); err != nil {
            diags.AddError(
                "Floating IP Association Timeout",
                fmt.Sprintf("Floating IP %s did not complete association in time: %s", floatingIPID, err.Error()),
            )
        }
    }
    
    return diags
}

// waitForFloatingIPAssociated polls until floating IP is associated
func waitForFloatingIPAssociated(
    ctx context.Context,
    client *vps.Client,
    floatingIPID string,
    expectedDeviceID string,
    timeout time.Duration,
) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for floating IP association")
        case <-ticker.C:
            fip, err := client.GetFloatingIP(ctx, floatingIPID)
            if err != nil {
                return err
            }
            if fip.DeviceID == expectedDeviceID {
                return nil
            }
        }
    }
}
```

### 2. SDK → Terraform (Read)

**File**: `internal/vps/helper/server.go` (MODIFY EXISTING)

```go
// MapServerToResourceModel (MODIFY to populate floating IP info)
func MapServerToResourceModel(
    ctx context.Context,
    client *vps.Client,
    server *servermodels.Server,
    data *model.ServerResourceModel,
) diag.Diagnostics {
    var diags diag.Diagnostics
    
    // ... existing mappings for id, name, status, etc. ...
    
    // Map network ports to network_attachment with floating IP info
    attachmentModels := make([]model.NetworkAttachmentModel, len(server.NetworkPorts))
    for i, port := range server.NetworkPorts {
        // Extract security group IDs
        sgIDs := make([]attr.Value, len(port.SecurityGroupIDs))
        for j, sg := range port.SecurityGroupIDs {
            sgIDs[j] = types.StringValue(sg)
        }
        sgList, diagList := types.ListValue(types.StringType, sgIDs)
        diags.Append(diagList...)
        
        // Find associated floating IP for this port
        floatingIPID, floatingIPAddress := findFloatingIPForPort(ctx, client, server.ID, port.PortID)
        
        attachmentModels[i] = model.NetworkAttachmentModel{
            NetworkID:        types.StringValue(port.NetworkID),
            IPAddress:        types.StringValue(port.IPAddress),
            Primary:          types.BoolValue(port.IsPrimary),
            SecurityGroupIDs: sgList,
            FloatingIPID:     types.StringPointerValue(floatingIPID),     // NEW
            FloatingIP:       types.StringPointerValue(floatingIPAddress), // NEW
        }
    }
    
    // Convert to types.List
    attachmentList, diagList := types.ListValueFrom(ctx, 
        types.ObjectType{AttrTypes: networkAttachmentAttrTypes()}, 
        attachmentModels)
    diags.Append(diagList...)
    data.NetworkAttachment = attachmentList
    
    return diags
}

// findFloatingIPForNIC queries all floating IPs to find one associated with this server/NIC
func findFloatingIPForNIC(
    ctx context.Context,
    floatingIPClient *floatingipsdk.Client,
    serverID string,
    nicID string,
) (*string, *string) {
    // Query floating IPs associated with this server
    // Note: This may require listing all floating IPs and filtering by device_id
    // The SDK may provide a query method to filter by server
    
    // Simplified approach: List all floating IPs, filter by device_id == serverID
    fips, err := client.ListFloatingIPs(ctx, nil)
    if err != nil {
        return nil, nil
    }
    
    for _, fip := range fips {
        if fip.DeviceID == serverID {
            // Found a floating IP for this server
            // Note: We may need additional port_id matching if SDK provides it
            return &fip.ID, &fip.Address
        }
    }
    
    return nil, nil // No floating IP found
}
```

### 3. Terraform → SDK (Update - Swap/Associate/Disassociate)

**File**: `internal/vps/helper/server.go` (NEW FUNCTION)

```go
// UpdateFloatingIPAssociations handles floating IP changes during server update
func UpdateFloatingIPAssociations(
    ctx context.Context,
    serversClient *serversdk.Client,
    floatingIPClient *floatingipsdk.Client,
    serverID string,
    planAttachments []model.NetworkAttachmentModel,
    stateAttachments []model.NetworkAttachmentModel,
) diag.Diagnostics {
    var diags diag.Diagnostics
    
    // Get server resource for NIC operations
    server, err := serversClient.Get(ctx, serverID)
    if err != nil {
        diags.AddError("Failed to Get Server", err.Error())
        return diags
    }
    
    nicMap := make(map[string]string) // network_id -> NIC ID
    nics, err := server.NICs().List(ctx)
    if err != nil {
        diags.AddError("Failed to List NICs", err.Error())
        return diags
    }
    for _, nic := range nics {
        nicMap[nic.NetworkID] = nic.ID
    }
    
    // Compare plan vs state for each attachment
    for i := range planAttachments {
        if i >= len(stateAttachments) {
            // New attachment added (handle in network attachment update logic)
            continue
        }
        
        planFIPID := planAttachments[i].FloatingIPID
        stateFIPID := stateAttachments[i].FloatingIPID
        
        // Skip if no change
        if planFIPID.Equal(stateFIPID) {
            continue
        }
        
        networkID := planAttachments[i].NetworkID.ValueString()
        
        // Get server resource for NIC operations
        server, err := serversClient.Get(ctx, serverID)
        if err != nil {
            diags.AddError("Failed to Get Server", fmt.Sprintf("Could not retrieve server %s: %s", serverID, err.Error()))
            continue
        }
        
        // Find NIC ID for network
        var nicID string
        for _, nic := range server.NICs {
            if nic.NetworkID == networkID {
                nicID = nic.ID
                break
            }
        }
        if nicID == "" {
            diags.AddError("NIC Not Found", fmt.Sprintf("Could not find NIC for network %s", networkID))
            continue
        }
        
        // Disassociate old floating IP if present (via Delete)
        if !stateFIPID.IsNull() && !stateFIPID.IsUnknown() {
            oldFIPID := stateFIPID.ValueString()
            if err := floatingIPClient.Delete(ctx, oldFIPID); err != nil {
                diags.AddError(
                    "Failed to Disassociate Floating IP",
                    fmt.Sprintf("Could not disassociate floating IP %s: %s", oldFIPID, err.Error()),
                )
                continue
            }
            
            // Wait for disassociation using SDK waiter
            if _, err := WaitForFloatingIPDisassociated(ctx, floatingIPClient, oldFIPID, 15*time.Second); err != nil {
                diags.AddError("Disassociation Timeout", err.Error())
                continue
            }
        }
        
        // Associate new floating IP if present (via server NIC)
        if !planFIPID.IsNull() && !planFIPID.IsUnknown() {
            newFIPID := planFIPID.ValueString()
            
            // Validate availability
            fip, err := floatingIPClient.Get(ctx, newFIPID)
            if err != nil {
                diags.AddError("Floating IP Not Found", err.Error())
                continue
            }
            if fip.DeviceID != "" {
                diags.AddError(
                    "Floating IP Already In Use",
                    fmt.Sprintf("Floating IP %s is already associated with device %s", newFIPID, fip.DeviceID),
                )
                continue
            }
            
            // Associate via server NIC
            req := servermodels.ServerNICAssociateFloatingIPRequest{FloatingIPID: newFIPID}
            if err := server.NICs().AssociateFloatingIP(ctx, nicID, req); err != nil {
                diags.AddError("Failed to Associate Floating IP", err.Error())
                continue
            }
            
            // Wait for association using SDK waiter
            if _, err := WaitForFloatingIPAssociated(ctx, floatingIPClient, newFIPID, serverID, 30*time.Second); err != nil {
                diags.AddError("Association Timeout", err.Error())
            }
        }
    }
    
    return diags
}

// waitForFloatingIPDisassociated polls until floating IP device_id is empty
func waitForFloatingIPDisassociated(
    ctx context.Context,
    client *vps.Client,
    floatingIPID string,
    timeout time.Duration,
) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for floating IP disassociation")
        case <-ticker.C:
            fip, err := client.GetFloatingIP(ctx, floatingIPID)
            if err != nil {
                return err
            }
            if fip.DeviceID == "" {
                return nil
            }
        }
    }
}
```

---

## State Transitions

### Create

```
User Config: server with network_attachment { floating_ip_id = "uuid-123" }
↓
1. Create server (existing logic)
2. Call AssociateFloatingIPsForServer()
   - Get server details → extract port IDs
   - For each attachment with floating_ip_id:
     - Validate floating IP exists and available
     - Call AssociateFloatingIP(floating_ip_id, port_id)
     - Poll until DeviceID == serverID
3. Read server → populate floating_ip computed attribute
↓
Terraform State: network_attachment { 
  floating_ip_id = "uuid-123", 
  floating_ip = "203.0.113.10" 
}
```

### Update (Swap)

```
Current State: network_attachment { floating_ip_id = "uuid-123", floating_ip = "203.0.113.10" }
User Config:   network_attachment { floating_ip_id = "uuid-456", floating_ip = <unknown> }
↓
1. Call UpdateFloatingIPAssociations()
   - Detect change: uuid-123 → uuid-456
   - Disassociate uuid-123
     - Call DisassociateFloatingIP(uuid-123)
     - Poll until DeviceID == ""
   - Associate uuid-456
     - Validate availability
     - Call AssociateFloatingIP(uuid-456, port_id)
     - Poll until DeviceID == serverID
2. Read server → populate new floating_ip
↓
New State: network_attachment { 
  floating_ip_id = "uuid-456", 
  floating_ip = "203.0.113.20" 
}
```

### Update (Disassociate)

```
Current State: network_attachment { floating_ip_id = "uuid-123", floating_ip = "203.0.113.10" }
User Config:   network_attachment { floating_ip_id = null }
↓
1. Call UpdateFloatingIPAssociations()
   - Detect change: uuid-123 → null
   - Disassociate uuid-123
     - Call DisassociateFloatingIP(uuid-123)
     - Poll until DeviceID == ""
2. Read server → floating_ip becomes null
↓
New State: network_attachment { 
  floating_ip_id = null, 
  floating_ip = null 
}
```

### Delete

```
Current State: server with floating IP associations
↓
1. Call DisassociateFloatingIPsForServer() (before/during server delete)
   - For each attachment with floating_ip_id:
     - Call DisassociateFloatingIP(floating_ip_id)
2. Delete server (existing logic)
↓
State: (removed)
```

---

## Validation

### Schema-Level Validation

1. **UUID Format**: `floating_ip_id` must match UUID regex if specified
   ```go
   stringvalidator.RegexMatches(
       regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
       "must be a valid UUID format",
   )
   ```

### Runtime Validation (Pre-SDK Calls)

1. **Floating IP Exists**: Call `GetFloatingIP()` before association
2. **Floating IP Available**: Check `DeviceID == ""` before association
3. **Server Status**: Verify server is in ACTIVE status (per assumption)

### Error Messages

| Scenario | Error Message |
|----------|--------------|
| Invalid UUID format | "floating_ip_id must be a valid UUID format (e.g., 550e8400-e29b-41d4-a716-446655440000)" |
| Floating IP not found | "Floating IP {id} does not exist. Verify the floating IP ID is correct using data.zillaforge_floating_ips data source." |
| Already in use | "Floating IP {id} is already associated with server {device_id}. Disassociate it first or choose a different floating IP." |
| Server not ACTIVE | "Cannot associate floating IP while server is in {status} status. Wait for server to reach ACTIVE status." |
| Association timeout | "Floating IP association did not complete within 30 seconds. Check server and floating IP status in the ZillaForge console." |
| Disassociation timeout | "Floating IP disassociation did not complete within 15 seconds. Check floating IP status in the ZillaForge console." |

---

## Summary

### Changes Required

**Modified Files**:
- `internal/vps/model/server.go`: Add `FloatingIPID` and `FloatingIP` to `NetworkAttachmentModel`
- `internal/vps/resource/server_resource.go`: Update schema, Create, Read, Update, Delete methods
- `internal/vps/helper/server.go`: Add floating IP association/disassociation helper functions

**New Functions**:
- `AssociateFloatingIPsForServer()`: Associate floating IPs after server creation (via server.NICs().AssociateFloatingIP)
- `UpdateFloatingIPAssociations()`: Handle floating IP changes during update
- `DisassociateFloatingIPsForServer()`: Disassociate floating IPs before delete (via floatingIPClient.Delete)
- `WaitForFloatingIPAssociated()`: Use SDK waiter until association completes
- `WaitForFloatingIPDisassociated()`: Use SDK waiter until disassociation completes
- `findFloatingIPForNIC()`: Query floating IPs to populate Read state

**Authoritative SDK Reference**: See [SDK_API_CORRECTIONS.md](SDK_API_CORRECTIONS.md) for correct method signatures and patterns.

### Data Flow

1. **User → Terraform**: User specifies `floating_ip_id` in HCL
2. **Terraform → SDK**: Convert to `AssociateRequest` with `port_id`
3. **SDK → Cloud API**: HTTP requests to associate/disassociate
4. **Cloud API → SDK**: Response with updated floating IP state
5. **SDK → Terraform**: Map `DeviceID` and `Address` back to model
6. **Terraform → State**: Store `floating_ip_id` and `floating_ip` in state file

### Next Phase

Proceed to Phase 1 completion: Generate quickstart.md (user guide) and contracts/ (API specification).
