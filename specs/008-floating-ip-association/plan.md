# Implementation Plan: Floating IP Association with Network Attachments

**Branch**: `008-floating-ip-association` | **Date**: 2025-12-25 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-floating-ip-association/spec.md`

## Summary

This feature enables associating and disassociating floating IPs (public IP addresses) with network attachments on VPS servers through Terraform. Users can attach floating IPs to server network interfaces to provide public internet access, and detach them to reassign to different servers or release back to the pool.

**Technical Approach**: Extend the existing `zillaforge_server` resource by adding two new optional attributes to the `network_attachment` nested block:
- `floating_ip_id` (types.String) - UUID of the floating IP to associate
- `floating_ip` (types.String, computed) - The actual IP address of the associated floating IP

The implementation will use the cloud-sdk's floating IP association/disassociation API methods. Operations will be synchronous with polling, fail immediately on errors, and support sequential swap behavior (disassociate old, then associate new).

## Technical Context

**Language/Version**: Go 1.22.4  
**Primary Dependencies**: 
- github.com/hashicorp/terraform-plugin-framework v1.14.1
- github.com/Zillaforge/cloud-sdk v0.0.0-20251209081935-79e26e215136
- github.com/hashicorp/terraform-plugin-testing v1.11.0

**Storage**: N/A (Terraform state managed by framework)  
**Testing**: `make testacc TESTARGS='-run=TestAccXXXX' PARALLEL=1` (acceptance tests), `go test` (unit tests)  
**Target Platform**: Terraform providers for Linux/macOS/Windows  
**Project Type**: Single project (Terraform provider)  
**Performance Goals**: 
- Floating IP association: <30 seconds
- Floating IP disassociation: <15 seconds
- Synchronous operations with polling

**Constraints**: 
- Must use Terraform Plugin Framework (not SDK v2)
- All operations must respect context timeouts
- State must accurately reflect infrastructure after every operation
- Error messages must be actionable
- UUID validation for floating_ip_id

**Scale/Scope**: 
- Extend 1 existing resource (zillaforge_server)
- Add 2 attributes to network_attachment nested block
- Add helper functions for floating IP association/disassociation
- Add acceptance tests for all CRUD scenarios

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Framework Compliance

**Status**: ✅ PASS

- Uses Terraform Plugin Framework (existing server resource already compliant)
- Schema extensions will include MarkdownDescription for new attributes
- State management will handle null/unknown values per framework semantics
- All new functions will have godoc comments

### II. Test-Driven Development

**Status**: ✅ PASS  

**Plan**:
- Write acceptance tests FIRST for each scenario before implementation
- Tests will initially fail (RED)
- Implement code to make tests pass (GREEN)
- Refactor without breaking tests
- Use `make testacc TESTARGS='-run=TestAccServerResource_FloatingIP*' PARALLEL=1`

**Test Coverage**:
- Associate floating IP to network_attachment (create server with floating IP, add floating IP to existing server)
- Disassociate floating IP from network_attachment (remove floating_ip_id attribute)
- Update/swap floating IPs (change floating_ip_id to different IP)
- Multiple network attachments with different floating IPs
- Error scenarios (invalid UUID, IP already in use, server not ACTIVE)
- Import verification for server with floating IP associations

### III. User Experience Consistency

**Status**: ✅ PASS

- Attribute naming follows conventions: `floating_ip_id`, `floating_ip` (snake_case, descriptive)
- `floating_ip_id` is Optional (can be null/empty)
- `floating_ip` is Computed (read-only, shows actual IP address)
- Error messages will be actionable (e.g., "floating IP xyz is already associated with server abc")
- No breaking changes to existing server resource schema

### IV. Performance & Resource Efficiency

**Status**: ✅ PASS

- Synchronous operations with polling (respects SC-001: <30s, SC-002: <15s)
- Context timeouts respected in all SDK calls
- Sequential swap minimizes API calls (disassociate, then associate)
- No unnecessary API calls during Read if state is fresh
- Fail-fast on validation errors (UUID format, floating IP existence)

**Overall Assessment**: ✅ ALL CONSTITUTIONAL REQUIREMENTS MET

No violations requiring justification. Feature extends existing patterns without introducing complexity.

## Project Structure

### Documentation (this feature)

```text
specs/008-floating-ip-association/
├── spec.md              # Feature specification (completed)
├── plan.md              # This file (implementation plan)
├── research.md          # Phase 0 output (technical research)
├── data-model.md        # Phase 1 output (data structures)
├── quickstart.md        # Phase 1 output (user guide)
├── contracts/           # Phase 1 output (API contracts)
│   └── floating-ip-association-api.md
└── checklists/
    └── requirements.md  # Spec quality checklist (completed)
```

### Source Code (repository root)

```text
internal/vps/
├── model/
│   └── server.go                    # MODIFY: Add floating_ip_id, floating_ip to NetworkAttachmentModel
├── helper/
│   ├── server.go                    # MODIFY: Add floating IP association/disassociation logic
│   └── floating_ip.go               # EXISTS: Reuse for SDK model mapping
├── resource/
│   └── server_resource.go           # MODIFY: Add floating IP schema, Update/Create/Read logic
│       └── server_resource_test.go  # MODIFY: Add floating IP association tests

examples/
└── resources/
    └── zillaforge_server/
        └── floating-ip.tf           # NEW: Example with floating IP association

docs/
└── resources/
    └── server.md                    # REGENERATE: Update docs with new attributes
```

**Structure Decision**: This is a single Terraform provider project. We extend the existing `zillaforge_server` resource rather than creating a separate resource for floating IP associations. This follows Terraform best practices where associations are managed as attributes of the primary resource (similar to AWS EC2 instance + EIP association).

## Complexity Tracking

**Status**: No constitutional violations - this section is empty per template guidance.

This feature extends existing patterns without introducing architectural complexity. All changes are additive (new optional attributes) and follow established Terraform provider conventions.

---

## Phase 0: Research & Decision Capture

**Output**: [research.md](research.md)

**Completed**: ✅

**Summary**: Investigated 7 technical areas to resolve all unknowns from Technical Context:

1. **SDK API Interface** - Confirmed cloud-sdk methods: `AssociateFloatingIP(floatingIPID, AssociateRequest{PortID})` and `DisassociateFloatingIP(floatingIPID)`. Server read provides `network_ports[].port_id` needed for association.

2. **Schema Design** - Decided on two attributes within `network_attachment`:
   - `floating_ip_id` (types.String, Optional) - User specifies UUID to associate
   - `floating_ip` (types.String, Computed) - Framework populates with actual IP address

3. **State Management** - Null/value transitions handled per Terraform semantics:
   - null → "uuid-123": Associate floating IP
   - "uuid-123" → null: Disassociate floating IP
   - "uuid-123" → "uuid-456": Sequential swap (disassociate, then associate)

4. **Polling Implementation** - Synchronous operations with polling:
   - Association: 2-second intervals, 30-second timeout
   - Disassociation: 2-second intervals, 15-second timeout
   - Check `device_id` field in floating IP state

5. **Error Handling** - Fail-fast strategy (no retries per clarification #1):
   - UUID validation pre-flight
   - Actionable error messages mapped from SDK errors
   - Partial update handling (apply successful changes, fail with clear error)

6. **Import Support** - Automatic via existing Read implementation:
   - No custom ImportState needed
   - Read will populate `floating_ip_id` and `floating_ip` from API

7. **Testing Strategy** - TDD approach with 8 acceptance tests:
   - Create with floating IP, Add to existing, Multiple NICs
   - Swap floating IPs, Change on same server, Remove
   - Invalid UUID, Already in use errors

**Decision**: Use NIC ID (from server.NICs()) for SDK association calls, mapping from Terraform's `network_id` to find the correct NIC. See [SDK_API_CORRECTIONS.md](SDK_API_CORRECTIONS.md) for authoritative SDK method signatures.

---

## Phase 1: Design Artifacts

### Data Model

**Output**: [data-model.md](data-model.md)

**Completed**: ✅

**Summary**: Extended `NetworkAttachmentModel` with two new fields:

```go
type NetworkAttachmentModel struct {
    NetworkID        types.String   `tfsdk:"network_id"`
    IPAddress        types.String   `tfsdk:"ip_address"`
    IsPrimary        types.Bool     `tfsdk:"is_primary"`
    SecurityGroupIDs types.Set      `tfsdk:"security_group_ids"`
    FloatingIPID     types.String   `tfsdk:"floating_ip_id"` // NEW: Optional
    FloatingIP       types.String   `tfsdk:"floating_ip"`    // NEW: Computed
}
```

**Schema Definition**:
- `floating_ip_id`: Optional, UUID validation via custom validator
- `floating_ip`: Computed, shows actual IP address or empty string

**Helper Functions Specified**:
- `AssociateFloatingIPsForServer(ctx, client, serverID, planAttachments, apiServer)` - Create scenario
- `UpdateFloatingIPAssociations(ctx, client, serverID, stateAttachments, planAttachments, apiServer)` - Update scenario
- `waitForFloatingIPAssociated(ctx, client, floatingIPID, serverID, timeout)` - Polling for association
- `waitForFloatingIPDisassociated(ctx, client, floatingIPID, timeout)` - Polling for disassociation
- `findFloatingIPForPort(portID, ipAddresses)` - Map port_id to actual IP address

**State Transitions**: Documented for Create/Read/Update/Delete with null handling.

### User Guide

**Output**: [quickstart.md](quickstart.md)

**Completed**: ✅

**Summary**: Comprehensive user guide with 9 usage scenarios:

**Basic Usage**:
1. Create server with floating IP association
2. Add floating IP to existing server
3. Multiple network attachments with floating IPs

**Advanced Scenarios**:
4. Swap floating IP between servers (requires lifecycle depends_on)
5. Change floating IP on same server
6. Remove floating IP from server

**Error Handling**:
- Invalid UUID format
- Floating IP already in use
- Floating IP not found

**Additional Sections**:
- Import documentation (automatic via existing mechanism)
- Best practices (explicit dependencies, separate resources, use data sources for lookup)
- Troubleshooting (association timeouts, swap failures)
- Related resources (zillaforge_floating_ip, zillaforge_network data sources)

### API Contracts

**Output**: [contracts/floating-ip-association-api.md](contracts/floating-ip-association-api.md)

**Completed**: ✅

**Summary**: Documented cloud-sdk API contract:

**Endpoints**:
- `POST /vps/floating-ips/{id}/associate` - Body: `{"port_id": "uuid"}`
- `POST /vps/floating-ips/{id}/disassociate` - No body
- `GET /vps/floating-ips/{id}` - Check association status
- `GET /vps/servers/{id}` - Get network_ports for port_id mapping

**Response Formats**: FloatingIP model with `device_id` field indicating association.

**Error Codes**: 404 (not found), 409 (conflict/already in use), 422 (invalid state).

**Operation Sequences**: Documented for associate, disassociate, and sequential swap.

**Polling Strategy**: 2-second intervals with 30s (associate) / 15s (disassociate) timeouts.

**SDK Method Signatures**: Expected Go interfaces for VPSClient and floatingipmodels.

---

## Constitution Re-Check (Post-Design)

*GATE: Re-evaluate constitution compliance after Phase 1 design artifacts.*

### I. Code Quality & Framework Compliance

**Status**: ✅ PASS

**Evidence from Design**:
- Schema definitions in data-model.md follow framework conventions (Optional, Computed)
- NetworkAttachmentModel struct uses proper tfsdk tags
- Helper functions have descriptive names and parameters
- Error handling returns wrapped errors with context

### II. Test-Driven Development

**Status**: ✅ PASS

**Evidence from Design**:
- Testing strategy in research.md defines 8 acceptance tests FIRST
- Test scenarios cover all state transitions (null→value, value→null, value→value)
- Error scenarios included (invalid UUID, 409 conflict, 404 not found)
- Import test verifies Read populates floating IP attributes

**Next Steps**: Write failing tests before implementation (RED phase).

### III. User Experience Consistency

**Status**: ✅ PASS

**Evidence from Design**:
- quickstart.md demonstrates intuitive HCL syntax matching existing patterns
- Attribute naming is self-documenting (`floating_ip_id` vs `floating_ip`)
- Error messages mapped from SDK are actionable (research.md error handling section)
- Examples show common patterns (add, remove, swap) without complexity

### IV. Performance & Resource Efficiency

**Status**: ✅ PASS

**Evidence from Design**:
- Polling strategy respects performance targets (30s/15s) from constitution
- Sequential swap minimizes API calls (disassociate completes before associate)
- No redundant API calls during Read (only when necessary to resolve floating_ip address)
- Helper functions reuse server API response to avoid duplicate fetches

**Overall Assessment**: ✅ ALL CONSTITUTIONAL REQUIREMENTS STILL MET

Design phase did not introduce any violations. Implementation can proceed to Phase 2 (tasks.md generation).

---

## Planning Phase Complete

### Deliverables Summary

| Artifact | Status | Purpose |
|----------|--------|---------|
| [spec.md](spec.md) | ✅ Complete | Feature specification with user stories, requirements, success criteria |
| [plan.md](plan.md) | ✅ Complete | This document - implementation plan with technical context |
| [research.md](research.md) | ✅ Complete | Phase 0 - Technical research resolving unknowns |
| [data-model.md](data-model.md) | ✅ Complete | Phase 1 - Data structures, schema, helper functions |
| [quickstart.md](quickstart.md) | ✅ Complete | Phase 1 - User guide with 9 scenarios |
| [contracts/floating-ip-association-api.md](contracts/floating-ip-association-api.md) | ✅ Complete | Phase 1 - SDK API contract specification |
| [checklists/requirements.md](checklists/requirements.md) | ✅ Complete | Spec quality validation |

### Ready for Implementation

**Branch**: `008-floating-ip-association`

**Next Command**: `/speckit.tasks` to generate Phase 2 implementation tasks (tasks.md)

**Implementation Approach**: Test-Driven Development
1. Write acceptance tests FIRST (8 tests from research.md)
2. Run tests - should FAIL (RED)
3. Implement code to pass tests (GREEN)
4. Refactor while keeping tests passing
5. Use `make testacc TESTARGS='-run=TestAccServerResource_FloatingIP*' PARALLEL=1`

**Key Implementation Notes**:
- Extend NetworkAttachmentModel in [internal/vps/model/server.go](../../internal/vps/model/server.go)
- Add floating IP logic to [internal/vps/helper/server.go](../../internal/vps/helper/server.go)
- Update schema in [internal/vps/resource/server_resource.go](../../internal/vps/resource/server_resource.go)
- UUID validation for floating_ip_id (custom validator)
- Polling with context timeouts (30s associate, 15s disassociate)
- Sequential swap behavior (disassociate old, then associate new)

---

## Report

**Feature**: 008-floating-ip-association  
**Specification**: [/workspaces/terraform-provider-zillaforge/specs/008-floating-ip-association/spec.md](spec.md)  
**Implementation Plan**: [/workspaces/terraform-provider-zillaforge/specs/008-floating-ip-association/plan.md](plan.md)  
**Branch**: `008-floating-ip-association`

**Planning Status**: ✅ Complete (Phase 0 + Phase 1)

**Generated Artifacts**:
- Phase 0: research.md (7 technical investigations)
- Phase 1: data-model.md (data structures and mapping logic)
- Phase 1: quickstart.md (user guide with 9 scenarios)
- Phase 1: contracts/floating-ip-association-api.md (API specification)

**Constitution Gates**: ✅ All passed (pre-design and post-design checks)

**Next Steps**: Run `/speckit.tasks` to generate implementation task breakdown (tasks.md) and begin TDD cycle.
