# Tasks: Flavor and Network Data Sources

**Input**: Design documents from `/specs/002-flavor-network-datasources/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Acceptance tests are REQUIRED per TDD workflow in constitution. Tests must be written FIRST and FAIL before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and ensure provider scaffolding is ready

- [X] T001 Verify existing provider structure and dependencies in go.mod
- [X] T002 Verify provider.go DataSourcesMap is ready for new data source registration

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Verify VPS client initialization pattern in provider configuration (from provider.go)
- [X] T004 Verify SDK imports available: github.com/Zillaforge/cloud-sdk/models/vps/flavors
- [X] T005 Verify SDK imports available: github.com/Zillaforge/cloud-sdk/models/vps/networks

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Query Available Flavors (Priority: P1) 🎯 MVP

**Goal**: Implement zillaforge_flavors data source that queries and filters compute flavors by name, vcpus, and memory

**Independent Test**: Configure provider, define data source without filters, run terraform plan, verify list of flavors returned with all attributes (id, name, vcpus, memory, disk, description)

### Acceptance Tests for User Story 1 (TDD - Write FIRST)

> **CRITICAL: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T006 [P] [US1] Write acceptance test for basic flavors query (no filters) in internal/provider/flavor_data_source_test.go
- [X] T007 [P] [US1] Write acceptance test for name filter (exact match) in internal/provider/flavor_data_source_test.go
- [X] T008 [P] [US1] Write acceptance test for vcpus filter (minimum) in internal/provider/flavor_data_source_test.go
- [X] T009 [P] [US1] Write acceptance test for memory filter (minimum GB) in internal/provider/flavor_data_source_test.go
- [X] T010 [P] [US1] Write acceptance test for multiple filters (AND logic) in internal/provider/flavor_data_source_test.go
- [X] T011 [P] [US1] Write acceptance test for empty results (no match) in internal/provider/flavor_data_source_test.go
- [X] T012 [P] [US1] Write acceptance test for API authentication error in internal/provider/flavor_data_source_test.go
- [X] T013 [P] [US1] Write acceptance test for API error handling in internal/provider/flavor_data_source_test.go

**Checkpoint**: Run tests - ALL MUST FAIL (RED phase of TDD)

### Implementation for User Story 1

- [X] T014 [US1] Define FlavorDataSource struct in internal/provider/flavor_data_source.go
- [X] T015 [US1] Implement Metadata method for zillaforge_flavors in internal/provider/flavor_data_source.go
- [X] T016 [US1] Define FlavorDataSourceModel with filter and result attributes in internal/provider/flavor_data_source.go
- [X] T017 [US1] Define FlavorModel with computed attributes (id, name, vcpus, memory, disk, description) in internal/provider/flavor_data_source.go
- [X] T018 [US1] Implement Schema method with filter attributes (name, vcpus, memory) in internal/provider/flavor_data_source.go
- [X] T019 [US1] Implement Schema method with flavors list result attribute in internal/provider/flavor_data_source.go
- [X] T020 [US1] Add MarkdownDescription to all schema attributes in internal/provider/flavor_data_source.go
- [X] T021 [US1] Add int64validator.AtLeast(1) to vcpus and memory filters in internal/provider/flavor_data_source.go
- [X] T022 [US1] Implement Read method - get VPS client from provider config in internal/provider/flavor_data_source.go
- [X] T023 [US1] Implement Read method - call vpsClient.Flavors().List(ctx, nil) in internal/provider/flavor_data_source.go
- [X] T024 [US1] Implement Read method - handle SDK errors and convert to diagnostics in internal/provider/flavor_data_source.go
- [X] T025 [US1] Implement client-side name filter (exact match, case-sensitive) in internal/provider/flavor_data_source.go
- [X] T026 [US1] Implement client-side vcpus filter (minimum comparison) in internal/provider/flavor_data_source.go
- [X] T027 [US1] Implement client-side memory filter with MiB to GB conversion in internal/provider/flavor_data_source.go
- [X] T028 [US1] Implement AND logic for multiple filters in internal/provider/flavor_data_source.go
- [X] T029 [US1] Map SDK Flavor objects to FlavorModel with type conversions in internal/provider/flavor_data_source.go
- [X] T030 [US1] Handle null values for optional fields (disk, description) in internal/provider/flavor_data_source.go
- [X] T031 [US1] Set state with filtered results or empty list in internal/provider/flavor_data_source.go
- [X] T032 [US1] Register zillaforge_flavors in provider.go DataSourcesMap
- [X] T033 [US1] Create example configuration in examples/data-sources/zillaforge_flavor/data-source.tf
- [X] T034 [US1] Add examples for all filter combinations in examples/data-sources/zillaforge_flavor/data-source.tf

**Checkpoint**: Run acceptance tests - ALL MUST PASS (GREEN phase of TDD). User Story 1 is now fully functional and independently testable.

---

## Phase 4: User Story 2 - Query Available Networks (Priority: P1)

**Goal**: Implement zillaforge_networks data source that queries and filters networks by name and status

**Independent Test**: Configure provider, define data source without filters, run terraform plan, verify list of networks returned with all attributes (id, name, cidr, status, description)

### Acceptance Tests for User Story 2 (TDD - Write FIRST)

> **CRITICAL: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T035 [P] [US2] Write acceptance test for basic networks query (no filters) in internal/provider/network_data_source_test.go
- [X] T036 [P] [US2] Write acceptance test for name filter (exact match) in internal/provider/network_data_source_test.go
- [X] T037 [P] [US2] Write acceptance test for status filter in internal/provider/network_data_source_test.go
- [X] T038 [P] [US2] Write acceptance test for multiple filters (AND logic) in internal/provider/network_data_source_test.go
- [X] T039 [P] [US2] Write acceptance test for empty results (no match) in internal/provider/network_data_source_test.go
- [X] T040 [P] [US2] Write acceptance test for API authentication error in internal/provider/network_data_source_test.go
- [X] T041 [P] [US2] Write acceptance test for API error handling in internal/provider/network_data_source_test.go

**Checkpoint**: Run tests - ALL MUST FAIL (RED phase of TDD)

### Implementation for User Story 2

- [X] T042 [US2] Define NetworkDataSource struct in internal/provider/network_data_source.go
- [X] T043 [US2] Implement Metadata method for zillaforge_networks in internal/provider/network_data_source.go
- [X] T044 [US2] Define NetworkDataSourceModel with filter and result attributes in internal/provider/network_data_source.go
- [X] T045 [US2] Define NetworkModel with computed attributes (id, name, cidr, status, description) in internal/provider/network_data_source.go
- [X] T046 [US2] Implement Schema method with filter attributes (name, status) in internal/provider/network_data_source.go
- [X] T047 [US2] Implement Schema method with networks list result attribute in internal/provider/network_data_source.go
- [X] T048 [US2] Add MarkdownDescription to all schema attributes in internal/provider/network_data_source.go
- [X] T049 [US2] Implement Read method - get VPS client from provider config in internal/provider/network_data_source.go
- [X] T050 [US2] Implement Read method - call vpsClient.Networks().List(ctx, nil) in internal/provider/network_data_source.go
- [X] T051 [US2] Implement Read method - handle SDK errors and convert to diagnostics in internal/provider/network_data_source.go
- [X] T052 [US2] Implement client-side name filter (exact match, case-sensitive) in internal/provider/network_data_source.go
- [X] T053 [US2] Implement client-side status filter (exact match) in internal/provider/network_data_source.go
- [X] T054 [US2] Implement AND logic for multiple filters in internal/provider/network_data_source.go
- [X] T055 [US2] Map SDK Network objects to NetworkModel with type conversions in internal/provider/network_data_source.go
- [X] T056 [US2] Handle null values for optional field (description) in internal/provider/network_data_source.go
- [X] T057 [US2] Set state with filtered results or empty list in internal/provider/network_data_source.go
- [X] T058 [US2] Register zillaforge_networks in provider.go DataSourcesMap
- [X] T059 [US2] Create example configuration in examples/data-sources/zillaforge_network/data-source.tf
- [X] T060 [US2] Add examples for all filter combinations in examples/data-sources/zillaforge_network/data-source.tf

**Checkpoint**: Run acceptance tests - ALL MUST PASS (GREEN phase of TDD). User Story 2 is now fully functional and independently testable.

---

## Phase 5: User Story 3 - Reference Data Sources in Resources (Priority: P2)

**Goal**: Validate data source integration by creating examples that reference flavors and networks in resource configurations

**Independent Test**: Create Terraform config using flavor data source to populate instance flavor, network data source to populate network attachment, run terraform plan, verify correct attribute references

### Integration Examples for User Story 3

- [ ] T061 [P] [US3] Create integration example using flavors[0].id in resource reference in examples/data-sources/zillaforge_flavor/data-source.tf
- [ ] T062 [P] [US3] Create integration example using networks[0].id in resource reference in examples/data-sources/zillaforge_network/data-source.tf
- [ ] T063 [US3] Create combined example using both data sources in single config in examples/verification/main.tf
- [ ] T064 [US3] Add output examples demonstrating data source attribute access in examples/verification/main.tf
- [ ] T065 [US3] Document common integration patterns in quickstart.md

**Checkpoint**: All user stories are independently functional and integration patterns are validated

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, and final quality improvements

- [ ] T066 [P] Generate provider documentation using tfplugindocs tool
- [ ] T067 [P] Verify generated docs/data-sources/flavors.md matches schema
- [ ] T068 [P] Verify generated docs/data-sources/networks.md matches schema
- [ ] T069 Run quickstart.md validation - verify all examples execute successfully
- [ ] T070 Code review and refactoring for consistency across both data sources
- [ ] T071 Add godoc comments to all exported types and methods
- [ ] T072 Run go fmt and linting tools on all modified files
- [ ] T073 Update CHANGELOG.md with new data sources

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 and US2 can proceed in parallel (different files)
  - US3 depends on at least one of US1 or US2 being complete
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories (parallel with US1)
- **User Story 3 (P2)**: Requires at least US1 OR US2 complete - validates integration

### Within Each User Story

**TDD Workflow (CRITICAL)**:
1. Write ALL acceptance tests FIRST (T006-T013 for US1, T035-T041 for US2)
2. Run tests - verify they FAIL (RED)
3. Implement data source (T014-T034 for US1, T042-T060 for US2)
4. Run tests - verify they PASS (GREEN)
5. Refactor as needed while keeping tests passing

**Implementation Order**:
- Struct definitions before schema
- Schema definition before Read method
- Read method core logic before filtering
- Filter logic before state setting
- Provider registration after implementation complete
- Examples after functionality works

### Parallel Opportunities

**Phase 1 (Setup)**:
- Both verification tasks can run in parallel

**Phase 2 (Foundational)**:
- All SDK import verifications can run in parallel

**Phase 3 (User Story 1 - Tests)**:
- All acceptance tests T006-T013 can be written in parallel (same file, different test functions)

**Phase 4 (User Story 2 - Tests)**:
- All acceptance tests T035-T041 can be written in parallel (same file, different test functions)

**Phase 5 (User Story 3 - Integration)**:
- T061 and T062 can run in parallel (different directories)

**Phase 6 (Polish)**:
- T066, T067, T068, T071, T072 can all run in parallel (different files/tools)

**Cross-Story Parallelization**:
- After Foundational phase complete, entire US1 (Phase 3) and US2 (Phase 4) can proceed in parallel
- Two developers can work simultaneously:
  - Developer A: US1 tests → US1 implementation
  - Developer B: US2 tests → US2 implementation

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all acceptance tests for US1 together:
# All in same file (flavor_data_source_test.go) but different test functions

Task: "Write acceptance test for basic flavors query (no filters)"
Task: "Write acceptance test for name filter (exact match)"
Task: "Write acceptance test for vcpus filter (minimum)"
Task: "Write acceptance test for memory filter (minimum GB)"
Task: "Write acceptance test for multiple filters (AND logic)"
Task: "Write acceptance test for empty results (no match)"
Task: "Write acceptance test for API authentication error"
Task: "Write acceptance test for API error handling"
```

---

## Parallel Example: User Story 1 & 2 Together

```bash
# After Foundational phase, two developers work in parallel:

Developer A (US1 - Flavors):
- Write all US1 acceptance tests (T006-T013)
- Verify tests fail
- Implement flavor_data_source.go (T014-T031)
- Register in provider.go (T032)
- Create examples (T033-T034)
- Verify tests pass

Developer B (US2 - Networks):
- Write all US2 acceptance tests (T035-T041)
- Verify tests fail
- Implement network_data_source.go (T042-T057)
- Register in provider.go (T058)
- Create examples (T059-T060)
- Verify tests pass

# No file conflicts - complete independence
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup → 2 tasks
2. Complete Phase 2: Foundational → 3 tasks
3. Complete Phase 3: User Story 1 → 29 tasks (8 tests + 21 implementation)
4. **STOP and VALIDATE**: Run all US1 tests, verify flavors data source works
5. Deploy/demo if ready → **MVP Complete!**

**MVP Delivers**: Fully functional zillaforge_flavors data source with filtering

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready (5 tasks)
2. Add User Story 1 → Test independently → Deploy/Demo (29 tasks → **MVP**)
3. Add User Story 2 → Test independently → Deploy/Demo (26 tasks → **Feature Complete**)
4. Add User Story 3 → Validate integration → Deploy/Demo (5 tasks → **Validated**)
5. Polish → Final release (8 tasks → **Production Ready**)

**Total**: 73 tasks organized into 6 deliverable phases

### Parallel Team Strategy

With 2 developers after Foundational phase:

1. Team completes Setup + Foundational together (5 tasks)
2. **Split for parallel work**:
   - Developer A: User Story 1 (29 tasks)
   - Developer B: User Story 2 (26 tasks)
3. Rejoin: User Story 3 together (5 tasks)
4. Polish together (8 tasks)

**Time Savings**: ~55 tasks done in parallel instead of sequentially

---

## Notes

- **[P]** tasks = different files or independent test functions, no dependencies
- **[Story]** label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- **TDD is mandatory**: Verify tests fail before implementing (constitution requirement)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Both data sources follow identical patterns (different entities, same structure)
- No cross-story dependencies between US1 and US2 → perfect for parallel development

---

## Task Count Summary

- **Phase 1 (Setup)**: 2 tasks
- **Phase 2 (Foundational)**: 3 tasks
- **Phase 3 (US1 - Flavors)**: 29 tasks (8 tests + 21 implementation)
- **Phase 4 (US2 - Networks)**: 26 tasks (7 tests + 19 implementation)
- **Phase 5 (US3 - Integration)**: 5 tasks
- **Phase 6 (Polish)**: 8 tasks
- **Total**: 73 tasks

**Parallel Opportunities**: 41 tasks can run in parallel across different phases
**MVP Scope**: 34 tasks (Setup + Foundational + US1)
**Independent Stories**: US1 and US2 have zero dependencies (55 tasks parallelizable)
