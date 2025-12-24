---

description: "Task list for Server Resource (VPS Virtual Machine) implementation"
---

# Tasks: Server Resource (VPS Virtual Machine)

**Input**: Design documents from `/specs/006-server-resource/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Acceptance tests are included to verify each user story independently.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Terraform Provider**: `internal/vps/resource/`, `internal/vps/validators/`
- **Documentation**: `docs/resources/`, `examples/resources/`
- **Tests**: `internal/vps/resource/*_test.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for server resource

- [X] T001 Create server resource file structure in internal/vps/resource/
- [X] T002 [P] Create server resource documentation skeleton in docs/resources/server.md
- [X] T003 [P] Create server resource examples directory in examples/resources/zillaforge_server/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Implement ServerResourceModel struct in internal/vps/resource/server_resource.go
- [X] T005 Implement NetworkAttachmentModel struct with security_group_ids field in internal/vps/resource/server_resource.go
- [X] T006 Implement TimeoutsModel struct in internal/vps/resource/server_resource.go
- [X] T007 [P] Create server resource schema with all required attributes (name, flavor, image, network_attachment with nested security_group_ids) in internal/vps/resource/server_resource.go
- [X] T008 [P] Create FlavorIDValidator to verify flavor exists using cloud-sdk in internal/vps/validators/flavor_validator.go
- [X] T009 [P] Create ImageIDValidator to verify image exists using cloud-sdk in internal/vps/validators/image_validator.go
- [X] T010 [P] Create NetworkIDValidator to verify network exists using cloud-sdk in internal/vps/validators/network_validator.go
- [X] T011 [P] Create SecurityGroupIDValidator to verify security group exists using cloud-sdk in internal/vps/validators/security_group_validator.go
- [X] T012 Create NetworkAttachmentPrimaryConstraint custom validator in internal/vps/validators/network_attachment_validator.go
- [X] T013 Register zillaforge_server resource in provider.go Resources() method in internal/provider/provider.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Create and Configure VPS Instances (Priority: P1) 🎯 MVP

**Goal**: Users can create VPS instances with name, flavor (by ID), image (by ID), network attachments (with per-NIC security_group_ids), and optional attributes (keypair, user data)

**Independent Test**: Define a server resource in Terraform with required attributes, apply it, and verify the instance is active in ZillaForge with correct specifications

### Acceptance Tests for User Story 1

> **NOTE: These tests verify the complete user story functionality**

- [X] T014 [P] [US1] Create acceptance test for basic server creation with required attributes in internal/vps/resource/server_resource_test.go
- [X] T015 [P] [US1] Create acceptance test for server with optional attributes (keypair, user_data) in internal/vps/resource/server_resource_test.go
- [X] T016 [P] [US1] Create acceptance test for server with multiple network attachments in internal/vps/resource/server_resource_test.go
- [X] T017 [P] [US1] Create acceptance test for server destruction in internal/vps/resource/server_resource_test.go

### Implementation for User Story 1

- [X] T018 [US1] Implement Create method for zillaforge_server resource in internal/vps/resource/server_resource.go
- [X] T019 [US1] Implement buildCreateRequest helper to map Terraform plan to cloud-sdk CreateRequest in internal/vps/resource/server_resource.go
- [X] T020 [US1] Implement waitForServerActive polling logic with timeout handling in internal/vps/resource/server_resource.go
- [X] T021 [US1] Implement Read method for zillaforge_server resource in internal/vps/resource/server_resource.go
- [X] T022 [US1] Implement mapServerToState helper to map cloud-sdk Server to Terraform state in internal/vps/resource/server_resource.go
- [X] T023 [US1] Implement Delete method for zillaforge_server resource in internal/vps/resource/server_resource.go
- [X] T024 [US1] Implement waitForServerDeleted polling logic with timeout handling in internal/vps/resource/server_resource.go
- [X] T025 [US1] Add user_data base64 encoding validation and handling in internal/vps/resource/server_resource.go
- [X] T026 [P] [US1] Create basic example configuration in examples/resources/zillaforge_server/resource.tf
- [X] T027 [P] [US1] Create example with optional attributes in examples/resources/zillaforge_server/resource-with-options.tf
- [X] T028 [P] [US1] Create example with multiple networks in examples/resources/zillaforge_server/resource-multi-network.tf
- [X] T029 [P] [US1] Document resource attributes and usage in docs/resources/server.md

**Checkpoint**: At this point, User Story 1 should be fully functional - users can create, read, and delete server instances

---

## Phase 4: User Story 2 - Update Instance Configuration (Priority: P2)

**Goal**: Users can update instance attributes (name, description, network attachments, security groups) in-place without recreating the instance

**Independent Test**: Modify updateable attributes in Terraform config, apply changes, and verify instance updates without replacement

### Acceptance Tests for User Story 2

- [X] T030 [P] [US2] Create acceptance test for updating server name in-place in internal/vps/resource/server_resource_test.go
- [X] T031 [P] [US2] Create acceptance test for updating server description in-place in internal/vps/resource/server_resource_test.go
- [X] T032 [P] [US2] Create acceptance test for updating network attachments in-place in internal/vps/resource/server_resource_test.go (COMMENTED OUT - SDK limitation)
- [X] T033 [P] [US2] Create acceptance test for updating security_group_ids within network_attachment in-place in internal/vps/resource/server_resource_test.go (COMMENTED OUT - SDK limitation)
- [X] T034 [P] [US2] Create acceptance test verifying flavor change forces replacement in internal/vps/resource/server_resource_test.go
- [X] T035 [P] [US2] Create acceptance test verifying image change forces replacement in internal/vps/resource/server_resource_test.go

### Implementation for User Story 2

- [X] T036 [US2] Implement Update method for zillaforge_server resource in internal/vps/resource/server_resource.go
- [X] T037 [US2] Implement buildUpdateRequest helper to map changed attributes to cloud-sdk UpdateRequest in internal/vps/resource/server_resource.go
- [X] T038 [US2] Add plan modifiers for immutable attributes (flavor, image, keypair, user_data) requiring replacement in internal/vps/resource/server_resource.go
- [X] T039 [US2] Implement waitForServerUpdate polling logic with timeout handling in internal/vps/resource/server_resource.go
- [X] T040 [P] [US2] Create example demonstrating in-place updates in examples/resources/zillaforge_server/resource-update.tf
- [X] T041 [P] [US2] Document updateable vs immutable attributes in docs/resources/server.md

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - create, update, and delete operations are complete

**NOTE**: Cloud-SDK v0.0.0-20251209081935-79e26e215136 does not support updating network_attachment or security_group_ids. The Update method only supports name and description changes. Network configuration changes require server recreation. Tests T032 and T033 are commented out until SDK support is added.

---

## Phase 5: User Story 3 - Asynchronous Server Creation (Priority: P3)

**Goal**: Users can create servers without waiting for active status by setting wait_for_active = false for faster batch deployments

**Independent Test**: Set wait_for_active = false, apply config, and verify Terraform returns immediately without polling

### Acceptance Tests for User Story 3

- [X] T042 [P] [US3] Create acceptance test for server creation with wait_for_active = false in internal/vps/resource/server_resource_test.go
- [X] T043 [P] [US3] Create acceptance test for server creation with wait_for_active = true (default) in internal/vps/resource/server_resource_test.go
- [X] T044 [P] [US3] Create acceptance test verifying state after async creation (may show building status) in internal/vps/resource/server_resource_test.go

### Implementation for User Story 3

- [X] T045 [US3] Add wait_for_active boolean attribute to server resource schema in internal/vps/resource/server_resource.go
- [X] T046 [US3] Modify Create method to conditionally skip waitForServerActive based on wait_for_active flag in internal/vps/resource/server_resource.go
- [X] T047 [US3] Update Create method to handle and store intermediate status states (building, error) in internal/vps/resource/server_resource.go
- [X] T048 [P] [US3] Create example demonstrating asynchronous creation in examples/resources/zillaforge_server/resource-async.tf
- [X] T049 [P] [US3] Document wait_for_active behavior and use cases in docs/resources/server.md

**Checkpoint**: All three user stories (1, 2, 3) should now be independently functional

---

## Phase 6: User Story 4 - Import Existing VPS Instances (Priority: P3)

**Goal**: Users can import manually-created VPS instances into Terraform management using instance ID

**Independent Test**: Create instance manually, import it into Terraform state, verify subsequent plan shows no changes with matching config

### Acceptance Tests for User Story 4

- [X] T050 [P] [US4] Create acceptance test for importing existing server by ID in internal/vps/resource/server_resource_test.go
- [X] T051 [P] [US4] Create acceptance test verifying imported server shows no drift with matching config in internal/vps/resource/server_resource_test.go
- [X] T052 [P] [US4] Create acceptance test verifying import with invalid ID returns error in internal/vps/resource/server_resource_test.go

### Implementation for User Story 4

- [X] T053 [US4] Implement ImportState method for zillaforge_server resource in internal/vps/resource/server_resource.go
- [X] T054 [US4] Add import validation to ensure retrieved server matches provider project context in internal/vps/resource/server_resource.go
- [X] T055 [US4] Handle missing user_data field during import (not returned by API for security) in internal/vps/resource/server_resource.go
- [X] T056 [P] [US4] Create import example script in examples/resources/zillaforge_server/import.sh
- [X] T057 [P] [US4] Document import procedure and limitations in docs/resources/server.md

**Checkpoint**: All four user stories should now be complete - full CRUD + import functionality

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T058 [P] Add comprehensive error handling for quota exceeded scenarios across all operations in internal/vps/resource/server_resource.go
- [ ] T059 [P] Add detailed error messages for validation failures (invalid flavor/image combinations) in internal/vps/resource/server_resource.go
- [ ] T060 [P] Add retry logic for transient API errors in internal/vps/resource/server_resource.go
- [ ] T061 [P] Implement timeouts configuration block (create, update, delete) in internal/vps/resource/server_resource.go
- [ ] T062 [P] Add schema descriptions and markdown descriptions for all attributes in internal/vps/resource/server_resource.go
- [ ] T063 Complete resource documentation with all examples and edge cases in docs/resources/server.md
- [X] T064 [P] Removed `availability_zone` attribute from the schema (not supported) and updated specs/docs
- [ ] T065 [P] Add password optional attribute to schema (sensitive, requires replacement) in internal/vps/resource/server_resource.go
- [ ] T066 Run quickstart.md validation scenarios to verify end-to-end functionality
- [ ] T067 Code review and refactoring for consistency with existing resources (keypair, security_group patterns)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3 → P3)
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Extends US1 but is independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Modifies US1 create behavior but is independently testable
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) - Independent import functionality, testable separately

### Within Each User Story

- Tests verify the complete user story functionality after implementation
- Models and data structures before API methods
- Create before Update/Delete
- Core implementation before documentation
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel (T002, T003)
- All Foundational tasks marked [P] can run in parallel (T007-T011)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All acceptance tests for a user story marked [P] can run in parallel
- Documentation tasks marked [P] can run in parallel with implementation
- Polish tasks marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all acceptance tests for User Story 1 together:
Task: "Create acceptance test for basic server creation with required attributes"
Task: "Create acceptance test for server with optional attributes"
Task: "Create acceptance test for server with multiple network attachments"
Task: "Create acceptance test for server destruction"

# Launch all documentation tasks together:
Task: "Create basic example configuration"
Task: "Create example with optional attributes"
Task: "Create example with multiple networks"
Task: "Document resource attributes and usage"
```

---

## Parallel Example: Foundational Phase

```bash
# Launch all validators in parallel (different files):
Task: "Create FlavorIDValidator"
Task: "Create ImageIDValidator"
Task: "Create NetworkIDValidator"
Task: "Create SecurityGroupIDValidator"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Create/Read/Delete)
4. **STOP and VALIDATE**: Test User Story 1 independently with acceptance tests
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 (Create/Read/Delete) → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 (Updates) → Test independently → Deploy/Demo
4. Add User Story 3 (Async creation) → Test independently → Deploy/Demo
5. Add User Story 4 (Import) → Test independently → Deploy/Demo
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Create/Read/Delete)
   - Developer B: User Story 2 (Updates)
   - Developer C: User Story 3 (Async) + User Story 4 (Import)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability (US1, US2, US3, US4)
- Each user story should be independently completable and testable
- Acceptance tests verify complete user story functionality
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Follow existing patterns from keypair_resource.go and security_group_resource.go
- All flavor and image references MUST use platform IDs (not names)
- Security groups are assigned per network interface via network_attachment.security_group_ids (list of security group IDs)
- User data and password are sensitive attributes
- Default timeout is 10 minutes, configurable via timeouts block
