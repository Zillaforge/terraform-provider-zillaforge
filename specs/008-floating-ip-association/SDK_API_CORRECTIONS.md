# SDK API Corrections

**Date**: 2025-12-25  
**Feature**: 008-floating-ip-association

## Critical API Corrections from User

### 1. Floating IP Association API

**INCORRECT ASSUMPTION**:
```go
vpsClient.FloatingIPs().AssociateFloatingIP(ctx, floatingIPID, AssociateRequest{PortID})
```

**ACTUAL API**:
```go
// Association is via server NIC operations, not floating IP client
server, err := serversClient.Get(ctx, serverID)
req := servermodels.ServerNICAssociateFloatingIPRequest{
    FloatingIPID: floatingIPID,
}
err = server.NICs().AssociateFloatingIP(ctx, nicID, req)
```

**Impact**: 
- Association requires `nicID` (from server resource), not `portID`
- Must call via `server.NICs()` operations, not `FloatingIPs()` client
- Mapping: `network_id` → NIC ID from `server.NICs`

---

### 2. Floating IP Disassociation API

**INCORRECT ASSUMPTION**:
```go
vpsClient.FloatingIPs().DisassociateFloatingIP(ctx, floatingIPID)
```

**ACTUAL API**:
```go
// Disassociation is via Delete method (does not delete resource)
err := vpsClient.FloatingIPs().Delete(ctx, floatingIPID)
```

**Impact**:
- Delete method disassociates floating IP from server
- Does NOT delete the floating IP resource itself
- Floating IP remains available for re-association

---

### 3. SDK Waiter Helpers

**INCORRECT ASSUMPTION**:
Manual polling with ticker loops

**ACTUAL SDK FEATURE**:
```go
// cloud-sdk already provides waiter helper for floating IPs
import vpscore "github.com/Zillaforge/cloud-sdk/modules/vps/core"

// Wait for association
err := vpscore.WaitForFloatingIPStatus(ctx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusActive,
    TargetDeviceID: serverID,
})

// Wait for disassociation
err := vpscore.WaitForFloatingIPStatus(ctx, vpscore.FloatingIPWaiterConfig{
    Client:         floatingIPClient,
    FloatingIPID:   floatingIPID,
    TargetStatus:   floatingipmodels.FloatingIPStatusDown,
    TargetDeviceID: "",
})
```

**Impact**:
- Use SDK waiter pattern like `WaitForServerStatus` in `internal/vps/helper/server.go`
- SDK handles polling intervals and retries internally
- Consistent with existing codebase patterns

---

### 4. Server ACTIVE Requirement

**CRITICAL CONSTRAINT**:
> If `floating_ip_id` is present in network_attachment within server block, floating_ip must start associate call **after server become ready**. Otherwise, because the corresponding nic is not ready, will result in fip associate fail.

**Implementation**:
```go
// STEP 1: Create server
serverResp, err := serversClient.Create(ctx, createReq)

// STEP 2: Wait for server ACTIVE (required before NIC operations)
waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
err = vpscore.WaitForServerStatus(waitCtx, vpscore.ServerWaiterConfig{
    Client:       serversClient,
    ServerID:     serverResp.ID,
    TargetStatus: servermodels.ServerStatusActive,
})

// STEP 3: Now safe to associate floating IPs
server, err := serversClient.Get(ctx, serverResp.ID)
// ... associate floating IPs via server.NICs()
```

**Impact**:
- Server must be ACTIVE before ANY floating IP association attempts
- NICs are not available during BUILD status
- Add explicit wait in `AssociateFloatingIPsForServer` helper

---

## Updated Documentation Files

All planning documents have been updated with correct SDK API usage:

1. **[contracts/floating-ip-association-api.md](contracts/floating-ip-association-api.md)**: Updated SDK method signatures, operation sequences, polling strategy
2. **[research.md](research.md)**: Updated Section 1 (SDK API Interface) and Section 4 (Polling Implementation) with correct methods
3. **[data-model.md](data-model.md)**: Updated helper function implementations (`AssociateFloatingIPsForServer`, `UpdateFloatingIPAssociations`)

---

## Existing Codebase Reference

The correct pattern already exists in [`internal/vps/helper/server.go`](../../internal/vps/helper/server.go):

```go
// WaitForServerActive waits for the server to reach "active" using the SDK-provided waiter helper.
func WaitForServerActive(ctx context.Context, serversClient *serversdk.Client, serverID string, timeout time.Duration) (*serversdk.ServerResource, error) {
    waitCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    if err := vpscore.WaitForServerStatus(waitCtx, vpscore.ServerWaiterConfig{
        Client:       serversClient,
        ServerID:     serverID,
        TargetStatus: servermodels.ServerStatusActive,
    }); err != nil {
        return nil, fmt.Errorf("waiting for server to become active: %w", err)
    }

    // After waiter completes, fetch the latest server resource
    return serversClient.Get(ctx, serverID)
}
```

Follow this exact pattern for floating IP waiters.

---

## Implementation Checklist

- [ ] Wait for server ACTIVE before floating IP association (5-minute timeout)
- [ ] Use `server.NICs().AssociateFloatingIP(ctx, nicID, req)` for association
- [ ] Use `vpsClient.FloatingIPs().Delete(ctx, floatingIPID)` for disassociation
- [ ] Use `vpscore.WaitForFloatingIPStatus()` with appropriate config for polling
- [ ] Map `network_id` to NIC ID via `server.NICs` (not port_id)
- [ ] Create waiter helpers in `internal/vps/helper/server.go` following existing pattern
- [ ] Test association fails gracefully if server not ACTIVE
- [ ] Test disassociation via Delete doesn't delete floating IP resource

---

## Next Steps

Proceed with implementation using corrected SDK API:
1. Run `/speckit.tasks` to generate task breakdown
2. Write acceptance tests FIRST (TDD)
3. Implement helpers following `WaitForServerActive` pattern
4. Test all scenarios including server state transitions
