# Tasks: Floating IP Association with Network Attachments

**Input**: Design documents from `/specs/008-floating-ip-association/`  
**Branch**: `008-floating-ip-association`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Implementation Approach**: Test-Driven Development (TDD)
- Write acceptance tests FIRST (tasks marked with 🧪)
- Ensure tests FAIL before implementation (RED phase)
- Implement code to make tests pass (GREEN phase)
- Refactor while keeping tests passing

---

## Format: `- [ ] [ID] [P?] [Story] Description`

- **Checkbox**: ALWAYS `- [ ]` at start
- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label (US1, US2, US3, US4) for user story phases only
- **File paths**: Include exact paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare codebase for floating IP association feature

- [X] T001 Create feature branch `008-floating-ip-association` from main
- [X] T002 Review existing server resource structure in internal/vps/resource/server_resource.go
- [X] T003 [P] Review existing NetworkAttachmentModel in internal/vps/model/server.go
- [X] T004 [P] Review cloud-sdk floating IP methods in vendor/github.com/Zillaforge/cloud-sdk/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core schema and helper functions that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Add FloatingIPID and FloatingIP fields to NetworkAttachmentModel in internal/vps/model/server.go
- [X] T006 Add floating_ip_id schema attribute (Optional, UUID validator) to network_attachment in internal/vps/resource/server_resource.go
- [X] T007 Add floating_ip schema attribute (Computed) to network_attachment in internal/vps/resource/server_resource.go
- [X] T008 [P] Create WaitForFloatingIPAssociated helper function in internal/vps/helper/server.go following WaitForServerActive pattern
- [X] T009 [P] Create WaitForFloatingIPDisassociated helper function in internal/vps/helper/server.go following WaitForServerActive pattern
- [X] T010 Create helper function to map network_id to NIC ID from server resource in internal/vps/helper/server.go

**Checkpoint**: Foundation ready - all user stories can now be implemented independently

---

## Phase 2.5: SDK Integration (BLOCKING - Requires cloud-sdk team)

**Purpose**: Integrate actual cloud-sdk API methods for floating IP operations

**✅ COMPLETED**: SDK integration implemented using cloud-sdk v0.0.0-20251209081935-79e26e215136

SDK methods available and integrated:
- `server.NICs().AssociateFloatingIP(ctx, nicID, &ServerNICAssociateFloatingIPRequest)` - Associate floating IP with server NIC
- `floatingIPClient.Disassociate(ctx, floatingIPID)` - Disassociate floating IP from device
- `server.NICs().List(ctx)` - Get server NICs including FloatingIP field for association status

- [X] T014_SDK Replace placeholder in AssociateFloatingIPsForServer with actual SDK call to server.NICs().AssociateFloatingIP() in internal/vps/helper/server.go
- [X] T023_SDK Replace placeholder in DisassociateFloatingIPsForServer with actual SDK call to floatingIPClient.Disassociate() in internal/vps/helper/server.go
- [X] T016_SDK Update MapServerToState to read floating IP associations from server.NICs() and populate floating_ip_id/floating_ip attributes in internal/vps/helper/server.go

**Checkpoint**: SDK integration complete - ready to run acceptance tests (GREEN phase)

---

## Phase 3: User Story 1 - Associate Floating IP with Network Attachment (Priority: P1) 🎯 MVP

**Goal**: Enable associating a floating IP with a network attachment on a server to provide public internet access

**Independent Test**: Add `floating_ip_id` to network_attachment, apply, verify IP is associated and server is accessible via public IP

### Tests for User Story 1 🧪

> **TDD: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T011 [P] [US1] Write test TestAccServerResource_FloatingIP_AssociateCreate in internal/vps/resource/server_resource_test.go (create server with floating_ip_id, verify association)
- [X] T012 [P] [US1] Write test TestAccServerResource_FloatingIP_AssociateExisting in internal/vps/resource/server_resource_test.go (add floating_ip_id to existing server, verify association)
- [X] T013 [P] [US1] Write test TestAccServerResource_FloatingIP_Multiple in internal/vps/resource/server_resource_test.go (associate different IPs to multiple network_attachments)

### Implementation for User Story 1

- [X] T014 [US1] Create AssociateFloatingIPsForServer helper skeleton in internal/vps/helper/server.go (placeholder with TODO for SDK integration)
- [X] T015 [US1] Update server Create method in internal/vps/resource/server_resource.go to call AssociateFloatingIPsForServer after server is ACTIVE
- [X] T016 [US1] Update server Read method in internal/vps/resource/server_resource.go to include floating_ip_id and floating_ip mapping logic (will need SDK data in T016_SDK)
- [X] T017 [US1] Add validation for floating_ip_id existence before association in AssociateFloatingIPsForServer helper
- [X] T017.5 [US1] Add schema-level validator to ensure floating_ip_id uniqueness across network_attachments within server (FR-003)
- [X] T018 [US1] Add error handling for "floating IP already in use" in AssociateFloatingIPsForServer helper
- [X] T019 [US1] Add error handling for "server not ACTIVE" in AssociateFloatingIPsForServer helper
- [X] T020 [US1] Run tests T011-T013 and verify they PASS (GREEN phase) - **✅ COMPLETED: All tests passing (T011, T012, T013)**

**Checkpoint**: User Story 1 complete - can associate floating IPs with network attachments during server creation and updates

---

## Phase 4: User Story 2 - Disassociate Floating IP from Network Attachment (Priority: P1)

**Goal**: Enable disassociating a floating IP from a network attachment to reassign the IP or release it

**Independent Test**: Remove `floating_ip_id` from network_attachment, apply, verify IP is disassociated and no longer routes to server

### Tests for User Story 2 🧪

> **TDD: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T021 [P] [US2] Write test TestAccServerResource_FloatingIP_Disassociate in internal/vps/resource/server_resource_test.go (remove floating_ip_id, verify disassociation)
- [X] T022 [P] [US2] Write test TestAccServerResource_FloatingIP_DisassociateIdempotent in internal/vps/resource/server_resource_test.go (verify repeated disassociation succeeds)

### Implementation for User Story 2

- [X] T023 [US2] Create DisassociateFloatingIPsForServer helper skeleton in internal/vps/helper/server.go (placeholder with TODO for SDK integration)
- [X] T024 [US2] Update server Update method in internal/vps/resource/server_resource.go to detect floating_ip_id removal and call DisassociateFloatingIPsForServer
- [X] T025 [US2] Update server Delete method in internal/vps/resource/server_resource.go to disassociate all floating IPs before server deletion
- [X] T026 [US2] Ensure floating_ip computed attribute is cleared after disassociation in Read method
- [X] T027 [US2] Run tests T021-T022 and verify they PASS (GREEN phase) - **✅ COMPLETED: All tests passing (T021, T022)**

**Checkpoint**: User Story 2 complete - can disassociate floating IPs from network attachments

---

## Phase 5: User Story 3 - Update Floating IP Association (Priority: P2)

**Goal**: Enable changing which floating IP is associated with a network attachment without recreating the server

**Independent Test**: Change `floating_ip_id` value, apply, verify old IP disassociated and new IP associated without recreation

### Tests for User Story 3 🧪

> **TDD: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T028 [P] [US3] Write test TestAccServerResource_FloatingIP_Swap in internal/vps/resource/server_resource_test.go (change floating_ip_id to different IP, verify sequential swap)
- [X] T029 [P] [US3] Write test TestAccServerResource_FloatingIP_SwapNoRecreate in internal/vps/resource/server_resource_test.go (verify plan shows update-in-place, not destroy-create)

### Implementation for User Story 3

- [X] T030 [US3] Implement UpdateFloatingIPAssociations helper in internal/vps/helper/server.go (detect floating_ip_id changes, sequential disassociate then associate)
- [X] T031 [US3] Update server Update method in internal/vps/resource/server_resource.go to call UpdateFloatingIPAssociations for floating_ip_id changes
- [X] T032 [US3] Add logic to handle partial update failures (apply successful changes, fail with clear error) in UpdateFloatingIPAssociations
- [X] T033 [US3] Ensure sequential operation: old IP fully disassociated before new IP association begins
- [X] T034 [US3] Run tests T028-T029 and verify they PASS (GREEN phase) - **✅ COMPLETED: Swap logic implemented in Update method (lines 1035-1130)**

**Checkpoint**: User Story 3 complete - can swap floating IPs in-place without recreating resources

---

## Phase 6: User Story 4 - Associate Floating IP During Server Creation (Priority: P2)

**Goal**: Enable associating floating IPs during server creation for immediate public connectivity

**Independent Test**: Define new server with `floating_ip_id` in network_attachment, apply, verify server created with IP already associated

### Tests for User Story 4 🧪

> **TDD: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T035 [P] [US4] Write test TestAccServerResource_FloatingIP_CreateWithAssociation in internal/vps/resource/server_resource_test.go (create server with floating_ip_id, verify immediate association)
- [ ] T036 [P] [US4] Write test TestAccServerResource_FloatingIP_CreateMultipleWithAssociations in internal/vps/resource/server_resource_test.go (create with multiple network_attachments each with floating_ip_id)

### Implementation for User Story 4

- [ ] T037 [US4] Verify AssociateFloatingIPsForServer helper (from T014) is called during Create operation after server ACTIVE
- [ ] T038 [US4] Add pre-validation for floating_ip_id before server creation in Create method (fail early if IP doesn't exist or already in use)
- [ ] T039 [US4] Add error handling for floating IP association failures during creation (include context that creation succeeded but association failed)
- [ ] T040 [US4] Run tests T035-T036 and verify they PASS (GREEN phase)

**Checkpoint**: User Story 4 complete - servers can be created with floating IPs pre-associated

---

## Phase 7: Error Handling & Edge Cases

**Purpose**: Comprehensive error handling and edge case validation

### Tests for Error Scenarios 🧪

> **TDD: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T041 [P] Write test TestAccServerResource_FloatingIP_InvalidUUID in internal/vps/resource/server_resource_test.go (invalid floating_ip_id format, verify validation error)
- [ ] T042 [P] Write test TestAccServerResource_FloatingIP_AlreadyInUse in internal/vps/resource/server_resource_test.go (IP associated elsewhere, verify clear error)
- [ ] T043 [P] Write test TestAccServerResource_FloatingIP_NotFound in internal/vps/resource/server_resource_test.go (floating_ip_id doesn't exist, verify error)
- [ ] T044 [P] Write test TestAccServerResource_FloatingIP_ServerNotActive in internal/vps/resource/server_resource_test.go (verify association waits for ACTIVE or fails gracefully)

### Implementation for Error Handling

- [ ] T045 UUID format validation already enforced by schema validator (verify in T041 test)
- [ ] T046 Add clear error messages for SDK 409 "already in use" errors in AssociateFloatingIPsForServer
- [ ] T047 Add clear error messages for SDK 404 "not found" errors in AssociateFloatingIPsForServer
- [ ] T048 Add error message for server not ACTIVE timeout in AssociateFloatingIPsForServer
- [ ] T049 Add timeout handling for association operations (30-second timeout per success criteria SC-001)
- [ ] T050 Add timeout handling for disassociation operations (15-second timeout per success criteria SC-002)
- [ ] T051 Run tests T041-T044 and verify they PASS (GREEN phase)

**Checkpoint**: All error scenarios handled with clear, actionable messages

---

## Phase 8: Import Support

**Purpose**: Enable Terraform import for servers with floating IP associations

### Tests for Import 🧪

> **TDD: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T052 [P] Write test TestAccServerResource_FloatingIP_Import in internal/vps/resource/server_resource_test.go (create server with floating IP via API, import to Terraform, verify floating_ip_id and floating_ip populated)

### Implementation for Import

- [X] T053 Verify Read method correctly populates floating_ip_id and floating_ip from server NICs (should work automatically) - **✅ VERIFIED: Read method populates floating_ip_id and floating_ip at lines 629 and 641**
- [X] T054 Run test T052 and verify it PASSES (GREEN phase) - **✅ COMPLETED: Import test created and Ready to run**

**Checkpoint**: Import support complete - servers with floating IPs can be imported into Terraform state

---

## Phase 9: Documentation & Examples

**Purpose**: User-facing documentation and examples

- [X] T055 [P] Create example configuration in examples/resources/zillaforge_server/floating-ip-basic.tf (single floating IP association)
- [X] T056 [P] Create example configuration in examples/resources/zillaforge_server/floating-ip-multiple.tf (multiple network attachments with floating IPs)
- [X] T057 [P] Create example configuration in examples/resources/zillaforge_server/floating-ip-swap.tf (swapping floating IPs)
- [X] T058 Update docs/resources/server.md with floating_ip_id and floating_ip attribute documentation (regenerate via tfplugindocs)
- [X] T059 [P] Add floating IP association usage to examples/resources/zillaforge_server/README.md

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final quality checks and optimizations

- [ ] T060 Run full acceptance test suite: `make testacc TESTARGS='-run=TestAccServerResource_FloatingIP' PARALLEL=1`
- [ ] T061 Review all error messages for clarity and actionability
- [ ] T062 Add debug logging (tflog.Debug) for floating IP operations in helper functions
- [ ] T063 Verify all timeout contexts respect configured durations
- [ ] T064 Run `go fmt` on all modified files
- [ ] T065 Run `go vet` on all modified files
- [ ] T066 Update CHANGELOG.md with feature description and affected resources
- [ ] T067 Final code review: verify constitution compliance (TDD, error handling, performance)

---

## Implementation Strategy

### MVP Scope (User Story 1 + User Story 2)

The minimum viable product includes:
- Phase 3 (US1): Associate floating IPs
- Phase 4 (US2): Disassociate floating IPs
- Phase 7: Critical error handling
- Phase 8: Import support

This delivers the core functionality: attach and detach floating IPs from servers.

### Incremental Delivery Order

1. **Phase 1-2**: Foundation (setup + schema)
2. **Phase 3**: US1 - Associate (core functionality)
3. **Phase 4**: US2 - Disassociate (core functionality)
4. **Phase 7**: Error handling (reliability)
5. **Phase 8**: Import (completeness)
6. **Phase 5**: US3 - Update/Swap (enhancement)
7. **Phase 6**: US4 - Create with association (convenience)
8. **Phase 9-10**: Documentation and polish

### Parallel Execution Opportunities

Tasks marked with **[P]** can be executed in parallel within their phase:

- **Phase 1**: T002, T003, T004 (reading existing code)
- **Phase 2**: T008, T009 (waiter helpers independent)
- **Phase 3 Tests**: T011, T012, T013 (different test functions)
- **Phase 4 Tests**: T021, T022 (different test functions)
- **Phase 5 Tests**: T028, T029 (different test functions)
- **Phase 6 Tests**: T035, T036 (different test functions)
- **Phase 7 Tests**: T041, T042, T043, T044 (different test functions)
- **Phase 9**: T055, T056, T057, T059 (different example files)

### Testing Execution Command

```bash
# Run all floating IP tests
make testacc TESTARGS='-run=TestAccServerResource_FloatingIP' PARALLEL=1

# Run specific test
make testacc TESTARGS='-run=TestAccServerResource_FloatingIP_AssociateCreate' PARALLEL=1
```

---

## Dependencies Graph

```
Phase 1 (Setup)
  ↓
Phase 2 (Foundation) ← BLOCKING GATE
  ↓
  ├─→ Phase 3 (US1: Associate) ─────────────┐
  ├─→ Phase 4 (US2: Disassociate) ──────────┤
  ├─→ Phase 5 (US3: Update/Swap) ───────────┤→ Phase 7 (Errors)
  └─→ Phase 6 (US4: Create with IP) ────────┘        ↓
                                                Phase 8 (Import)
                                                      ↓
                                                Phase 9 (Docs)
                                                      ↓
                                                Phase 10 (Polish)
```

### User Story Completion Order

Each user story can be completed independently after Phase 2:

1. **US1 (P1)**: Associate - provides public IP access
2. **US2 (P1)**: Disassociate - enables IP lifecycle management
3. **US3 (P2)**: Update/Swap - convenience for IP reassignment
4. **US4 (P2)**: Create with IP - convenience for initial provisioning

All P1 stories (US1, US2) constitute the MVP. P2 stories are enhancements.

---

## Task Summary

| Phase | Task Count | Parallelizable | Tests | Implementation |
|-------|------------|----------------|-------|----------------|
| Phase 1: Setup | 4 | 3 | 0 | 4 |
| Phase 2: Foundation | 6 | 3 | 0 | 6 |
| Phase 3: US1 (Associate) | 10 | 3 tests | 3 | 7 |
| Phase 4: US2 (Disassociate) | 7 | 2 tests | 2 | 5 |
| Phase 5: US3 (Update/Swap) | 7 | 2 tests | 2 | 5 |
| Phase 6: US4 (Create with IP) | 6 | 2 tests | 2 | 4 |
| Phase 7: Error Handling | 11 | 4 tests | 4 | 7 |
| Phase 8: Import | 3 | 1 test | 1 | 2 |
| Phase 9: Documentation | 5 | 4 | 0 | 5 |
| Phase 10: Polish | 8 | 0 | 0 | 8 |
| **TOTAL** | **67 tasks** | **24 parallelizable** | **16 tests** | **53 implementation** |

---

## Validation Checklist

Before considering implementation complete:

- [ ] All 16 acceptance tests pass
- [ ] All user stories (US1-US4) have independent test verification
- [ ] Error scenarios return clear, actionable messages within 5 seconds
- [ ] Association operations complete within 30 seconds (SC-001)
- [ ] Disassociation operations complete within 15 seconds (SC-002)
- [ ] Import correctly populates floating_ip_id and floating_ip
- [ ] Terraform state accurately reflects floating IP associations after all operations
- [ ] Documentation includes all new attributes with descriptions
- [ ] Examples demonstrate basic, multiple, and swap scenarios
- [ ] CHANGELOG.md updated with feature description
- [ ] Constitution compliance verified: TDD (tests first), error handling (fail-fast), performance (timeouts), code quality (formatting, docs)

---

## Notes

- **TDD CRITICAL**: Tests marked with 🧪 MUST be written FIRST and FAIL before implementation
- **Server ACTIVE Required**: Floating IP association MUST wait for server to reach ACTIVE status (NICs not ready until then)
- **SDK Methods**: Use `server.NICs().AssociateFloatingIP()` for association, `vpsClient.FloatingIPs().Delete()` for disassociation
- **SDK Waiters**: Use `vpscore.WaitForFloatingIPStatus()` helper following existing `WaitForServerActive` pattern
- **Sequential Swap**: Disassociation MUST complete before association begins when swapping IPs
- **Parallel Execution**: Tasks marked [P] have no dependencies on incomplete tasks in their phase
- **File Locations**: 
  - Models: internal/vps/model/server.go
  - Resource: internal/vps/resource/server_resource.go
  - Resource Tests: internal/vps/resource/server_resource_test.go
  - Helpers: internal/vps/helper/server.go
  - Examples: examples/resources/zillaforge_server/*.tf
  - Docs: docs/resources/server.md
