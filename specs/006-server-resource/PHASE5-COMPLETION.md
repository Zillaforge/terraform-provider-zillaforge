# Phase 5 (User Story 3) Completion Report

## 🎯 Goal: Asynchronous Server Creation

Enable users to create servers without waiting for active status by setting `wait_for_active = false` for faster batch deployments.

## ✅ Completed Tasks

### Acceptance Tests (T042-T044)

All three acceptance tests have been implemented in `internal/vps/resource/server_resource_test.go`:

1. **T042**: `TestAccServerResource_WaitForActiveFalse` (lines 272-310)
   - Tests server creation with `wait_for_active = false`
   - Verifies Terraform returns immediately without waiting
   - Status may be "building" or "active" depending on API speed

2. **T043**: `TestAccServerResource_WaitForActiveTrue` (lines 319-368)
   - Tests explicit `wait_for_active = true` setting
   - Verifies server reaches "active" status before return
   - Includes custom timeout configuration (15m)

3. **T044**: `TestAccServerResource_WaitForActiveDefault` (lines 370-416)
   - Tests default behavior (wait_for_active not specified)
   - Verifies default is `true` (wait for active)
   - Confirms computed value stored in state

### Implementation Tasks (T045-T047)

These were already completed in previous work:

1. **T045**: `wait_for_active` attribute in schema
   - Location: `internal/vps/resource/server_resource.go:219-224`
   - Type: BoolAttribute
   - Optional: true
   - Computed: true (defaults to true)
   - MarkdownDescription documents behavior

2. **T046**: Conditional waiting logic
   - Location: `internal/vps/resource/server_resource.go:302-338`
   - Checks `plan.WaitForActive.IsNull()` and `.ValueBool()`
   - Skips `waitForServerActive()` when false
   - Stores server state immediately when skipped

3. **T047**: Intermediate state handling
   - Status attribute stores current server state
   - Values: "building", "active", "error", "deleted"
   - Populated from API response regardless of wait setting

### Documentation & Examples (T048-T049)

1. **T048**: Async creation example
   - File: `examples/resources/zillaforge_server/resource-async.tf`
   - Demonstrates batch creation of 3 servers with `wait_for_active = false`
   - Includes outputs for server IDs and statuses
   - Documents use case for large deployments

2. **T049**: Documentation complete
   - File: `docs/resources/server.md:102`
   - wait_for_active fully documented in Optional attributes section
   - Describes behavior when true vs false
   - Notes default value is true

## 🔧 Technical Implementation

### SDK Integration

The implementation uses the cloud-sdk's built-in waiter:

```go
// internal/vps/resource/server_resource.go:741-752
func (r *ServerResource) waitForServerActive(ctx context.Context, client vps.VPSService, serverID string, timeout time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	waiterConfig := vpscore.ServerWaiterConfig{
		Client:       client,
		ServerID:     serverID,
		TargetStatus: servermodels.ServerStatusActive,
	}

	return vpscore.WaitForServerStatus(ctxWithTimeout, waiterConfig)
}
```

### Conditional Execution

```go
// internal/vps/resource/server_resource.go:302-338
waitForActive := true
if !plan.WaitForActive.IsNull() {
	waitForActive = plan.WaitForActive.ValueBool()
}

if waitForActive {
	tflog.Info(ctx, "Waiting for server to become active", map[string]interface{}{
		"server_id": result.Server.ID,
		"timeout":   createTimeout.String(),
	})

	if err := r.waitForServerActive(ctx, r.vpsClient, result.Server.ID, createTimeout); err != nil {
		return err
	}

	// Refresh server state after waiting
	getResult, err := r.vpsClient.GetServer(ctx, result.Server.ID)
	// ...
}
```

## 🧪 Testing

Tests are structured following Terraform acceptance test patterns:

- Use `resource.Test()` framework
- Leverage data sources for flavors, images, networks, security groups
- Verify attributes with `resource.TestCheckResourceAttr()`
- Three scenarios: false, explicit true, default (implicit true)

To run the tests:

```bash
TF_ACC=1 go test -v ./internal/vps/resource -run "TestAccServerResource_(WaitForActiveFalse|WaitForActiveTrue|WaitForActiveDefault)" -timeout 30m
```

## 📊 Phase 5 Status

| Task | Type | Status | Location |
|------|------|--------|----------|
| T042 | Test | ✅ Complete | `server_resource_test.go:272-310` |
| T043 | Test | ✅ Complete | `server_resource_test.go:319-368` |
| T044 | Test | ✅ Complete | `server_resource_test.go:370-416` |
| T045 | Code | ✅ Complete | `server_resource.go:64, 219-224` |
| T046 | Code | ✅ Complete | `server_resource.go:302-338` |
| T047 | Code | ✅ Complete | `server_resource.go:226 (status field)` |
| T048 | Example | ✅ Complete | `resource-async.tf` |
| T049 | Docs | ✅ Complete | `docs/resources/server.md:102` |

## ✨ User Story 3: Complete

All requirements for User Story 3 (Asynchronous Server Creation) have been implemented and tested:

- ✅ Users can set `wait_for_active = false` for immediate return
- ✅ Default behavior waits for active status (backwards compatible)
- ✅ Batch deployment scenario documented with example
- ✅ Acceptance tests verify all three scenarios
- ✅ Documentation complete with behavior explanation

**Independent Test Verification**: Users can set `wait_for_active = false`, apply config, and Terraform returns immediately without polling server status.
