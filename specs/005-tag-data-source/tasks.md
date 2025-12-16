---

description: "Task list for Image Data Source implementation"
---

# Tasks: Image Data Source (zillaforge_images)

**Input**: Design documents from `/specs/005-tag-data-source/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/images-data-source-schema.md

**Tests**: Tests are REQUIRED per TDD mandate in spec.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create feature branch `005-tag-data-source` and switch to it
- [X] T002 Create base files for images data source: `internal/vrm/data/images_data_source.go` and `internal/vrm/data/images_data_source_test.go`
- [X] T003 [P] Create example directory and file: `examples/data-sources/zillaforge_images/data-source.tf`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Define `ImagesDataSourceModel` and `ImageModel` structs in `internal/vrm/data/images_data_source.go`
- [X] T005 [P] Implement `NewImagesDataSource()` factory function in `internal/vrm/data/images_data_source.go`
- [X] T006 [P] Implement `Metadata()` method to set TypeName to `zillaforge_images` in `internal/vrm/data/images_data_source.go`
- [X] T007 Implement `Schema()` method with all attributes per contracts/images-data-source-schema.md in `internal/vrm/data/images_data_source.go`
- [X] T008 [P] Implement `Configure()` method to receive provider client in `internal/vrm/data/images_data_source.go`
- [X] T009 Register `zillaforge_images` data source in provider `DataSources()` method in `internal/provider/provider.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Query Image Tags in Repository (Priority: P1) 🎯 MVP

**Goal**: Users can query available images from a ZillaForge repository to discover and reference specific image versions when creating virtual machines

**Independent Test**: Can be fully tested by configuring the provider with valid credentials, writing a data source block that queries image tags for a specific repository, running `terraform plan`, and verifying that the data source returns a list of available images with their attributes

### Tests for User Story 1 (RED Phase - Write First)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T010 [P] [US1] Write acceptance test `TestAccImagesDataSource_RepositoryAndTag` in `internal/vrm/data/images_data_source_test.go` (both filters specified)
- [X] T011 [P] [US1] Write acceptance test `TestAccImagesDataSource_RepositoryOnly` in `internal/vrm/data/images_data_source_test.go` (repository filter only)
- [X] T012 [P] [US1] Write acceptance test `TestAccImagesDataSource_TagOnly` in `internal/vrm/data/images_data_source_test.go` (tag filter only)
- [X] T013 [P] [US1] Write acceptance test `TestAccImagesDataSource_NoFilters` in `internal/vrm/data/images_data_source_test.go` (no filters)
- [X] T014 [P] [US1] Write acceptance test `TestAccImagesDataSource_AttributeReference` in `internal/vrm/data/images_data_source_test.go` (verify all attributes including id)
- [X] T015 [US1] Run acceptance tests to verify they FAIL (RED) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource'`

### Implementation for User Story 1 (GREEN Phase)

- [X] T016 [US1] Implement `Read()` method skeleton with mutual exclusivity validation in `internal/vrm/data/images_data_source.go`
- [X] T017 [US1] Implement API call logic per FR-022: use repository-scoped listing (`Repositories().Get()/List()` + `RepositoryResource.Tags().List()`) when `repository` filter is provided, otherwise use `vrmClient.Tags().List()` in `internal/vrm/data/images_data_source.go`
- [X] T018 [US1] Implement client-side filtering for repository name in `internal/vrm/data/images_data_source.go`
- [X] T019 [US1] Implement client-side filtering for exact tag name in `internal/vrm/data/images_data_source.go`
- [X] T020 [US1] Implement deterministic sorting (repository_name asc, tag_name asc) in `internal/vrm/data/images_data_source.go`
- [X] T021 [US1] Implement Tag to ImageModel conversion helper function in `internal/vrm/data/images_data_source.go`
- [X] T022 [US1] Set `images` list attribute with converted models in `internal/vrm/data/images_data_source.go`
- [X] T023 [US1] Implement error handling for API failures with clear diagnostics in `internal/vrm/data/images_data_source.go`
- [X] T023b [US1] Add unit test or code comment verifying repository-scoped API is used when repository filter is provided (validates FR-022 optimization) in `internal/vrm/data/images_data_source_test.go`
- [X] T024 [US1] Run acceptance tests to verify they PASS (GREEN) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource'`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Filter Images by Pattern (Priority: P2)

**Goal**: Users can filter image tags using pattern matching to discover tags following specific naming conventions (e.g., semantic versioning, environment prefixes)

**Independent Test**: Can be tested by creating a repository with tags following different patterns, then using pattern filters to query subsets, and verifying only matching tags are returned

### Tests for User Story 2 (RED Phase - Write First)

- [X] T025 [P] [US2] Write acceptance test `TestAccImagesDataSource_TagPattern_Wildcard` in `internal/vrm/data/images_data_source_test.go` (pattern with * wildcard)
- [X] T026 [P] [US2] Write acceptance test `TestAccImagesDataSource_TagPattern_SemanticVersioning` in `internal/vrm/data/images_data_source_test.go` (v1.* pattern)
- [X] T027 [P] [US2] Write acceptance test `TestAccImagesDataSource_TagPattern_EnvironmentPrefix` in `internal/vrm/data/images_data_source_test.go` (prod-* pattern)
- [X] T028 [P] [US2] Write acceptance test `TestAccImagesDataSource_TagPattern_NoMatches` in `internal/vrm/data/images_data_source_test.go` (pattern with no matches returns empty list)
- [X] T029 [US2] Run pattern tests to verify they FAIL (RED) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource_TagPattern'`

### Implementation for User Story 2 (GREEN Phase)

- [X] T030 [US2] Implement glob pattern matching using `filepath.Match()` in `internal/vrm/data/images_data_source.go`
- [X] T031 [US2] Integrate pattern filter into tag filtering logic in `internal/vrm/data/images_data_source.go`
- [X] T032 [US2] Add error handling for invalid glob patterns in `internal/vrm/data/images_data_source.go`
- [X] T033 [US2] Run pattern tests to verify they PASS (GREEN) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource_TagPattern'`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Reference Image Attributes in VM Provisioning (Priority: P2)

**Goal**: Users can reference image tag attribute `id` in their resource configurations to create virtual machines

**Independent Test**: Can be tested by creating a Terraform configuration that uses an image data source to populate a VM resource with an image `id`, then verifying the VM is provisioned using the correct image identifier

### Tests for User Story 3 (RED Phase - Write First)

- [ ] T034 [P] [US3] Write acceptance test `TestAccImagesDataSource_VMProvisioning_IDReference` in `internal/vrm/data/images_data_source_test.go` (verify id attribute exists and is usable)
- [ ] T035 [P] [US3] Write acceptance test `TestAccImagesDataSource_SortingNewest` in `internal/vrm/data/images_data_source_test.go` (verify deterministic sort order)
- [ ] T036 [P] [US3] Write acceptance test `TestAccImagesDataSource_SizeAttribute` in `internal/vrm/data/images_data_source_test.go` (verify size metadata)
- [ ] T037 [P] [US3] Write acceptance test `TestAccImagesDataSource_EmptyResults_ErrorHandling` in `internal/vrm/data/images_data_source_test.go` (verify error when data source is empty and referenced)
- [ ] T038 [US3] Run VM provisioning tests to verify they FAIL (RED) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource_(VMProvisioning|Sorting|Size|EmptyResults)'`

### Implementation for User Story 3 (GREEN Phase)

- [ ] T039 [US3] Ensure all image attributes are correctly exposed (id, repository_name, tag_name, size, operating_system, description, type, status) in `internal/vrm/data/images_data_source.go`
- [ ] T040 [US3] Add validation that id attribute is always set for each image in `internal/vrm/data/images_data_source.go`
- [ ] T041 [US3] Run VM provisioning tests to verify they PASS (GREEN) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource_(VMProvisioning|Sorting|Size|EmptyResults)'`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Edge Cases & Validation

**Purpose**: Handle edge cases and validation scenarios

### Tests for Edge Cases (RED Phase - Write First)

- [X] T042 [P] Write acceptance test `TestAccImagesDataSource_NonExistentRepository` in `internal/vrm/data/images_data_source_test.go` (returns empty list)
- [X] T043 [P] Write acceptance test `TestAccImagesDataSource_MutualExclusivity` in `internal/vrm/data/images_data_source_test.go` (tag and tag_pattern cannot both be specified)
- [X] T044 [P] Write acceptance test `TestAccImagesDataSource_SpecialCharacters` in `internal/vrm/data/images_data_source_test.go` (handle special characters in tag names)
- [X] T044b [P] Write acceptance test `TestAccImagesDataSource_AuthenticationError` in `internal/vrm/data/images_data_source_test.go` (verify authentication errors are distinct from empty results with appropriate diagnostic severity)
- [X] T045 Run edge case tests to verify they FAIL (RED) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource_(NonExistent|MutualExclusivity|SpecialCharacters)'`

### Implementation for Edge Cases (GREEN Phase)

- [X] T046 Ensure mutual exclusivity validation returns clear error message in `internal/vrm/data/images_data_source.go`
- [X] T047 Ensure empty results return empty list (not error) for non-existent repository in `internal/vrm/data/images_data_source.go`
- [X] T048 Add tag name validation per image repository naming standards in `internal/vrm/data/images_data_source.go` (N/A - read-only data source accepts API responses as-is)
- [X] T049 Run edge case tests to verify they PASS (GREEN) - `cd /workspaces/terraform-provider-zillaforge && make testacc TESTARGS='-run=TestAccImagesDataSource_(NonExistent|MutualExclusivity|SpecialCharacters)'`

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T050 [P] Create example configuration in `examples/data-sources/zillaforge_images/data-source.tf` with all filter scenarios
- [ ] T051 [P] Generate provider documentation - `cd /workspaces/terraform-provider-zillaforge && make generate`
- [ ] T052 [P] Verify generated documentation in `docs/data-sources/images.md`
- [ ] T053 Code cleanup and refactoring for readability in `internal/vrm/data/images_data_source.go`
- [ ] T054 Add inline code comments for complex filtering logic in `internal/vrm/data/images_data_source.go`
- [ ] T055 Run full acceptance test suite - `cd /workspaces/terraform-provider-zillaforge && make testacc`
- [ ] T056 Verify quickstart.md validation steps work correctly
- [ ] T057 Update CHANGELOG.md with new `zillaforge_images` data source feature

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Edge Cases (Phase 6)**: Depends on User Story 1 completion (basic Read implementation needed)
- **Polish (Phase 7)**: Depends on all user stories and edge cases being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Extends US1 filtering logic but independently testable
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Validates US1 attributes but independently testable

### Within Each User Story

- Tests (RED) MUST be written and FAIL before implementation (GREEN)
- Follow TDD Red-Green-Refactor cycle
- Core implementation before integration
- Story complete and tests passing before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- All tests for a user story marked [P] can be written in parallel
- Different user stories can be worked on in parallel by different team members after Phase 2

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all test writing for User Story 1 together:
Task: "Write acceptance test TestAccImagesDataSource_RepositoryAndTag"
Task: "Write acceptance test TestAccImagesDataSource_RepositoryOnly"
Task: "Write acceptance test TestAccImagesDataSource_TagOnly"
Task: "Write acceptance test TestAccImagesDataSource_NoFilters"
Task: "Write acceptance test TestAccImagesDataSource_AttributeReference"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (RED tests → GREEN implementation)
4. **STOP and VALIDATE**: Run `make testacc TESTARGS='-run=TestAccImagesDataSource'` to verify US1
5. Optionally deploy/demo basic image querying

### Incremental Delivery (Recommended)

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 (basic querying) → Test independently → Basic MVP ready!
3. Add User Story 2 (pattern matching) → Test independently → Enhanced filtering available
4. Add User Story 3 (VM integration) → Test independently → Full VM provisioning workflow
5. Add Edge Cases (Phase 6) → Robust error handling
6. Add Polish (Phase 7) → Production ready
7. Each phase adds value without breaking previous functionality

### Parallel Team Strategy

With multiple developers (after Phase 2 complete):

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (basic querying)
   - Developer B: User Story 2 (pattern matching)
   - Developer C: User Story 3 (VM integration)
3. Coordinate on shared `images_data_source.go` file - use feature branches and merge carefully
4. Stories complete and integrate independently

**Note**: Since all user stories modify the same file (`images_data_source.go`), sequential implementation (P1 → P2 → P3) is recommended to avoid merge conflicts. Parallel work is better suited for test writing and documentation tasks.

---

## Notes

- [P] tasks = different files or independent operations, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- **TDD is mandatory**: RED (failing tests) → GREEN (passing implementation) → REFACTOR
- Verify tests fail before implementing (ensures tests are valid)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Run `make testacc` frequently to catch regressions early
- Update CHANGELOG.md at the end

---

## Task Summary

- **Total Tasks**: 59
- **Setup Tasks**: 3
- **Foundational Tasks**: 6
- **User Story 1 Tasks**: 16 (6 tests + 10 implementation including FR-022 validation)
- **User Story 2 Tasks**: 9 (5 tests + 4 implementation)
- **User Story 3 Tasks**: 8 (5 tests + 3 implementation)
- **Edge Cases Tasks**: 9 (5 tests + 4 implementation)
- **Polish Tasks**: 8
- **Parallel Opportunities**: 26 tasks marked [P]
- **Independent Test Criteria**: Each user story has acceptance tests that verify functionality independently
- **Suggested MVP Scope**: Phase 1 + Phase 2 + Phase 3 (User Story 1 only) = Basic image querying capability

---

## Validation Checklist

- ✅ All tasks follow checklist format (checkbox, ID, labels, file paths)
- ✅ Tasks organized by user story for independent implementation
- ✅ TDD workflow enforced (RED tests before GREEN implementation)
- ✅ Clear dependencies and execution order documented
- ✅ Parallel opportunities identified (24 tasks)
- ✅ MVP scope defined (User Story 1)
- ✅ File paths are absolute and specific
- ✅ Each user story has independent test criteria
- ✅ Edge cases and polish phases included
