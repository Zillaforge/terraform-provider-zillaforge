<!--
Sync Impact Report (Version 1.0.0):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Version Change: [template] → 1.0.0 (initial ratification)

Modified Principles:
- NEW: Code Quality & Framework Compliance
- NEW: Test-Driven Development (NON-NEGOTIABLE)
- NEW: User Experience Consistency
- NEW: Performance & Resource Efficiency

Added Sections:
- Core Principles (4 principles for Terraform provider development)
- Quality Standards (provider-specific requirements)
- Development Workflow (TDD and acceptance testing gates)
- Governance (versioning and compliance)

Templates Status:
✅ plan-template.md - Aligned (constitution check gate present)
✅ spec-template.md - Aligned (acceptance scenarios structure matches principles)
✅ tasks-template.md - Aligned (test-first workflow and independent testing)

Follow-up TODOs:
- None (all placeholders filled with concrete values)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
-->

# Terraform Provider Zillaforge Constitution

## Core Principles

### I. Code Quality & Framework Compliance

**MUST Requirements**:
- All provider code MUST use the Terraform Plugin Framework (not Plugin SDK v2)
- Resources, data sources, and ephemeral resources MUST implement framework interfaces correctly
- Schema definitions MUST include MarkdownDescription for every attribute and resource
- Provider configuration MUST validate inputs and return actionable diagnostics
- State management MUST handle null, unknown, and computed values according to framework semantics
- All exported types MUST have godoc comments explaining their purpose

**Rationale**: Terraform providers are infrastructure-critical components. Framework compliance ensures compatibility with Terraform CLI, correct state handling, and predictable user experience. Documentation enables HashiCorp Registry publishing and user adoption.

### II. Test-Driven Development (NON-NEGOTIABLE)

**MUST Requirements**:
- Acceptance tests MUST be written before implementation code
- Tests MUST fail initially (red), then pass after implementation (green)
- Every resource CRUD operation (Create, Read, Update, Delete) MUST have acceptance test coverage
- Import functionality MUST be tested for every resource
- Data sources MUST have acceptance tests validating read operations
- Unit tests MUST cover validation logic, type conversions, and error handling
- Red-Green-Refactor cycle MUST be strictly followed—no implementation without failing tests first

**Rationale**: Terraform providers interact with real cloud APIs. TDD prevents regressions, validates API contract understanding, and ensures state correctness. Writing tests first forces clear specification of expected behavior before implementation complexity.

### III. User Experience Consistency

**MUST Requirements**:
- Error messages MUST be actionable (tell users what to fix and how)
- Diagnostics MUST use appropriate severity levels (Error vs Warning)
- Attribute naming MUST follow Terraform conventions (snake_case, avoid abbreviations)
- Required vs Optional attributes MUST be clearly defined in schemas
- Computed attributes MUST be explicitly marked as such
- Timeouts MUST be configurable for long-running operations
- Import documentation MUST include concrete examples with actual IDs
- Breaking changes MUST follow semantic versioning (major version bump)

**Rationale**: Users expect consistency across Terraform providers. Poor UX leads to support burden, GitHub issues, and adoption friction. Clear diagnostics reduce troubleshooting time and improve DevOps workflows.

### IV. Performance & Resource Efficiency

**MUST Requirements**:
- API calls MUST be minimized (batch operations where possible)
- Read operations MUST not make unnecessary API calls if state is fresh
- Context timeouts MUST be respected and propagated to HTTP clients
- Large resource sets MUST support pagination
- Provider configuration MUST be cached (not recomputed per resource operation)
- Logs MUST use appropriate levels (Debug for verbose, Info for key events, Warn/Error for issues)
- HTTP clients MUST implement retry logic with exponential backoff for transient failures

**Rationale**: Terraform apply/plan operations amplify performance issues across resource counts. Poor performance blocks CI/CD pipelines. Excessive API calls can hit rate limits or incur costs. Efficient providers respect user time and infrastructure constraints.

## Quality Standards

### Code Organization
- Resources MUST be in `internal/provider/resource_*.go` files
- Data sources MUST be in `internal/provider/data_source_*.go` files
- Shared client code MUST be in `internal/provider/client.go` or dedicated package
- Tests MUST be colocated with implementation (`*_test.go` files)
- Examples MUST exist in `examples/` for every resource and data source

### Documentation Requirements
- Generated docs MUST be created via `tfplugindocs` (not manually written)
- Schema descriptions MUST be written in Markdown
- Examples MUST be valid Terraform HCL that can be copied and run
- Import blocks MUST show exact command syntax
- Changelogs MUST follow Keep a Changelog format

### Testing Requirements
- Acceptance tests MUST clean up created resources (avoid test pollution)
- Tests MUST NOT depend on hardcoded external resource IDs (use data sources to discover)
- PreCheck functions MUST validate required environment variables
- Tests MUST use terraform-plugin-testing framework CheckFunctions
- Parallel test execution MUST be safe (no shared mutable state)

## Development Workflow

### Test-First Implementation Process
1. Write acceptance test that exercises the desired behavior (test MUST fail)
2. Confirm test failure with clear error indicating missing functionality
3. Implement minimal code to make test pass
4. Verify test passes (green)
5. Refactor for code quality without breaking tests
6. Repeat for next behavior

### Quality Gates
- All tests MUST pass before merge (`make testacc`)
- Go linters MUST pass (`golangci-lint run`)
- Documentation MUST be generated and committed (`make generate`)
- No new acceptance tests MAY be skipped without documented justification
- Breaking changes MUST be documented in CHANGELOG.md with migration guide

### Acceptance Testing Standards
- Use `resource.Test` or `resource.ParallelTest` from terraform-plugin-testing
- Each test MUST have descriptive name explaining scenario
- Use `TestStep` with `Config` for state transitions
- Use `Check` with multiple `TestCheckFunc` for comprehensive validation
- Use `ImportState: true` with `ImportStateVerify: true` to validate import

## Governance

This constitution is the authoritative source for development standards in the Terraform Provider Zillaforge project. All code reviews, feature designs, and architectural decisions MUST comply with these principles.

### Amendment Process
- Amendments require documented rationale and impact analysis
- Version bumps follow semantic versioning:
  - **MAJOR**: Backward incompatible principle removal or redefinition
  - **MINOR**: New principle added or material guidance expansion
  - **PATCH**: Clarifications, wording improvements, non-semantic fixes
- Template files MUST be updated to reflect constitutional changes
- Migration plans MUST be provided for breaking governance changes

### Compliance & Enforcement
- All pull requests MUST verify constitutional compliance
- Deviations MUST be explicitly justified in PR description
- Recurring violations indicate need for constitutional amendment (update rules, don't break them)
- Specification documents in `/specs/` MUST reference applicable principles
- Complexity MUST be justified (document rationale when violating YAGNI)

**Version**: 1.0.0 | **Ratified**: 2025-12-06 | **Last Amended**: 2025-12-06
