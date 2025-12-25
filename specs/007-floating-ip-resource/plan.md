# Implementation Plan: 007-floating-ip-resource

**Branch**: `007-floating-ip-resource` | **Date**: December 24, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/007-floating-ip-resource/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement Terraform resource (`zillaforge_floating_ip`) and data source (`zillaforge_floating_ips`) for managing ZillaForge floating IP addresses (public IPs) through infrastructure-as-code. The resource supports allocation, update (name/description), import, and release operations. The data source enables querying floating IPs with client-side filtering by id, name, ip_address, and status. Association with VPS instances is out of scope. Implementation uses cloud-sdk v0.0.0-20251209081935-79e26e215136 with Terraform Plugin Framework v1.14.1, following TDD workflow with acceptance tests as the primary validation mechanism.

## Technical Context

**Language/Version**: Go 1.22.4  
**Primary Dependencies**: Terraform Plugin Framework v1.14.1, cloud-sdk v0.0.0-20251209081935-79e26e215136, terraform-plugin-testing v1.11.0  
**Storage**: N/A (Terraform state managed by Terraform Core)  
**Testing**: terraform-plugin-testing acceptance tests via `make testacc TESTARGS='-run=TestAccXXXX' PARALLEL=1`  
**Target Platform**: Linux server (dev container: Alpine Linux v3.20)  
**Project Type**: Terraform provider (infrastructure management)  
**Performance Goals**: Minimize API calls, client-side filtering to avoid SDK List() filter bug, batch operations where possible  
**Constraints**: 
- Single IP pool (no pool selection parameter)
- Modifiable attributes limited to name and description
- Status values (ACTIVE, DOWN, PENDING, REJECTED) are informational only
- Client-side filtering required due to SDK List() bug
- All attributes must have MarkdownDescription
**Scale/Scope**: 
- 2 new files (resource + data source)
- 1 shared model file (internal/vps/model/floating_ip.go)
- 2 test files with minimum 6 acceptance test scenarios
- 4 user stories (P1: Allocate, P2: Query, P3: Update, P3: Import)
- 20 functional requirements

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Framework Compliance
- ✅ Uses Terraform Plugin Framework v1.14.1 (not SDK v2)
- ✅ Resources/data sources implement correct interfaces (Resource, ResourceWithConfigure, ResourceWithImportState, DataSource, DataSourceWithConfigure)
- ✅ All attributes have MarkdownDescription (required per constitution)
- ✅ Schema includes Optional/Computed markers and plan modifiers (UseStateForUnknown)
- ✅ State management handles null/unknown/computed values (stringPointerOrNull helper)
- ✅ Error diagnostics are actionable ("Unable to allocate floating IP: [API error]")

### II. Test-Driven Development (NON-NEGOTIABLE)
- ✅ Acceptance tests written in spec (6 scenarios for resource, 6 for data source)
- ✅ TDD workflow: Write failing test → Implement → Test passes → Refactor
- ✅ CRUD operations all tested (Create, Read, Update, Delete)
- ✅ Import functionality tested (ImportStateVerify)
- ✅ Data source Read tested with filters
- ✅ Unit tests for model conversions and filtering logic
- ✅ Test command: `make testacc TESTARGS='-run=TestAccFloatingIP' PARALLEL=1`

### III. User Experience Consistency
- ✅ Error messages actionable ("Unable to allocate floating IP: [API message]")
- ✅ Diagnostics use appropriate severity (Error for failures, not Warning)
- ✅ Attribute naming follows conventions (snake_case: ip_address, device_id)
- ✅ Required vs Optional clear (name/description optional, id/ip_address computed)
- ✅ Computed attributes marked explicitly with plan modifiers
- ✅ Import documentation with examples (`terraform import zillaforge_floating_ip.example fip-uuid-123`)
- ✅ Breaking changes: None (new feature)

### IV. Performance & Resource Efficiency
- ✅ API calls minimized (single Get per Read, batch List for data source)
- ✅ Read operations efficient (no unnecessary calls if state fresh)
- ✅ Context timeouts respected (ctx passed to all SDK calls)
- ✅ Pagination: Not needed (floating IPs expected to be small dataset)
- ✅ Provider config cached (VPSClient configured once)
- ✅ Client-side filtering (avoids SDK List() filter bug, reduces API calls)

**Status**: ✅ **PASS** - All constitutional requirements met

**Violations**: None

**Re-evaluation Post-Design**: 
- ✅ Data model design confirms single-pool approach (no pool parameter)
- ✅ API contracts confirm CRUD patterns and error handling
- ✅ Client-side filtering documented (SDK List() filter has bugs)
- ✅ MarkdownDescription required for all 10 attributes (6 resource + 4 data source filters + 1 nested result list)

## Project Structure

### Documentation (this feature)

```text
specs/007-floating-ip-resource/
├── plan.md              # This file (/speckit.plan command output)
├── spec.md              # Feature specification with 4 user stories, 20 requirements
├── research.md          # Phase 0 output: 6 research areas, key decisions
├── data-model.md        # Phase 1 output: 3 model structs, conversion functions
├── quickstart.md        # Phase 1 output: Developer guide with examples
├── checklists/
│   └── requirements.md  # Quality validation checklist
└── contracts/
    ├── cloud-sdk-api.md      # SDK FloatingIP CRUD API documentation
    └── terraform-schema.md   # Terraform resource/data source schemas
```

### Source Code (repository root)

```text
internal/vps/
├── model/
│   └── floating_ip.go                 # NEW: Shared data models
│       - FloatingIPResourceModel (6 attributes)
│       - FloatingIPDataSourceModel (4 filters + results)
│       - FloatingIPModel (nested result type)
│       - MapFloatingIPToResourceModel, MapFloatingIPToModel
│       - BuildCreateRequest, BuildUpdateRequest
│       - FilterFloatingIPs, matchesFilters
│       - stringPointerOrNull helper
├── resource/
│   ├── floating_ip_resource.go        # NEW: Resource implementation
│   │   - FloatingIPResource struct
│   │   - Metadata, Schema, Configure
│   │   - Create, Read, Update, Delete, ImportState
│   └── floating_ip_resource_test.go   # NEW: Acceptance tests
│       - TestAccFloatingIPResource_Basic
│       - TestAccFloatingIPResource_WithNameDescription
│       - TestAccFloatingIPResource_Import
│       - [6+ scenarios total]
└── data/
    ├── floating_ips_data_source.go    # NEW: Data source implementation
    │   - FloatingIPsDataSource struct
    │   - Metadata, Schema, Configure
    │   - Read (with client-side filtering)
    └── floating_ips_data_source_test.go # NEW: Acceptance tests
        - TestAccFloatingIPsDataSource_All
        - TestAccFloatingIPsDataSource_FilterByID
        - TestAccFloatingIPsDataSource_FilterByName
        - [6+ scenarios total]

internal/provider/
└── provider.go                         # MODIFIED: Register new resource/data source
    - Add resource.NewFloatingIPResource to Resources()
    - Add data.NewFloatingIPsDataSource to DataSources()

docs/                                   # AUTO-GENERATED by `make generate`
├── resources/
│   └── floating_ip.md                 # Resource documentation
└── data-sources/
    └── floating_ips.md                # Data source documentation

examples/
├── resources/
│   └── zillaforge_floating_ip/        # NEW: Example configurations
│       └── resource.tf
└── data-sources/
    └── zillaforge_floating_ips/       # NEW: Example queries
        └── data-source.tf
```

**Structure Decision**: Follows existing Terraform provider conventions with internal/vps/{model,resource,data} organization. Models are shared in internal/vps/model package to avoid circular dependencies and enable reuse between resource and data source. Tests are colocated with implementation (*_test.go pattern). Documentation is auto-generated per constitution requirement.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**Status**: N/A - No constitutional violations

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
