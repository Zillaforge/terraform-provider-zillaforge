# Tasks: Zillaforge Provider Migration

**Feature**: [001-zillaforge-provider-migration](./spec.md)  
**Input**: Design documents from `/specs/001-zillaforge-provider-migration/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Organization**: Tasks grouped by user story for independent implementation and testing. Follow TDD (RED-GREEN-REFACTOR) workflow per constitution Principle II.

**Tests**: Not explicitly requested in specification - focusing on acceptance tests to verify each story works.

## Format: `- [ ] [ID] [P?] [Story?] Description`

- **[P]**: Parallelizable (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label (US1, US2, US3)
- File paths use repository root as base

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and baseline verification

- [X] T001 Verify current provider compiles successfully with `go build` at repository root
- [X] T002 Run existing acceptance test suite with `make testacc` to establish baseline (all tests must pass)
- [X] T003 Create git branch `001-zillaforge-provider-migration` and check out

**Checkpoint**: Setup complete - baseline established, no regressions allowed

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core dependencies and tooling MUST be complete before any user story work

- [X] T004 Add JWT validation library to go.mod for api_key format checks (e.g., github.com/golang-jwt/jwt/v5)
- [X] T005 Verify terraform-plugin-log v0.9.0 is available for structured logging (already in go.mod)
- [X] T006 Create test helper function for JWT token generation in internal/provider/provider_test.go

**Checkpoint**: Foundation ready - user story implementation can begin

---

## Phase 3: User Story 1 - Provider Rebranding (Priority: P1) 🎯 MVP

**Goal**: Rename ScaffoldingProvider to ZillaforgeProvider  
**Independent Test**: Provider compiles, `terraform init` recognizes "zillaforge"

### Tests for User Story 1

- [X] T007 [P] [US1] Write failing acceptance test TestZillaforgeProvider_Metadata in internal/provider/provider_test.go (verify TypeName == "zillaforge")

### Implementation for User Story 1

- [X] T008 [US1] Rename ScaffoldingProvider type to ZillaforgeProvider in internal/provider/provider.go
- [X] T009 [US1] Rename ScaffoldingProviderModel to ZillaforgeProviderModel in internal/provider/provider.go
- [X] T010 [US1] Update Metadata() method to return TypeName "zillaforge" in internal/provider/provider.go
- [X] T011 [US1] Update New() function to return ZillaforgeProvider in internal/provider/provider.go
- [X] T012 [US1] Update interface checks (var _ provider.Provider) in internal/provider/provider.go
- [X] T013 [P] [US1] Update module path to github.com/Zillaforge/terraform-provider-zillaforge in go.mod
- [X] T014 [P] [US1] Update provider name in examples/provider/provider.tf (scaffolding → zillaforge)
- [X] T015 [P] [US1] Update resource examples in examples/resources/ if they reference provider name
- [X] T016 [P] [US1] Update data source examples in examples/data-sources/ if they reference provider name
- [X] T017 [US1] Run `go mod tidy` to update dependencies
- [X] T018 [US1] Verify TestZillaforgeProvider_Metadata now passes
- [X] T019 [US1] Run full acceptance test suite `make testacc` to ensure no regressions

**Checkpoint**: User Story 1 complete - provider rebranded and functional

---

## Phase 4: User Story 2 - Zillaforge SDK Integration (Priority: P2)

**Goal**: Integrate Zillaforge SDK, initialize client in Configure()  
**Independent Test**: SDK client initializes, resources receive client

### Tests for User Story 2

- [X] T020 [P] [US2] Write failing acceptance test TestZillaforgeProvider_Configure_InitializesSDK in internal/provider/provider_test.go
- [X] T021 [P] [US2] Write failing test TestZillaforgeProvider_Configure_InvalidCredentials in internal/provider/provider_test.go (expect diagnostic error)

### Implementation for User Story 2

- [X] T022 [US2] Add github.com/Zillaforge/cloud-sdk@latest dependency with `go get github.com/Zillaforge/cloud-sdk@latest`
- [X] T023 [US2] Import zillaforge SDK in internal/provider/provider.go
- [X] T024 [US2] Implement Configure() method SDK client initialization in internal/provider/provider.go
- [X] T025 [US2] Add INFO level logging for provider configuration start in internal/provider/provider.go (using tflog.Info)
- [X] T026 [US2] Add DEBUG level logging for SDK client initialization in internal/provider/provider.go (using tflog.Debug)
- [X] T027 [US2] Add error handling for SDK initialization failures with detailed diagnostics in internal/provider/provider.go
- [X] T028 [US2] Share SDK client via resp.ResourceData and resp.DataSourceData in internal/provider/provider.go
- [X] T029 [P] [US2] Update ExampleResource.Configure() to accept *zillaforge.Client in internal/provider/example_resource.go
- [X] T030 [P] [US2] Update ExampleResource struct to store *zillaforge.Client in internal/provider/example_resource.go
- [X] T031 [P] [US2] Update ExampleDataSource.Configure() to accept *zillaforge.Client in internal/provider/example_data_source.go
- [X] T032 [P] [US2] Update ExampleDataSource struct to store *zillaforge.Client in internal/provider/example_data_source.go
- [X] T033 [P] [US2] Update ExampleEphemeralResource.Configure() to accept *zillaforge.Client in internal/provider/example_ephemeral_resource.go
- [X] T034 [P] [US2] Update ExampleEphemeralResource struct to store *zillaforge.Client in internal/provider/example_ephemeral_resource.go
- [X] T035 [US2] Run `go mod tidy` to update go.sum
- [X] T036 [US2] Verify TestZillaforgeProvider_Configure_InitializesSDK now passes
- [X] T037 [US2] Verify TestZillaforgeProvider_Configure_InvalidCredentials now passes
- [X] T038 [US2] Run full acceptance test suite `make testacc` to ensure no regressions

**Checkpoint**: User Story 2 complete - SDK integrated, client available to resources

---

## Phase 5: User Story 3 - Provider Schema Update (Priority: P3)

**Goal**: Add proper provider schema attributes with validation  
**Independent Test**: Schema accepts config, validates mutual exclusivity, JWT format

### Tests for User Story 3

- [X] T039 [P] [US3] Write failing test TestZillaforgeProvider_Schema_BothProjectIdentifiers in internal/provider/provider_test.go (expect conflict error)
- [X] T040 [P] [US3] Write failing test TestZillaforgeProvider_Schema_NeitherProjectIdentifier in internal/provider/provider_test.go (expect required error)
- [X] T041 [P] [US3] Write failing test TestZillaforgeProvider_Schema_ValidWithProjectID in internal/provider/provider_test.go (expect success)
- [X] T042 [P] [US3] Write failing test TestZillaforgeProvider_Schema_ValidWithProjectSysCode in internal/provider/provider_test.go (expect success)
- [X] T043 [P] [US3] Write failing test TestZillaforgeProvider_Schema_InvalidJWTFormat in internal/provider/provider_test.go (expect JWT format error)
- [X] T044 [P] [US3] Write failing test TestZillaforgeProvider_MultiInstance_Aliases in internal/provider/provider_test.go (verify multiple provider instances work)

### Implementation for User Story 3

- [X] T045 [US3] Update ZillaforgeProviderModel with 4 attributes (api_endpoint, api_key, project_id, project_sys_code) in internal/provider/provider.go
- [X] T046 [US3] Implement Schema() method with full attribute definitions and MarkdownDescription in internal/provider/provider.go
- [X] T047 [US3] Mark api_key as Sensitive: true in schema in internal/provider/provider.go
- [X] T048 [US3] Implement JWT token format validation helper function in internal/provider/provider.go (check header.payload.signature structure)
- [X] T049 [US3] Add environment variable fallback logic for all 4 attributes in Configure() in internal/provider/provider.go
- [X] T050 [US3] Implement api_key presence validation in Configure() in internal/provider/provider.go
- [X] T051 [US3] Implement api_key JWT format validation (<100ms per NFR-001) in Configure() in internal/provider/provider.go
- [X] T052 [US3] Implement project identifier mutual exclusivity validation in Configure() in internal/provider/provider.go
- [X] T053 [US3] Add detailed diagnostic messages for all validation errors per contracts/provider-config-schema.md in internal/provider/provider.go
- [X] T054 [US3] Add structured logging with provider alias context for multi-instance support in internal/provider/provider.go
- [X] T055 [US3] Update SDK client initialization to use validated config values in internal/provider/provider.go
- [X] T056 [US3] Add retry count and error details to SDK initialization error diagnostics (NFR-003) in internal/provider/provider.go
- [X] T057 [US3] Update examples/provider/provider.tf with all 4 attributes and usage examples
- [X] T058 [US3] Add multi-instance example with provider alias to examples/provider/provider.tf
- [X] T059 [US3] Verify all 6 US3 tests now pass (T039-T044)
- [X] T060 [US3] Run full acceptance test suite `make testacc` to ensure no regressions

**Checkpoint**: User Story 3 complete - provider schema fully implemented with validation

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements affecting multiple user stories

- [ ] T061 [P] Generate provider documentation with `make generate` (runs tfplugindocs)
- [ ] T062 [P] Verify docs/index.md includes all 4 attributes with MarkdownDescription (SC-005)
- [ ] T063 [P] Update README.md with Zillaforge provider usage instructions
- [ ] T064 Run golangci-lint with `golangci-lint run` to ensure code quality
- [ ] T065 Verify all 6 success criteria from spec.md are met:
  - SC-001: Zero compilation errors (`go build`)
  - SC-002: `terraform init` recognizes "zillaforge"
  - SC-003: SDK client initializes (acceptance tests)
  - SC-004: Clear diagnostics <5s (acceptance tests)
  - SC-005: 100% attribute MarkdownDescription coverage (docs/)
  - SC-006: All existing tests pass (`make testacc`)
- [ ] T066 Run quickstart.md validation checklist manually
- [ ] T067 Create 3 git commits per quickstart.md commit strategy:
  - Commit 1: feat(P1): rebrand ScaffoldingProvider to ZillaforgeProvider
  - Commit 2: feat(P2): integrate Zillaforge SDK client
  - Commit 3: feat(P3): implement provider configuration schema

**Checkpoint**: All tasks complete - ready for PR

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational completion
- **User Story 2 (Phase 4)**: Depends on User Story 1 completion (needs renamed types)
- **User Story 3 (Phase 5)**: Depends on User Story 2 completion (needs SDK integration)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Completion Order

**Sequential (Recommended)**:
1. User Story 1 (P1) - establishes naming foundation
2. User Story 2 (P2) - builds on renamed types
3. User Story 3 (P3) - completes schema on SDK integration

**Rationale**: Each story builds on the previous, following natural dependency flow (naming → SDK → schema)

### Within Each User Story

**TDD Workflow** (per Constitution Principle II):
1. Write failing tests FIRST (RED)
2. Implement minimal code to pass tests (GREEN)
3. Verify tests pass
4. Run full test suite to prevent regressions
5. Refactor if needed (REFACTOR)

### Parallel Opportunities

#### Phase 1 (Setup)
- T001, T002, T003 are sequential (baseline before branching)

#### Phase 2 (Foundational)
- T004 and T005 can run in parallel (independent dependencies)
- T006 depends on T004 (needs JWT library)

#### Phase 3 (User Story 1)
- T007 (test) runs first alone (TDD RED phase)
- After T011 completes, these can run in parallel:
  - T013 (go.mod)
  - T014 (provider.tf)
  - T015 (resource examples)
  - T016 (data source examples)

#### Phase 4 (User Story 2)
- T020, T021 (tests) run in parallel (TDD RED phase)
- After T028 completes, these can run in parallel:
  - T029, T030 (ExampleResource)
  - T031, T032 (ExampleDataSource)
  - T033, T034 (ExampleEphemeralResource)

#### Phase 5 (User Story 3)
- T039-T044 (all tests) run in parallel (TDD RED phase)
- T057, T058 (examples) run in parallel after T056

#### Phase 6 (Polish)
- T061, T062, T063 can run in parallel
- T064-T067 are sequential (validation → commits)

---

## Parallel Example: User Story 3

```bash
# Terminal 1: Write test for both identifiers
code internal/provider/provider_test.go  # T039

# Terminal 2: Write test for neither identifier  
code internal/provider/provider_test.go  # T040

# Terminal 3: Write test for valid project_id
code internal/provider/provider_test.go  # T041

# Terminal 4: Write test for valid project_sys_code
code internal/provider/provider_test.go  # T042

# Terminal 5: Write test for invalid JWT
code internal/provider/provider_test.go  # T043

# Terminal 6: Write test for multi-instance
code internal/provider/provider_test.go  # T044

# Run all tests (should FAIL - RED phase)
go test ./internal/provider -v

# Then implement schema and validation sequentially (T045-T056)

# After T056, parallelize examples:
# Terminal 1: Update provider.tf
code examples/provider/provider.tf  # T057

# Terminal 2: Add multi-instance example
code examples/provider/provider.tf  # T058
```

---

## Implementation Strategy

### MVP Delivery (Minimum Viable Product)

**Phase 3 (User Story 1) = MVP**: Delivers a properly branded, functional provider.

**Rationale**: 
- Provider compiles and works with Terraform
- Establishes correct identity (zillaforge)
- All existing tests pass
- Can be released immediately if needed

### Incremental Delivery

- **After Phase 3**: MVP released
- **After Phase 4**: SDK-integrated provider released (enables real API operations)
- **After Phase 5**: Production-ready provider released (full schema, validation, multi-instance)

### Testing Strategy

**Per Constitution Principle II (NON-NEGOTIABLE)**:
- Tests written BEFORE implementation (RED-GREEN-REFACTOR)
- All existing tests must pass after each phase
- No skipped tests without justification
- Acceptance tests verify user-facing behavior

**Test Coverage**:
- Phase 3: 1 test (metadata)
- Phase 4: 2 tests (SDK initialization + error handling)
- Phase 5: 6 tests (validation scenarios + multi-instance)
- Total: 9 new acceptance tests

### Validation Checkpoints

**After each phase**:
1. Run `go build` (must succeed)
2. Run `make testacc` (all tests pass)
3. Manual smoke test (follow quickstart.md)

**Before final commit**:
1. Run `golangci-lint run` (no issues)
2. Run `make generate` (docs updated)
3. Verify all 6 success criteria (SC-001 to SC-006)

---

## Task Summary

- **Total Tasks**: 67
- **Setup**: 3 tasks
- **Foundational**: 3 tasks
- **User Story 1**: 13 tasks (1 test + 12 implementation)
- **User Story 2**: 19 tasks (2 tests + 17 implementation)
- **User Story 3**: 22 tasks (6 tests + 16 implementation)
- **Polish**: 7 tasks

**Parallelizable Tasks**: 24 marked with [P]

**Estimated Effort**: 8-12 hours (per plan.md)

**Risk Level**: Low - standard provider patterns, TDD workflow, clear validation requirements
