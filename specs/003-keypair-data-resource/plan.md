# Implementation Plan: Keypair Data Source and Resource

**Branch**: `003-keypair-data-resource` | **Date**: December 13, 2025 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-keypair-data-resource/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement Terraform data source and resource for managing SSH keypairs in ZillaForge VPS service. The feature enables infrastructure engineers to create, query, import, and delete SSH keypairs through Terraform configurations, with support for both user-provided public keys and system-generated keypairs. The data source follows the established pattern of flavor and network data sources, supporting both individual lookups and listing all keypairs with optional filtering.

## Technical Context

**Language/Version**: Go 1.22.4  
**Primary Dependencies**: 
- Terraform Plugin Framework v1.14.1
- github.com/Zillaforge/cloud-sdk v0.0.0-20251209081935-79e26e215136
- Terraform Plugin Testing v1.11.0

**Storage**: Cloud-sdk backed API (RESTful), Terraform state file  
**Testing**: Go testing with Terraform Plugin Testing framework (acceptance tests + unit tests)  
**Target Platform**: Linux server (Alpine Linux v3.20 dev container)  
**Project Type**: Terraform Provider (Go module)  
**Performance Goals**: 
- Data source queries complete within 1 second (per spec SC-002)
- Resource create operations complete within 2 seconds (per spec SC-001)
- Import operations complete within 3 seconds (per spec SC-007)

**Constraints**: 
- Must use Terraform Plugin Framework (not SDK v2) per constitution
- Private keys must be marked as sensitive attributes
- Keypair names validated by API (FR-017)
- API enforces account-level quotas
- Limited update support for keypairs: only description field is updatable (name and public_key are immutable per SDK)

**Scale/Scope**: 
- Single project: terraform provider extension
- 2 new files: data source + resource implementation
- Integration with existing github.com/Zillaforge/cloud-sdk keypairs module
- Consistent with existing flavor/network data source patterns

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Code Quality & Framework Compliance

- ✅ **Terraform Plugin Framework**: Will use Plugin Framework v1.14.1 (not SDK v2)
- ✅ **Framework Interfaces**: Data source implements `datasource.DataSource`, resource implements `resource.Resource`
- ✅ **MarkdownDescription**: All schema attributes will include MarkdownDescription
- ✅ **Input Validation**: Provider will validate keypair names via API, public key formats
- ✅ **State Management**: Will handle null, unknown, computed values per framework semantics (sensitive attribute for private_key)
- ✅ **Godoc Comments**: All exported types will have documentation

**Status**: ✅ PASS - No violations

### Principle II: Test-Driven Development (NON-NEGOTIABLE)

- ✅ **Tests Written First**: Acceptance tests will be written before implementation
- ✅ **Red-Green-Refactor**: Will follow TDD cycle strictly
- ✅ **CRUD Coverage**: Create, Read, Update (description-only), Delete operations for resource
- ✅ **Import Testing**: Import functionality will have acceptance test coverage
- ✅ **Data Source Testing**: Read operations will be tested
- ✅ **Unit Tests**: Will cover validation logic, type conversions, error handling

**Status**: ✅ PASS - TDD workflow planned

### Principle III: User Experience Consistency

- ✅ **Actionable Errors**: Error messages will specify what to fix (e.g., "invalid public key format: expected RSA/ECDSA/ED25519")
- ✅ **Diagnostics Severity**: Will use Error for failures, Warning for in-use keypair deletion
- ✅ **Naming Conventions**: snake_case attributes (public_key, private_key)
- ✅ **Required vs Optional**: Schema clearly defines required (name) vs optional (public_key, description)
- ✅ **Computed Attributes**: id, fingerprint, private_key marked as Computed
- ✅ **Import Documentation**: Will include concrete examples with actual keypair IDs
- ✅ **Semantic Versioning**: New resource/data source = minor version bump

**Status**: ✅ PASS - UX standards met

### Principle IV: Performance & Resource Efficiency

- ✅ **Minimize API Calls**: Single API call per operation (Get, List, Create, Delete)
- ✅ **Context Timeouts**: Will respect context and propagate to HTTP clients (via cloud-sdk)
- ✅ **Pagination**: cloud-sdk List() handles pagination internally
- ✅ **Provider Caching**: ProjectClient cached in provider configuration
- ✅ **Logging Levels**: Debug for verbose, Info for operations, Warn/Error for issues
- ✅ **Retry Logic**: cloud-sdk baseClient implements exponential backoff

**Status**: ✅ PASS - Performance requirements met

### Overall Constitution Status: ✅ PASS

All four core principles satisfied. No violations requiring justification. Proceed to Phase 0.

---

## Post-Design Constitution Re-Evaluation

*Performed after Phase 1 design completion*

### Principle I: Code Quality & Framework Compliance

- ✅ **Terraform Plugin Framework**: Contracts use Plugin Framework v1.14.1 schemas
- ✅ **Framework Interfaces**: datasource.DataSource and resource.Resource interfaces defined
- ✅ **MarkdownDescription**: All attributes documented in contracts (see contracts/*.md)
- ✅ **Input Validation**: Mutual exclusivity check for id/name filters, API-side validation for keys
- ✅ **State Management**: Sensitive attribute, RequiresReplace, UseStateForUnknown plan modifiers specified
- ✅ **Godoc Comments**: Quickstart shows proper struct and function documentation patterns

**Status**: ✅ PASS

### Principle II: Test-Driven Development (NON-NEGOTIABLE)

- ✅ **Tests Written First**: Quickstart emphasizes TDD workflow (RED-GREEN-REFACTOR)
- ✅ **Test Coverage**: Acceptance test cases defined in both contracts
- ✅ **CRUD Coverage**: Create, Read, Update, Delete test scenarios documented
- ✅ **Import Testing**: Import test case included with private_key verification ignore
- ✅ **Data Source Testing**: Multiple query modes tested (by id, by name, list all, invalid)

**Status**: ✅ PASS

### Principle III: User Experience Consistency

- ✅ **Actionable Errors**: Error messages documented in contracts with specific guidance
- ✅ **Diagnostics Severity**: Warnings for deletion, Errors for failures
- ✅ **Naming Conventions**: All attributes use snake_case (public_key, private_key)
- ✅ **Required vs Optional**: Clearly defined in schema contracts
- ✅ **Computed Attributes**: All computed fields marked with plan modifiers
- ✅ **Import Documentation**: Import examples in resource contract and quickstart

**Status**: ✅ PASS

### Principle IV: Performance & Resource Efficiency

- ✅ **Minimize API Calls**: Single Get() or List() per operation
- ✅ **Context Timeouts**: cloud-sdk integration preserves context
- ✅ **Pagination**: Handled by cloud-sdk internally
- ✅ **Provider Caching**: ProjectClient reused from provider configuration
- ✅ **Logging Levels**: tflog.Warn() for deletion warnings, errors for failures

**Status**: ✅ PASS

### Final Constitution Status: ✅ PASS

All principles validated post-design. Design artifacts (research, data-model, contracts) maintain constitutional compliance. Ready for Phase 2 (tasks generation) and implementation.

---

## Project Structure

### Documentation (this feature)

```text
specs/003-keypair-data-resource/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── keypair-data-source-schema.md
│   └── keypair-resource-schema.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
└── vps/
    ├── data/
    │   ├── flavor_data_source.go      # Existing (pattern reference)
    │   ├── network_data_source.go     # Existing (pattern reference)
    │   └── keypair_data_source.go     # NEW - Keypair data source
    │       └── keypair_data_source_test.go  # NEW - Acceptance tests
    └── resource/
        └── keypair_resource.go         # NEW - Keypair resource
            └── keypair_resource_test.go      # NEW - Acceptance tests

examples/
├── data-sources/
│   └── zillaforge_keypairs/
│       └── data-source.tf             # NEW - Usage example
└── resources/
    └── zillaforge_keypair/
        ├── resource.tf                # NEW - Usage example
        └── import.sh                  # NEW - Import example

docs/
├── data-sources/
│   └── keypairs.md                    # Generated by tfplugindocs
└── resources/
    └── keypair.md                     # Generated by tfplugindocs
```

**Structure Decision**: Following established Terraform provider pattern with separate internal/vps/data/ and internal/vps/resource/ directories. This maintains consistency with existing flavor and network implementations. The cloud-sdk integration pattern is reused with minimal changes.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations - this section intentionally left empty.
