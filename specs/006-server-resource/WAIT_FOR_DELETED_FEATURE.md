# wait_for_deleted Feature Implementation Summary

## Overview

Added a new `wait_for_deleted` boolean attribute to the `zillaforge_server` resource that controls whether Terraform waits for server deletion to complete or returns immediately after initiating the delete operation.

## Changes Made

### 1. Core Implementation

#### File: [internal/vps/resource/server_resource.go](internal/vps/resource/server_resource.go)

**ServerResourceModel struct** - Added new field:
```go
WaitForDeleted  types.Bool   `tfsdk:"wait_for_deleted"`
```

**Schema definition** - Added new attribute:
```go
"wait_for_deleted": schema.BoolAttribute{
    MarkdownDescription: "Whether to wait for the server to be fully deleted. When set to `true` (default), Terraform will poll the server status until it is fully deleted or the timeout is exceeded. When set to `false`, Terraform will return immediately after the delete API call, without waiting for the server deletion to complete. Default is `true`.",
    Optional:            true,
    Computed:            true,
    Default:             booldefault.StaticBool(true),
},
```

**Delete() function** - Updated logic to check `wait_for_deleted`:
- When `wait_for_deleted = true` (default): Polls server status until deleted or timeout
- When `wait_for_deleted = false`: Returns immediately after delete API call

### 2. Test Cases

#### File: [internal/vps/resource/server_resource_test.go](internal/vps/resource/server_resource_test.go)

Added three new test cases:

1. **TestAccServerResource_WaitForDeletedTrue** (T045)
   - Explicitly sets `wait_for_deleted = true`
   - Verifies attribute is set correctly

2. **TestAccServerResource_WaitForDeletedFalse** (T046)
   - Sets `wait_for_deleted = false`
   - Verifies attribute is set correctly and delete returns immediately

3. **TestAccServerResource_WaitForDeletedDefault** (T047)
   - Does not specify `wait_for_deleted`
   - Verifies default value is `true`

### 3. Examples

#### File: [examples/resources/zillaforge_server/resource-wait-for-deleted.tf](examples/resources/zillaforge_server/resource-wait-for-deleted.tf)

Created comprehensive example demonstrating:
- Server with `wait_for_deleted = true` (explicit, default behavior)
- Server with `wait_for_deleted = false` (immediate return)
- Usage with custom delete timeout

### 4. Documentation

#### File: [docs/resources/server.md](docs/resources/server.md)

Added attribute documentation:
```markdown
- `wait_for_deleted` (Boolean) Whether to wait for the server to be fully deleted. 
  When set to `true` (default), Terraform will poll the server status until it is 
  fully deleted or the timeout is exceeded. When set to `false`, Terraform will 
  return immediately after the delete API call, without waiting for the server 
  deletion to complete. Default is `true`.
```

#### File: [specs/006-server-resource/data-model.md](specs/006-server-resource/data-model.md)

Updated:
- ServerResourceModel struct definition
- Added new test cases (11-13)
- Added "Behavior Flags" section documenting `wait_for_deleted` behavior

#### File: [specs/006-server-resource/spec.md](specs/006-server-resource/spec.md)

Updated:
- Added functional requirement **FR-009-A** for `wait_for_deleted`
- Added new **User Story 3.1 - Asynchronous Server Deletion (Priority: P3)** with acceptance scenarios

## Behavior

### Default Behavior (wait_for_deleted = true)
1. Terraform calls delete API
2. Polls server status every few seconds
3. Waits until server is fully deleted
4. Respects configured timeout (default: 10m)
5. Returns success or timeout error

### Async Behavior (wait_for_deleted = false)
1. Terraform calls delete API
2. Returns immediately after API call succeeds
3. Does not poll or wait for deletion
4. State is removed immediately

## Use Cases

### wait_for_deleted = true (default)
- Standard single server deletions
- When you need to ensure resources are fully cleaned up
- When dependent resources need the server gone before proceeding
- Production environments with strict cleanup requirements

### wait_for_deleted = false
- Batch deletion of multiple servers
- External orchestration handles deletion verification
- Faster teardown when deletion confirmation isn't critical
- CI/CD pipelines with external cleanup processes

## Testing

To test the implementation:

```bash
# Run specific tests
cd /workspaces/terraform-provider-zillaforge
make test

# Or run acceptance tests (requires API credentials)
TF_ACC=1 go test -v ./internal/vps/resource/ -run="TestAccServerResource_WaitForDeleted"
```

## Related Files

- Implementation: [internal/vps/resource/server_resource.go](internal/vps/resource/server_resource.go)
- Tests: [internal/vps/resource/server_resource_test.go](internal/vps/resource/server_resource_test.go)
- Example: [examples/resources/zillaforge_server/resource-wait-for-deleted.tf](examples/resources/zillaforge_server/resource-wait-for-deleted.tf)
- User docs: [docs/resources/server.md](docs/resources/server.md)
- Design docs: 
  - [specs/006-server-resource/spec.md](specs/006-server-resource/spec.md)
  - [specs/006-server-resource/data-model.md](specs/006-server-resource/data-model.md)

## Backward Compatibility

This change is fully backward compatible:
- Default value is `true`, maintaining existing behavior
- Existing configurations continue to work without modification
- Only affects behavior when explicitly set to `false`
