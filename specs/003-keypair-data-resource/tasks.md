---
description: "Implementation tasks for keypair data source and resource"
---

# Tasks: Keypair Data Source and Resource

**Input**: Design documents from `/specs/003-keypair-data-resource/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: TDD approach required per project constitution - all acceptance tests written first

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Review existing provider patterns in internal/vps/data/flavor_data_source.go and internal/vps/data/network_data_source.go
- [X] T002 Verify cloud-sdk keypairs module integration per research.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Create keypair package directory structure internal/vps/data/ and internal/vps/resource/
- [X] T004 [P] Create KeypairDataSourceModel struct in internal/vps/data/keypair_data_source.go
- [X] T005 [P] Create KeypairResourceModel struct in internal/vps/resource/keypair_resource.go
- [X] T006 Register keypair data source in internal/provider/provider.go DataSources() method
- [X] T007 Register keypair resource in internal/provider/provider.go Resources() method

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Create and Manage SSH Keypairs (Priority: P1) 🎯 MVP

**Goal**: Enable infrastructure engineers to create and manage SSH keypairs through Terraform for secure VPS instance access

**Independent Test**: Define a keypair resource in Terraform configuration, apply it, verify keypair exists in ZillaForge with correct properties; test destroy removes it

### Tests for User Story 1 (TDD - Write First, Must Fail)

- [X] T008 [P] [US1] Acceptance test: Create keypair with user-provided public key in internal/vps/resource/keypair_resource_test.go
- [X] T009 [P] [US1] Acceptance test: Create keypair with system-generated keys in internal/vps/resource/keypair_resource_test.go
- [X] T010 [P] [US1] Acceptance test: Delete keypair successfully in internal/vps/resource/keypair_resource_test.go
- [X] T011 [P] [US1] Acceptance test: Change immutable fields triggers replacement in internal/vps/resource/keypair_resource_test.go
- [X] T012 [P] [US1] Acceptance test: Duplicate keypair name returns error in internal/vps/resource/keypair_resource_test.go
- [X] T013 [P] [US1] Acceptance test: Invalid public key format returns error in internal/vps/resource/keypair_resource_test.go

### Implementation for User Story 1

- [X] T014 [US1] Implement Schema() method for keypair resource in internal/vps/resource/keypair_resource.go
- [X] T015 [US1] Implement Metadata() method for keypair resource in internal/vps/resource/keypair_resource.go
- [X] T016 [US1] Implement Configure() method for keypair resource in internal/vps/resource/keypair_resource.go
- [X] T017 [US1] Implement Create() method with cloud-sdk Create() call in internal/vps/resource/keypair_resource.go
- [X] T018 [US1] Implement Read() method with cloud-sdk Get() call in internal/vps/resource/keypair_resource.go
- [X] T019 [US1] Implement Update() method for description-only updates in internal/vps/resource/keypair_resource.go
- [X] T020 [US1] Implement Delete() method with warning logging in internal/vps/resource/keypair_resource.go
- [X] T021 [US1] Add private_key sensitive attribute handling in Create() method
- [X] T022 [US1] Add RequiresReplace plan modifiers for name and public_key attributes
- [X] T023 [US1] Add error handling with actionable messages for duplicate name, invalid key format
- [X] T024 [US1] Create resource example in examples/resources/zillaforge_keypair/resource.tf
- [ ] T025 [US1] Run acceptance tests and verify all pass

**Checkpoint**: At this point, User Story 1 should be fully functional - users can create, manage, and delete keypairs through Terraform

---

## Phase 4: User Story 2 - Query Existing Keypairs (Priority: P2)

**Goal**: Enable infrastructure engineers to query existing keypairs through a data source for read-only reference in Terraform configurations

**Independent Test**: Create a keypair outside Terraform, query it via data source by name and ID, verify all attributes are correctly retrieved; test list-all mode returns multiple keypairs

### Tests for User Story 2 (TDD - Write First, Must Fail)

- [X] T026 [P] [US2] Acceptance test: Query keypair by name in internal/vps/data/keypair_data_source_test.go
- [X] T027 [P] [US2] Acceptance test: Query keypair by ID in internal/vps/data/keypair_data_source_test.go
- [X] T028 [P] [US2] Acceptance test: List all keypairs (no filters) in internal/vps/data/keypair_data_source_test.go
- [X] T029 [P] [US2] Acceptance test: Both name and ID filters return validation error in internal/vps/data/keypair_data_source_test.go
- [X] T030 [P] [US2] Acceptance test: Non-existent ID returns error in internal/vps/data/keypair_data_source_test.go
- [X] T031 [P] [US2] Acceptance test: Non-existent name returns empty list in internal/vps/data/keypair_data_source_test.go

### Implementation for User Story 2

- [X] T032 [US2] Implement Schema() method for keypair data source in internal/vps/data/keypair_data_source.go
- [X] T033 [US2] Implement Metadata() method for keypair data source in internal/vps/data/keypair_data_source.go
- [X] T034 [US2] Implement Configure() method for keypair data source in internal/vps/data/keypair_data_source.go
- [X] T035 [US2] Implement Read() method with mutual exclusivity validation in internal/vps/data/keypair_data_source.go
- [X] T036 [US2] Add cloud-sdk Get() call for ID filter mode in Read() method
- [X] T037 [US2] Add cloud-sdk List() call for name filter and list-all modes in Read() method
- [X] T038 [US2] Implement keypairToModel() conversion helper in internal/vps/data/keypair_data_source.go
- [X] T039 [US2] Add error handling for not-found by ID scenario
- [X] T040 [US2] Create data source example in examples/data-sources/zillaforge_keypairs/data-source.tf
- [ ] T041 [US2] Run acceptance tests and verify all pass

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - create/manage resources and query via data source

---

## Phase 5: User Story 3 - Import Existing Keypairs (Priority: P3)

**Goal**: Enable infrastructure engineers to import manually-created keypairs into Terraform state for version control

**Independent Test**: Create a keypair manually via API/UI, import it into Terraform state, verify subsequent plan operations correctly detect drift

### Tests for User Story 3 (TDD - Write First, Must Fail)

- [X] T042 [P] [US3] Acceptance test: Import keypair by ID in internal/vps/resource/keypair_resource_test.go
- [X] T043 [P] [US3] Acceptance test: Imported keypair shows no changes on plan in internal/vps/resource/keypair_resource_test.go
- [X] T044 [P] [US3] Acceptance test: Non-existent import ID returns error in internal/vps/resource/keypair_resource_test.go
- [X] T045 [P] [US3] Acceptance test: Imported keypair private_key is null in internal/vps/resource/keypair_resource_test.go

### Implementation for User Story 3

- [X] T046 [US3] Implement ImportState() method in internal/vps/resource/keypair_resource.go
- [X] T047 [US3] Add cloud-sdk Get() call for import operation in ImportState() method
- [X] T048 [US3] Handle private_key as null for imported keypairs (not available from API)
- [X] T049 [US3] Add error handling for import not-found scenario
- [X] T050 [US3] Create import example script in examples/resources/zillaforge_keypair/import.sh
- [X] T051 [US3] Run acceptance tests and verify all pass

**Checkpoint**: All user stories should now be independently functional - create, query, and import keypairs

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, and quality improvements across all user stories

- [X] T052 [P] Generate documentation with tfplugindocs for docs/data-sources/keypairs.md
- [X] T053 [P] Generate documentation with tfplugindocs for docs/resources/keypair.md
- [X] T054 Validate all examples in examples/ directory execute successfully
- [ ] T055 Run full acceptance test suite (TF_ACC=1 go test ./...)
- [ ] T056 Update CHANGELOG.md with new data source and resource
- [ ] T057 Run quickstart.md validation workflow
- [ ] T058 Verify constitution compliance (TDD, error messages, performance)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (different files) if team capacity allows
  - Or sequentially in priority order: US1 (P1) → US2 (P2) → US3 (P3)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Independent of US1 (different file: data source vs resource)
- **User Story 3 (P3)**: Can start after US1 complete (extends same resource file for ImportState method)

### Within Each User Story

**TDD Workflow (REQUIRED)**:
1. Write all acceptance tests for story (T008-T013, T026-T031, T042-T045)
2. Run tests - verify they FAIL (no implementation yet)
3. Implement minimal code to pass first test
4. Run tests - verify first test PASSES
5. Repeat step 3-4 for each subsequent test
6. All tests passing = story complete

**Implementation Order**:
- Tests MUST be written first and FAIL before implementation
- Schema/Metadata/Configure before CRUD methods
- Create before Read (Create generates resources for Read to test)
- Read before Update/Delete (Update/Delete need Read to verify)
- ImportState after all CRUD methods complete

### Parallel Opportunities

**Phase 1 (Setup)**: Both tasks can run in parallel (T001, T002)

**Phase 2 (Foundational)**: T004, T005 can run in parallel (different files)

**Phase 3 (User Story 1 - Tests)**: All test tasks T008-T013 can be written in parallel (same file, different test functions)

**Phase 4 (User Story 2 - Tests)**: All test tasks T026-T031 can be written in parallel (same file, different test functions)

**Phase 5 (User Story 3 - Tests)**: All test tasks T042-T045 can be written in parallel (same file, different test functions)

**Phase 6 (Polish)**: T052, T053 can run in parallel (different files)

**User Story Parallelization**:
- Once Phase 2 complete: US1 and US2 can start in parallel (different files: resource vs data source)
- US3 must wait for US1 (extends same resource file)

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all acceptance tests for User Story 1 together:
# All test functions can be written simultaneously in internal/vps/resource/keypair_resource_test.go

Test: "TestAccKeypairResource_UserProvidedKey" (T008)
Test: "TestAccKeypairResource_SystemGenerated" (T009)
Test: "TestAccKeypairResource_Delete" (T010)
Test: "TestAccKeypairResource_RequiresReplace" (T011)
Test: "TestAccKeypairResource_DuplicateName" (T012)
Test: "TestAccKeypairResource_InvalidPublicKey" (T013)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T002)
2. Complete Phase 2: Foundational (T003-T007) - CRITICAL BLOCKER
3. Complete Phase 3: User Story 1 (T008-T025)
   - Write all tests first (T008-T013), verify FAIL
   - Implement resource (T014-T023)
   - Verify all tests PASS
   - Add examples (T024)
   - Final validation (T025)
4. **STOP and VALIDATE**: Test User Story 1 independently with examples
5. Deploy/demo if ready - users can now manage keypairs via Terraform

### Incremental Delivery

1. Complete Setup + Foundational (T001-T007) → Foundation ready
2. Add User Story 1 (T008-T025) → Test independently → Deploy/Demo (MVP! ✅)
3. Add User Story 2 (T026-T041) → Test independently → Deploy/Demo (Query feature added ✅)
4. Add User Story 3 (T042-T051) → Test independently → Deploy/Demo (Import feature added ✅)
5. Polish (T052-T058) → Final validation → Production release

Each story adds value without breaking previous stories.

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (T001-T007)
2. Once Foundational is done:
   - **Developer A**: User Story 1 (T008-T025) - Keypair resource
   - **Developer B**: User Story 2 (T026-T041) - Keypair data source (can start in parallel!)
3. **Developer A or B**: User Story 3 (T042-T051) - Import (after US1 complete)
4. Team: Polish phase together (T052-T058)

Stories US1 and US2 can be developed completely in parallel by different developers since they touch different files.

---

## Summary Statistics

- **Total Tasks**: 58
- **User Story 1 (P1 - MVP)**: 18 tasks (6 tests + 12 implementation)
- **User Story 2 (P2)**: 16 tasks (6 tests + 10 implementation)
- **User Story 3 (P3)**: 10 tasks (4 tests + 6 implementation)
- **Setup + Foundational**: 7 tasks
- **Polish**: 7 tasks
- **Parallel Opportunities**: 18 tasks can run in parallel (marked with [P])
- **TDD Coverage**: 16 acceptance test tasks covering all user stories

---

## Notes

- **[P] marker**: Tasks with different files or no dependencies on incomplete work
- **[Story] label**: Maps task to specific user story (US1, US2, US3) for traceability
- **TDD REQUIRED**: All acceptance tests MUST be written first and FAIL before implementation (project constitution)
- **Independent Stories**: Each user story should be independently completable and testable
- **Checkpoint Validation**: Stop at each checkpoint to validate story works independently
- **File Paths**: All file paths are absolute from repository root
- **Pattern Consistency**: Follow existing flavor_data_source.go and network_data_source.go patterns
- **cloud-sdk Integration**: Use github.com/Zillaforge/cloud-sdk/modules/vps/keypairs client
- **Sensitive Data**: private_key attribute MUST be marked Sensitive in schema
- **Immutability**: name and public_key require RequiresReplace plan modifiers
- **Error Messages**: All errors must be actionable per constitution Principle III

---

## Suggested MVP Scope

**Minimum Viable Product** = Phase 1 + Phase 2 + Phase 3 (User Story 1 only)

This delivers:
- ✅ Create SSH keypairs (user-provided or system-generated)
- ✅ Update keypair descriptions
- ✅ Delete keypairs
- ✅ Private key handling (sensitive, returned once)
- ✅ Validation (duplicate names, invalid formats)
- ✅ Full TDD test coverage

**Deploy MVP first**, then incrementally add:
- User Story 2: Query/data source capabilities
- User Story 3: Import existing keypairs
- Polish: Documentation and final validation

This approach delivers value early while maintaining quality and independence of features.
