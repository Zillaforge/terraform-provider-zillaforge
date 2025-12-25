# API Contract: Floating IP Association

**Feature**: 008-floating-ip-association  
**Date**: 2025-12-25

## Overview

This document specifies the cloud-sdk API methods used for associating and disassociating floating IPs with server network ports in the ZillaForge Terraform provider.

---

## Base URL & Authentication

```
Base URL: https://api.zillaforge.com/v1
Authentication: Bearer token (JWT) in Authorization header
Content-Type: application/json
```

---

## API Endpoints

### 1. Associate Floating IP with Server Port

**Purpose**: Associate a floating IP with a specific network port on a server.

**Method**: `POST /vps/floating-ips/{floating_ip_id}/associate`

**Path Parameters**:
- `floating_ip_id` (string, required): UUID of the floating IP to associate

**Request Body**:
```json
{
  "port_id": "port-uuid-here"
}
```

**Request Fields**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `port_id` | string (UUID) | Yes | UUID of the network port to associate with |

**Success Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "web-public-ip",
  "description": "Public IP for web server",
  "address": "203.0.113.10",
  "status": "ACTIVE",
  "device_id": "server-uuid-here"
}
```

**Response Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `id` | string (UUID) | Floating IP UUID |
| `name` | string | Floating IP name (optional) |
| `description` | string | Floating IP description (optional) |
| `address` | string (IP) | Public IP address |
| `status` | string (enum) | Status: "ACTIVE", "DOWN", "PENDING", "REJECTED" |
| `device_id` | string (UUID) | Server UUID the floating IP is associated with |

**Error Responses**:

**404 Not Found** - Floating IP doesn't exist:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Floating IP 550e8400-e29b-41d4-a716-446655440000 not found"
  }
}
```

**409 Conflict** - Floating IP already in use:
```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Floating IP is already associated with device abc-123-def-456"
  }
}
```

**400 Bad Request** - Invalid port ID:
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Port port-uuid-here not found or not accessible"
  }
}
```

**422 Unprocessable Entity** - Server not in valid state:
```json
{
  "error": {
    "code": "INVALID_STATE",
    "message": "Cannot associate floating IP when server is in BUILD status"
  }
}
```

---

### 2. Disassociate Floating IP

**Purpose**: Remove floating IP association from a server port.

**Method**: `POST /vps/floating-ips/{floating_ip_id}/disassociate`

**Path Parameters**:
- `floating_ip_id` (string, required): UUID of the floating IP to disassociate

**Request Body**: Empty (no body required)

**Success Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "web-public-ip",
  "description": "Public IP for web server",
  "address": "203.0.113.10",
  "status": "DOWN",
  "device_id": ""
}
```

**Response Fields**: Same as Associate endpoint, but `device_id` is empty string and `status` is typically "DOWN".

**Error Responses**:

**404 Not Found** - Floating IP doesn't exist:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Floating IP 550e8400-e29b-41d4-a716-446655440000 not found"
  }
}
```

**Note**: Disassociating an unassociated floating IP is idempotent and returns 200 OK.

---

### 3. Get Floating IP Details

**Purpose**: Retrieve current state of a floating IP, including association status.

**Method**: `GET /vps/floating-ips/{floating_ip_id}`

**Path Parameters**:
- `floating_ip_id` (string, required): UUID of the floating IP

**Success Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "web-public-ip",
  "description": "Public IP for web server",
  "address": "203.0.113.10",
  "status": "ACTIVE",
  "device_id": "server-uuid-here"
}
```

**Usage**: Used to:
- Verify floating IP exists before association
- Check if floating IP is available (device_id empty)
- Poll for association/disassociation completion

**Error Responses**:

**404 Not Found**:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Floating IP 550e8400-e29b-41d4-a716-446655440000 not found"
  }
}
```

---

### 4. Get Server Details (Existing)

**Purpose**: Retrieve server details including network ports (needed for port_id mapping).

**Method**: `GET /vps/servers/{server_id}`

**Path Parameters**:
- `server_id` (string, required): UUID of the server

**Success Response** (200 OK):
```json
{
  "id": "server-uuid-here",
  "name": "web-server-01",
  "description": "Production web server",
  "flavor_id": "flavor-uuid",
  "image_id": "image-uuid",
  "status": "ACTIVE",
  "network_ports": [
    {
      "port_id": "port-uuid-here",
      "network_id": "network-uuid",
      "ip_address": "10.0.0.10",
      "is_primary": true,
      "security_group_ids": ["sg-uuid-1", "sg-uuid-2"]
    }
  ],
  "keypair_id": "keypair-uuid",
  "ip_addresses": ["10.0.0.10", "203.0.113.10"],
  "created_at": "2025-12-25T10:00:00Z"
}
```

**Relevant Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `network_ports` | array | List of network port attachments |
| `network_ports[].port_id` | string (UUID) | Port UUID (used for floating IP association) |
| `network_ports[].network_id` | string (UUID) | Network UUID (matches Terraform network_attachment) |
| `network_ports[].ip_address` | string (IP) | Internal IP address |
| `network_ports[].is_primary` | boolean | Primary interface flag |
| `ip_addresses` | array of strings | All IP addresses (internal + floating) |
| `status` | string (enum) | Server status: "BUILDING", "ACTIVE", "ERROR", "DELETED" |

---

## Operation Sequence

### Associate Floating IP During Server Creation

```
1. POST /vps/servers
   Request: { name, flavor_id, image_id, network_ports, ... }
   Response: { id: "server-uuid", status: "BUILDING", ... }

2. Wait for Server Active (SDK Waiter):
   vpscore.WaitForServerStatus(ctx, ServerWaiterConfig{
       Client: serversClient,
       ServerID: serverID,
       TargetStatus: servermodels.ServerStatusActive,
   })
   Max wait: 5 minutes (default from spec 006)

3. GET /vps/servers/{server_id}
   Extract ServerResource for NIC operations
   Get NIC ID for target network from server.NICs

4. GET /vps/floating-ips/{floating_ip_id}
   Verify floating IP exists and device_id is empty

5. Associate via Server NIC:
   server.NICs().AssociateFloatingIP(ctx, nicID, ServerNICAssociateFloatingIPRequest{
       FloatingIPID: "floating-ip-uuid",
   })

6. Wait for Floating IP Active (SDK Waiter):
   vpscore.WaitForFloatingIPStatus(ctx, FloatingIPWaiterConfig{
       Client: floatingIPClient,
       FloatingIPID: floatingIPID,
       TargetStatus: floatingipmodels.FloatingIPStatusActive,
       TargetDeviceID: serverID,
   })
   Max wait: 30 seconds

7. GET /vps/servers/{server_id}
   Final read to populate Terraform state
```

### Disassociate Floating IP

```
1. Disassociate via FloatingIP Delete:
   vpsClient.FloatingIPs().Delete(ctx, floatingIPID)
   Note: Delete disassociates the floating IP from server
   (does not actually delete the floating IP resource)

2. Wait for Floating IP Disassociated (SDK Waiter):
   vpscore.WaitForFloatingIPStatus(ctx, FloatingIPWaiterConfig{
       Client: floatingIPClient,
       FloatingIPID: floatingIPID,
       TargetStatus: floatingipmodels.FloatingIPStatusDown,
       TargetDeviceID: "", // Wait for empty device_id
   })
   Max wait: 15 seconds

3. GET /vps/servers/{server_id}
   Final read to populate Terraform state
```

### Swap Floating IPs (Sequential)

```
# Disassociate old floating IP
1. vpsClient.FloatingIPs().Delete(ctx, oldFloatingIPID)
2. vpscore.WaitForFloatingIPStatus(ctx, FloatingIPWaiterConfig{
       Client: floatingIPClient,
       FloatingIPID: oldFloatingIPID,
       TargetStatus: floatingipmodels.FloatingIPStatusDown,
       TargetDeviceID: "",
   })

# Associate new floating IP
3. GET /vps/floating-ips/{new_floating_ip_id}
   Verify availability
4. server.NICs().AssociateFloatingIP(ctx, nicID, ServerNICAssociateFloatingIPRequest{
       FloatingIPID: newFloatingIPID,
   })
5. vpscore.WaitForFloatingIPStatus(ctx, FloatingIPWaiterConfig{
       Client: floatingIPClient,
       FloatingIPID: newFloatingIPID,
       TargetStatus: floatingipmodels.FloatingIPStatusActive,
       TargetDeviceID: serverID,
   })

6. GET /vps/servers/{server_id}
   Final read to populate Terraform state
```

---

## Polling Strategy

### Use SDK Waiter Helpers

The cloud-sdk provides built-in waiter helpers similar to server waiters. Use `vpscore.WaitForFloatingIPStatus` instead of manual polling.

### Association Polling

```go
waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

err := vpscore.WaitForFloatingIPStatus(waitCtx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusActive,
    TargetDeviceID: serverID, // Wait for association to specific server
})
```

### Disassociation Polling

```go
waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
defer cancel()

err := vpscore.WaitForFloatingIPStatus(waitCtx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusDown,
    TargetDeviceID: "", // Wait for empty device_id (disassociated)
})
```

---

## Error Handling

### SDK Client Error Handling

```go
// No automatic retries - fail immediately per clarification
req := servermodels.ServerNICAssociateFloatingIPRequest{
    FloatingIPID: floatingIPID,
}
if err := server.NICs().AssociateFloatingIP(ctx, nicID, req); err != nil {
    switch {
    case strings.Contains(err.Error(), "404"):
        return fmt.Errorf("floating IP %s not found or NIC %s not found", floatingIPID, nicID)
    case strings.Contains(err.Error(), "409"):
        return fmt.Errorf("floating IP %s is already in use", floatingIPID)
    case strings.Contains(err.Error(), "422"):
        return fmt.Errorf("server is not in a valid state for floating IP association (must be ACTIVE)")
    default:
        return fmt.Errorf("failed to associate floating IP: %w", err)
    }
}
```

### Timeout Handling

```go
// Association timeout (using SDK waiter)
waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

if err := vpscore.WaitForFloatingIPStatus(waitCtx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusActive,
    TargetDeviceID: serverID,
}); err != nil {
    return fmt.Errorf("floating IP association did not complete within 30 seconds: %w", err)
}

// Disassociation timeout (using SDK waiter)
waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
defer cancel()

if err := vpscore.WaitForFloatingIPStatus(waitCtx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusDown,
    TargetDeviceID: "",
}); err != nil {
    return fmt.Errorf("floating IP disassociation did not complete within 15 seconds: %w", err)
}
```

---

## SDK Method Signatures (Actual)

```go
package servers

import (
    "context"
    servermodels "github.com/Zillaforge/cloud-sdk/models/vps/servers"
    floatingipmodels "github.com/Zillaforge/cloud-sdk/models/vps/floatingips"
)

// ServerResource provides server operations including NIC management
type ServerResource struct {
    // ... fields ...
}

// NICs returns a NIC operations client for this server
func (s *ServerResource) NICs() *NICOperations

// NICOperations provides network interface operations
type NICOperations struct {
    // ... fields ...
}

// AssociateFloatingIP associates a floating IP with a specific NIC
// IMPORTANT: Server must be in ACTIVE status before calling
func (n *NICOperations) AssociateFloatingIP(ctx context.Context, nicID string, req servermodels.ServerNICAssociateFloatingIPRequest) error

// ServerNICAssociateFloatingIPRequest for associating floating IP with a NIC
type ServerNICAssociateFloatingIPRequest struct {
    FloatingIPID string `json:"floating_ip_id"`
}
```

```go
package floatingips

import (
    "context"
    floatingipmodels "github.com/Zillaforge/cloud-sdk/models/vps/floatingips"
)

// Client provides floating IP operations
type Client struct {
    // ... fields ...
}

// Get retrieves floating IP details
func (c *Client) Get(ctx context.Context, id string) (*floatingipmodels.FloatingIP, error)

// Delete disassociates and optionally deletes a floating IP
// When called, it disassociates the floating IP from any server
func (c *Client) Delete(ctx context.Context, floatingIPID string) error

// FloatingIP represents a floating IP resource
type FloatingIP struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Address     string `json:"address"`
    Status      string `json:"status"` // "ACTIVE", "DOWN", "PENDING", "REJECTED"
    DeviceID    string `json:"device_id"` // Server UUID or empty string
}
```

```go
package vpscore

import (
    "context"
    floatingipmodels "github.com/Zillaforge/cloud-sdk/models/vps/floatingips"
    floatingipsdk "github.com/Zillaforge/cloud-sdk/modules/vps/floatingips"
)

// FloatingIPWaiterConfig configures floating IP status polling
type FloatingIPWaiterConfig struct {
    Client         *floatingipsdk.Client
    FloatingIPID   string
    TargetStatus   floatingipmodels.FloatingIPStatus
    TargetDeviceID string // Optional: wait until associated with specific server
}

// WaitForFloatingIPStatus polls until floating IP reaches target status
func WaitForFloatingIPStatus(ctx context.Context, config FloatingIPWaiterConfig) error
```

---

## Summary

### Key SDK Interactions

1. **Associate**: `server.NICs().AssociateFloatingIP(ctx, nicID, ServerNICAssociateFloatingIPRequest{FloatingIPID})` - Must wait for server ACTIVE first
2. **Disassociate**: `vpsClient.FloatingIPs().Delete(ctx, floatingIPID)` - Disassociates from server
3. **Wait for Association**: `vpscore.WaitForFloatingIPStatus()` with `TargetDeviceID`
4. **Wait for Disassociation**: `vpscore.WaitForFloatingIPStatus()` with empty `TargetDeviceID`
5. **Get Server**: `serversClient.Get(ctx, serverID)` to get ServerResource with NICs

### Constraints

- **CRITICAL**: Server must be ACTIVE before floating IP association (NICs not ready until server active)
- Association via `server.NICs().AssociateFloatingIP()` - requires ServerResource and NIC ID
- Disassociation via `vpsClient.FloatingIPs().Delete()` - does not delete the resource, only disassociates
- NIC ID must exist on server (map from network_id via server.NICs)
- Floating IP must not be associated elsewhere
- No automatic retries - fail immediately on errors
- Use SDK waiter helpers (`vpscore.WaitForFloatingIPStatus`) with timeouts (30s associate, 15s disassociate)
- Sequential swap: disassociate completes before associate begins

### Next Phase

Implementation ready. All API contracts, data models, and logic flows are documented. Proceed to test-driven implementation following TDD red-green-refactor cycle.
