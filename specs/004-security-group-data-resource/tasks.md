# Tasks: Security Group Data Source and Resource

**Input**: Design documents from `/specs/004-security-group-data-resource/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: TDD approach required per project constitution - all acceptance tests written first

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Verify cloud-sdk supports SecurityGroups() client (confirm API availability)
- [X] T002 [P] Review existing VPS resource patterns (keypair, flavor, network) for consistency
- [X] T003 [P] Create validator utilities directory internal/vps/validators/ if not exists

**Checkpoint**: Setup verified, patterns understood

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core validation and shared utilities that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 [P] Implement port range validator in internal/vps/validators/port_range.go
- [X] T005 [P] Implement CIDR validator in internal/vps/validators/cidr.go
- [X] T006 [P] Implement protocol validator in internal/vps/validators/protocol.go
- [X] T007 [P] Unit tests for port range validator (test "all", single port, ranges, invalid)
- [X] T008 [P] Unit tests for CIDR validator (test IPv4, IPv6, invalid formats)
- [X] T009 [P] Unit tests for protocol validator (test tcp/udp/icmp/any, case-insensitive)

**Checkpoint**: Foundation ready - validators complete, all user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Create and Manage Security Groups with Rules (Priority: P1) 🎯 MVP

**Goal**: Enable users to create, update, and delete security groups with inbound/outbound firewall rules through Terraform

**Independent Test**: Define a security group resource with ingress/egress rules in Terraform, apply it, verify in ZillaForge platform, update rules, verify updates, destroy and verify deletion

### Tests for User Story 1 (TDD - Write First, Ensure FAIL) ⚠️

- [X] T010 [P] [US1] Acceptance test: Create security group with ingress rules in internal/vps/resource/security_group_resource_test.go
- [X] T011 [P] [US1] Acceptance test: Create security group with egress rules in internal/vps/resource/security_group_resource_test.go
- [X] T012 [P] [US1] Acceptance test: Update security group description (in-place) in internal/vps/resource/security_group_resource_test.go
- [X] T013 [P] [US1] Acceptance test: Add rules to existing security group in internal/vps/resource/security_group_resource_test.go
- [X] T014 [P] [US1] Acceptance test: Remove rules from security group in internal/vps/resource/security_group_resource_test.go
- [X] T015 [P] [US1] Acceptance test: Modify rule attributes (replace rule) in internal/vps/resource/security_group_resource_test.go
- [X] T016 [P] [US1] Acceptance test: Delete security group (not attached) in internal/vps/resource/security_group_resource_test.go
- [ ] T017 [P] [US1] Acceptance test: Block deletion when attached to instances (handle 409) in internal/vps/resource/security_group_resource_test.go
- [X] T018 [P] [US1] Acceptance test: ForceNew on name change in internal/vps/resource/security_group_resource_test.go

**Checkpoint**: Run tests - ALL MUST FAIL (RED phase of TDD)

### Implementation for User Story 1

- [X] T019 [US1] Create SecurityGroupResourceModel in internal/vps/resource/security_group_resource.go
- [X] T020 [US1] Create SecurityRuleModel nested struct in internal/vps/resource/security_group_resource.go
- [X] T021 [US1] Implement Schema() with all attributes (id, name, description, ingress_rules, egress_rules) in internal/vps/resource/security_group_resource.go
- [X] T022 [US1] Add MarkdownDescription to all schema attributes in internal/vps/resource/security_group_resource.go
- [X] T023 [US1] Add validators to ingress_rules/egress_rules nested attributes (protocol, port_range, CIDR) in internal/vps/resource/security_group_resource.go
- [X] T024 [US1] Add plan modifiers (UseStateForUnknown for id, RequiresReplace for name) in internal/vps/resource/security_group_resource.go
- [X] T025 [US1] Implement Create() method with cloud-sdk SecurityGroups().Create() call in internal/vps/resource/security_group_resource.go
- [X] T026 [US1] Implement Read() method with cloud-sdk SecurityGroups().Get() call in internal/vps/resource/security_group_resource.go
- [X] T027 [US1] Implement Update() method (full rule replacement strategy) in internal/vps/resource/security_group_resource.go
- [X] T028 [US1] Implement Delete() method with 409 conflict handling (attached instances check) in internal/vps/resource/security_group_resource.go
- [X] T029 [US1] Add error handling for 409 with instance ID extraction and actionable diagnostics in internal/vps/resource/security_group_resource.go
- [X] T030 [US1] Implement Metadata() method in internal/vps/resource/security_group_resource.go
- [X] T031 [US1] Register resource in internal/provider/provider.go Resources() method
- [X] T032 [US1] Create resource example in examples/resources/zillaforge_security_group/resource.tf
- [X] T033 [US1] Add web server security group example (HTTP/HTTPS/SSH) in examples/resources/zillaforge_security_group/resource.tf

**Checkpoint**: Run acceptance tests - ALL MUST PASS (GREEN phase of TDD). User Story 1 is now fully functional and independently testable.

---

## Phase 4: User Story 2 - Query Existing Security Groups (Priority: P2)

**Goal**: Enable users to query existing security groups via data source for read-only access and referencing in configurations

**Independent Test**: Create a security group outside Terraform (via UI/API), query it by name and ID using data source, verify all attributes and rules are retrieved correctly; test list-all mode returns multiple security groups

### Tests for User Story 2 (TDD - Write First, Ensure FAIL) ⚠️

- [X] T034 [P] [US2] Acceptance test: Query security group by name in internal/vps/data/security_groups_data_source_test.go
- [X] T035 [P] [US2] Acceptance test: Query security group by ID in internal/vps/data/security_groups_data_source_test.go
- [X] T036 [P] [US2] Acceptance test: Error when querying non-existent name (returns empty list) in internal/vps/data/security_groups_data_source_test.go
- [X] T037 [P] [US2] Acceptance test: List all security groups (no filters) in internal/vps/data/security_groups_data_source_test.go
- [X] T038 [P] [US2] Acceptance test: Error when both name and ID filters specified in internal/vps/data/security_groups_data_source_test.go
- [X] T039 [P] [US2] Acceptance test: Verify all rule attributes returned (protocol, port_range, CIDR) in internal/vps/data/security_groups_data_source_test.go

**Checkpoint**: Run tests - ALL MUST FAIL (RED phase of TDD) ✅ COMPLETE

### Implementation for User Story 2

- [X] T040 [US2] Create SecurityGroupsDataSourceModel in internal/vps/data/security_groups_data_source.go
- [X] T041 [US2] Create SecurityGroupModel for results in internal/vps/data/security_groups_data_source.go
- [X] T042 [US2] Implement Schema() with filter attributes (id, name) and results (security_groups) in internal/vps/data/security_groups_data_source.go
- [X] T043 [US2] Add MarkdownDescription to all schema attributes in internal/vps/data/security_groups_data_source.go
- [X] T044 [US2] Add mutually exclusive validator for id/name filters in internal/vps/data/security_groups_data_source.go
- [X] T045 [US2] Implement Read() method with filter logic (id, name, list-all) in internal/vps/data/security_groups_data_source.go
- [X] T046 [US2] Add cloud-sdk SecurityGroups().Get() call for ID filter in internal/vps/data/security_groups_data_source.go
- [X] T047 [US2] Add cloud-sdk SecurityGroups().List() call with Detail option for name filter and list-all modes in internal/vps/data/security_groups_data_source.go
- [X] T048 [US2] Implement client-side name filtering if SDK List() doesn't support it in internal/vps/data/security_groups_data_source.go
- [X] T049 [US2] Handle pagination if API supports it in internal/vps/data/security_groups_data_source.go (N/A - pagination not required)
- [X] T050 [US2] Implement Metadata() method in internal/vps/data/security_groups_data_source.go
- [X] T051 [US2] Register data source in internal/provider/provider.go DataSources() method
- [X] T052 [US2] Create data source example (query by name) in examples/data-sources/zillaforge_security_groups/data-source.tf
- [X] T053 [US2] Add data source example (query by ID) in examples/data-sources/zillaforge_security_groups/data-source.tf
- [X] T054 [US2] Add data source example (list all) in examples/data-sources/zillaforge_security_groups/data-source.tf

**Checkpoint**: Run acceptance tests - ALL MUST PASS (GREEN phase of TDD). User Story 2 is now fully functional and independently testable.

---

## Phase 5: User Story 3 - Import Existing Security Groups (Priority: P3)

**Goal**: Enable users to import manually-created security groups into Terraform state for management

**Independent Test**: Create a security group manually (with rules) via UI/API, run terraform import, verify state contains all attributes and rules, verify subsequent plan shows no changes

### Tests for User Story 3 (TDD - Write First, Ensure FAIL) ⚠️

- [X] T055 [P] [US3] Acceptance test: Import security group by ID in internal/vps/resource/security_group_resource_test.go
- [X] T056 [P] [US3] Acceptance test: Plan after import shows no changes (matching config) in internal/vps/resource/security_group_resource_test.go
- [X] T057 [P] [US3] Acceptance test: Plan after import detects drift (config mismatch) in internal/vps/resource/security_group_resource_test.go
- [X] T058 [P] [US3] Acceptance test: Error on invalid import ID format in internal/vps/resource/security_group_resource_test.go

**Checkpoint**: Run tests - ALL MUST FAIL (RED phase of TDD)

### Implementation for User Story 3

- [X] T059 [US3] Implement ImportState() method in internal/vps/resource/security_group_resource.go
- [X] T060 [US3] Validate import ID is valid UUID format in internal/vps/resource/security_group_resource.go
- [X] T061 [US3] Call Read() method to populate state after import in internal/vps/resource/security_group_resource.go
- [X] T062 [US3] Handle import errors (not found, invalid ID) with clear diagnostics in internal/vps/resource/security_group_resource.go
- [X] T063 [US3] Create import example script in examples/resources/zillaforge_security_group/import.sh
- [X] T064 [US3] Document import workflow in examples/resources/zillaforge_security_group/import.sh

**Checkpoint**: Run acceptance tests - ALL MUST PASS (GREEN phase of TDD). User Story 3 is now fully functional and independently testable.

---

## Phase 6: User Story 4 - Reference Security Groups Between Resources (Priority: P2)

**Goal**: Enable referencing security groups in VPS instance configurations (future VPS resource implementation)

**Independent Test**: This story depends on VPS instance resource (future work). For now, verify data source and resource expose `id` attribute correctly for referencing.

### Documentation for User Story 4

- [X] T065 [US4] Add example of referencing security group ID in future VPS instance resource in examples/resources/zillaforge_security_group/resource.tf
- [X] T066 [US4] Document security group attachment pattern in quickstart.md (placeholder for future instance resource)
- [X] T067 [US4] Verify resource exports `id` attribute (already done in US1, this is verification)

**Note**: Full implementation of US4 requires VPS instance resource (not in scope for this feature)

**Checkpoint**: Documentation complete, `id` attribute verified for future referencing

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, and quality improvements across all user stories

- [X] T068 Generate provider documentation with tfplugindocs
- [X] T069 Validate all examples in examples/ directory execute successfully
- [X] T070 Add comprehensive MarkdownDescription to resource and data source schemas
- [X] T071 Run go fmt and linting tools on all modified files
- [X] T072 Update CHANGELOG.md with new resource and data source
- [X] T073 Create documentation in docs/resources/security_group.md
- [X] T074 Create documentation in docs/data-sources/security_groups.md
- [X] T075 Review error messages for actionable diagnostics (especially 409 conflict)
- [X] T076 Add logging statements for debugging (Create, Read, Update, Delete operations)

**Checkpoint**: All user stories are independently functional, documented, and polished

---

## Dependencies

### Phase Order (Sequential)

- **Setup (Phase 1)**: Independent, can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1, US2, US3 can be implemented in parallel after Phase 2
  - US4 is documentation-only (VPS instance resource not in scope)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Completion Order

Per spec.md priorities:
1. **US1 (P1)** - MVP: Create and manage security groups
2. **US2 (P2)** - Query existing security groups
3. **US3 (P3)** - Import existing security groups
4. **US4 (P2)** - Reference between resources (documentation only)

**Recommended MVP Delivery**: Complete US1 only for initial release

---

## Parallel Execution Opportunities

### Within Phase 2 (Foundational)

- T004, T005, T006 can run in parallel (different validator files)
- T007, T008, T009 can run in parallel (different test files)

### Within Phase 3 (User Story 1 - Tests)

- All test tasks T010-T018 can be written in parallel (same file, different test functions)

### Within Phase 3 (User Story 1 - Implementation)

After schema (T019-T024) complete:
- T025, T026, T027 can run in parallel (different CRUD methods in same file, minimal conflicts)
- T032, T033 can run in parallel with implementation (different files)

### Within Phase 4 (User Story 2 - Tests)

- All test tasks T034-T039 can be written in parallel (same file, different test functions)

### Within Phase 4 (User Story 2 - Implementation)

- T040-T044 (schema) must complete first
- T045-T049 can run in parallel (different filter branches in Read method)
- T052-T054 can run in parallel (different example files)

### Within Phase 5 (User Story 3 - Tests)

- All test tasks T055-T058 can be written in parallel (same file, different test functions)

### Within Phase 7 (Polish)

- T068, T069, T071, T072, T073, T074, T075, T076 can all run in parallel (different files/tools)

---

## Example Parallel Execution

```bash
# Phase 2: Launch all validator implementations together
# Developers can work on different files simultaneously
Terminal 1: implement T004 (port_range.go)
Terminal 2: implement T005 (cidr.go)
Terminal 3: implement T006 (protocol.go)

# Phase 3: Launch all acceptance tests for User Story 1 together
# All test functions can be written simultaneously in security_group_resource_test.go
Terminal 1: write T010, T011, T012
Terminal 2: write T013, T014, T015
Terminal 3: write T016, T017, T018

# Phase 3: After schema complete, CRUD methods in parallel
Terminal 1: implement T025 (Create)
Terminal 2: implement T026 (Read)
Terminal 3: implement T027 (Update)
# Note: T028 (Delete) should wait for Read to be complete for testing
```

---

## Implementation Strategy

### TDD Workflow (REQUIRED per constitution)

For each user story:

1. Write all acceptance tests first (T010-T018 for US1, T034-T039 for US2, etc.)
2. Run tests, verify FAIL
3. Implement schema (T019-T024 for US1)
4. Implement CRUD methods (T025-T028 for US1)
5. Verify all tests PASS
6. Refactor if needed
7. Add examples and documentation

### MVP First Approach

- **Phase 1-3 only**: Delivers US1 (create/manage security groups) - fully functional MVP
- **Add Phase 4**: Adds data source querying capability
- **Add Phase 5**: Adds import capability
- **Phase 6**: Documentation for future integration
- **Phase 7**: Polish and documentation

### Incremental Delivery

Each phase delivers value:
- After Phase 3: Users can create/manage security groups via Terraform
- After Phase 4: Users can also query existing security groups
- After Phase 5: Users can also import existing security groups
- After Phase 7: Fully polished, production-ready feature

---

## Task Summary

- **Total tasks**: 76
- **Setup (Phase 1)**: 3 tasks
- **Foundational (Phase 2)**: 6 tasks
- **User Story 1 (Phase 3)**: 25 tasks (9 tests + 15 implementation + 1 checkpoint)
- **User Story 2 (Phase 4)**: 21 tasks (6 tests + 15 implementation)
- **User Story 3 (Phase 5)**: 10 tasks (4 tests + 6 implementation)
- **User Story 4 (Phase 6)**: 3 tasks (documentation only)
- **Polish (Phase 7)**: 9 tasks

### Test Coverage

- **Acceptance tests**: 19 test tasks (T010-T018, T034-T039, T055-T058)
- **Unit tests**: 3 validator test tasks (T007-T009)
- **Total test tasks**: 22

### Parallel Opportunities

- **Phase 2**: 3 validators + 3 test files = 6 parallel tasks
- **US1 tests**: 9 test functions parallel
- **US2 tests**: 6 test functions parallel
- **US3 tests**: 4 test functions parallel
- **Total parallel opportunities**: ~30 tasks can run concurrently at various stages

---

## Critical Path

**Blocking sequence**:
1. Phase 1 (Setup) → Phase 2 (Foundational validators)
2. Phase 2 → US1 Schema (T019-T024)
3. US1 Schema → US1 CRUD (T025-T028)
4. US1 complete → US2 can start
5. US2 complete → US3 can start
6. US1, US2, US3 complete → Phase 7 (Polish)

**Time estimate** (rough):
- Setup + Foundational: ~2 days (validators and tests)
- US1: ~5 days (TDD with 9 tests + resource implementation + examples)
- US2: ~3 days (TDD with 6 tests + data source implementation)
- US3: ~2 days (TDD with 4 tests + import implementation)
- US4: ~0.5 day (documentation only)
- Polish: ~1 day
- **Total**: ~13.5 days (assuming sequential execution)
- **With parallelization**: ~8-10 days (tests written in parallel, some implementation parallel)

---

## Notes

- **TDD REQUIRED**: All acceptance tests MUST be written first and FAIL before implementation (project constitution)
- **Validator reuse**: Validators from Phase 2 are used across resource and data source
- **Error handling**: 409 conflict handling for Delete operation is critical (FR-007)
- **File Paths**: All file paths are absolute from repository root
- **Cloud-SDK dependency**: Verify cloud-sdk SecurityGroups() API availability (T001)
- **Schema consistency**: Follow existing VPS resource patterns (keypairs, flavors, networks)
- **Error messages**: All errors must be actionable per constitution Principle III
- **Documentation**: MarkdownDescription on all attributes for tfplugindocs generation
