# Implementation Plan: Image Data Source

**Branch**: `005-tag-data-source` | **Date**: December 15, 2025 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-tag-data-source/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement `zillaforge_images` Terraform data source to query VM images (represented as repository:tag pairs) from the ZillaForge VRM API. The data source supports optional filtering by repository and tag (including glob pattern matching), returns a list of images with attributes including `id` (Tag ID from cloud-sdk), `repository_name`, `tag_name`, `size`, `operatingSystem`, `description`, `type`, and `status`. The feature uses `github.com/Zillaforge/cloud-sdk` VRM Tags client (`projectClient.VRM().Tags().List()`) to retrieve images from the `/vrm/api/v1/project/{project-id}/tags` endpoint, implements client-side filtering for glob patterns and exact matches, and handles pagination transparently. Images are sorted deterministically by repository_name ascending then tag_name ascending for consistent Terraform state management.

## Technical Context

**Language/Version**: Go 1.22.4  
**Primary Dependencies**: 
- `github.com/Zillaforge/cloud-sdk` v0.0.0-20251209081935-79e26e215136 (VRM Tags API)
- `github.com/hashicorp/terraform-plugin-framework` v1.14.1 (data source framework)
- `github.com/hashicorp/terraform-plugin-framework-validators` v0.17.0 (schema validators)
- `github.com/hashicorp/terraform-plugin-testing` v1.11.0 (acceptance tests)

**Storage**: N/A (read-only data source querying remote API)  
**Testing**: terraform-plugin-testing framework for acceptance tests, Go testing for unit tests  
**Target Platform**: Terraform providers (cross-platform Go binaries for Linux/macOS/Windows)  
**Project Type**: Terraform provider (single Go module with internal packages)  
**Performance Goals**: 
- Query all images in repository: <3 seconds (SC-001)
- Filter by exact tag name: <2 seconds (SC-002)
- Pattern filtering: <3 seconds (SC-003)
- Handle up to 1000 images without degradation (SC-008)

**Constraints**:
- Must use Terraform Plugin Framework (not SDK v2) per constitution
- All schema attributes must include MarkdownDescription
- Acceptance tests required before implementation (TDD mandate)
- Server-enforced maximum limit on result sets (1000 images)
- Glob-only pattern matching (* and ?) - no regex support
- Project-scoped API access: use `/vrm/api/v1/project/{project-id}/repositories` for repository lookups and `/vrm/api/v1/project/{project-id}/tags` or `/vrm/api/v1/project/{project-id}/repository/{repository-id}/tags` for tag retrieval depending on filters

**Scale/Scope**: 
- Expected data volume: 10-100 repositories, 1-50 tags per repository
- Single data source implementation in `internal/vrm/data/images_data_source.go`
- Reuses existing provider client infrastructure
- 21 functional requirements, 3 user stories (P1 + 2xP2)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Framework Compliance

✅ **PASS** - All requirements use Terraform Plugin Framework interfaces
- Data source implements `datasource.DataSource` interface
- Schema uses framework types (`types.String`, `types.Int64`, `types.List`)
- All attributes include `MarkdownDescription` per FR-017
- Uses `types.List` with nested object type for images array
- Context-aware error handling with `resp.Diagnostics`

✅ **PASS** - Provider integration follows established patterns
- Reuses existing provider client configuration
- VRM client accessed via `projectClient.VRM().Tags()`
- Follows same pattern as Flavors, Networks, Keypairs, SecurityGroups data sources

### II. Test-Driven Development (NON-NEGOTIABLE)

✅ **PASS** - TDD workflow mandated in spec
- Acceptance tests required before implementation per User Stories
- 3 user stories with specific acceptance scenarios
- Test-first pattern documented in 004-security-group spec precedent
- Follows Red-Green-Refactor cycle from existing provider tests

✅ **PASS** - Comprehensive test coverage planned
- Acceptance tests for all filter combinations (repository, tag, both, none)
- Pattern matching tests (glob wildcards)
- Error scenarios (non-existent repository, mutual exclusivity validation)
- Edge cases (empty results, identical timestamps, special characters)

### III. User Experience Consistency

✅ **PASS** - Error messages and diagnostics specified
- FR-009: Graceful API error handling with actionable messages
- Clear diagnostics for repository not found (FR-007 returns empty list consistent with FR-006)
- Mutual exclusivity validation for tag name vs pattern (FR-014, FR-018)
- Authentication errors distinguished from empty results (edge case documented)

✅ **PASS** - Schema design follows Terraform conventions
- Attribute naming: snake_case (`repository_name`, `tag_name`, `operating_system`)
- Optional filters clearly marked (repository, tag)
- Computed attributes explicitly marked
- Import not applicable (data sources are read-only)

### IV. Performance & Resource Efficiency

✅ **PASS** - Performance requirements defined
- Clear performance targets in success criteria (SC-001, SC-002, SC-003)
- Server-side limit prevents excessive API calls (1000 max, clarification Q3)
- Context timeout propagation to SDK client
- Read implementations aim for a minimal number of API calls and will use the most specific SDK method available: `Repositories().Get()/List()` + `RepositoryResource.Tags().List()` when `repository` filter specified, otherwise `Tags().List()` for project-wide scans.

✅ **PASS** - Efficient filtering strategy
- SDK List() methods provide pagination support (FR-016)
- Client-side filtering only where necessary (pattern matching)
- Results sorted once after retrieval (FR-015)
- Minimal API calls: prefer repository-scoped tag listing to avoid scanning large tag lists

### Gate Status: ✅ ALL GATES PASSED

No violations. Proceed to Phase 0 Research.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── provider/
│   ├── provider.go              # Register zillaforge_images in DataSources() method
│   └── provider_test.go
└── vrm/
    └── data/
        ├── images_data_source.go       # NEW: zillaforge_images implementation
        └── images_data_source_test.go  # NEW: Acceptance tests

examples/
└── data-sources/
    └── zillaforge_images/              # NEW: Example configurations
        └── data-source.tf

docs/
└── data-sources/
    └── images.md                       # NEW: Generated documentation (via tfplugindocs)
```

**Structure Decision**: Data source placed in `internal/vrm/data/` to align with the VRM (Virtual Registry Manager) domain, as images are managed by the VRM API rather than VPS. This differs from the VPS data sources (flavors, networks, keypairs, security_groups in `internal/vps/data/`). VRM Tags client accessed via same provider client infrastructure (`internal/provider/provider.go` configures cloud-sdk client, data source receives configured client via `Configure()` method).

## Complexity Tracking

No constitutional violations identified. All gates passed.

---

## Phase 0-1 Deliverables

### ✅ Phase 0: Research (Complete)

- [research.md](research.md) - Technical decisions for VRM Tags API integration

**Key Findings**:
- VRM module uses `projectClient.VRM().Tags().List()` (different from VPS module)
- Tag entity includes embedded Repository information
- Hybrid filtering: server-side pagination + client-side exact/pattern matching
- Digest information may exist in `Tag.Extra` but the data source does not expose a `digest` attribute (image content may be identified by an immutable cryptographic hash in the VRM service)
- Deterministic sort: repository_name asc, tag_name asc for determinism

### ✅ Phase 1: Design & Contracts (Complete)

- [data-model.md](data-model.md) - Entity definitions, attribute mappings, data flow
- [contracts/images-data-source-schema.md](contracts/images-data-source-schema.md) - Full Terraform schema, filter behavior, test contracts
- [quickstart.md](quickstart.md) - Developer guide with SDK patterns and TDD workflow
- Agent context updated with VRM/Tags technology

**Key Designs**:
- Data source: `zillaforge_images` with filters `repository`, `tag`, `tag_pattern`
- Output: `images` list (always list, even for unique results)
- Filter semantics: 4 scenarios (both, repo-only, tag-only, none)
- Pattern matching: glob-style via `filepath.Match()`
- VM integration: Use `images[0].id` for VM creation

---

## Constitution Re-Check (Post-Design)

*Required after Phase 1 completion*

### I. Code Quality & Framework Compliance

✅ **PASS** - Design maintains framework compliance
- Schema defined with proper MarkdownDescription for all attributes
- Uses framework types correctly (`types.List` with nested `types.Object`)
- Client configuration follows existing datasource patterns
- Error handling uses `resp.Diagnostics` API

### II. Test-Driven Development

✅ **PASS** - Test contracts fully specified
- 8 acceptance test scenarios defined in contracts document
- Test-first workflow documented in quickstart (RED → GREEN → REFACTOR)
- Follows existing test patterns from keypairs/security-groups specs

### III. User Experience Consistency

✅ **PASS** - UX design validated
- Error messages specified with actionable guidance
- Empty results return `[]` not errors (consistent with other data sources)
- Mutual exclusivity validation clear and helpful
- Filter behavior documented with examples

### IV. Performance & Resource Efficiency

✅ **PASS** - Performance design sound
- Single API call strategy confirmed
- Client-side filtering is O(n) with n ≤ 1000
- Sorting is O(n log n) acceptable for scale
- No unnecessary API calls or memory allocations

### Final Gate Status: ✅ ALL GATES PASSED (POST-DESIGN)

Design is constitutionally compliant. Ready for implementation phase (`/speckit.tasks`).

---

## Next Steps

1. **Run `/speckit.tasks`** to generate implementation task breakdown
2. **Implement acceptance tests** (test data source config, expected outputs)
3. **Implement data source** following quickstart TDD workflow
4. **Register in provider** (`internal/provider/provider.go`)
5. **Generate documentation** (`make generate`)
6. **Update CHANGELOG** with new data source feature
7. **Create PR** with spec artifacts and implementation

---

## Planning Summary

| Artifact | Status | Location |
|----------|--------|----------|
| plan.md | ✅ Complete | specs/005-tag-data-source/plan.md |
| research.md | ✅ Complete | specs/005-tag-data-source/research.md |
| data-model.md | ✅ Complete | specs/005-tag-data-source/data-model.md |
| contracts/ | ✅ Complete | specs/005-tag-data-source/contracts/images-data-source-schema.md |
| quickstart.md | ✅ Complete | specs/005-tag-data-source/quickstart.md |
| tasks.md | ⏳ Pending | Run `/speckit.tasks` to generate |

**Branch**: `005-tag-data-source`  
**Implementation Ready**: Yes  
**Constitutional Compliance**: ✅ Verified  
**Phase**: Planning Complete → Ready for Tasks Breakdown
