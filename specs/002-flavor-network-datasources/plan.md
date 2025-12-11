# Implementation Plan: Flavor and Network Data Sources

**Branch**: `002-flavor-network-datasources` | **Date**: 2025-12-11 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-flavor-network-datasources/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement two Terraform data sources (`zillaforge_flavors` and `zillaforge_networks`) that query available compute flavors and networks from Zillaforge cloud API using the `github.com/Zillaforge/cloud-sdk`. Both data sources return lists of results with optional filtering support (exact name match, minimum CPU/memory for flavors, name/status for networks). Filters use AND logic when combined. Empty result sets return empty lists (not errors).

## Technical Context

**Language/Version**: Go 1.22.4  
**Primary Dependencies**: 
- `github.com/hashicorp/terraform-plugin-framework v1.14.1` (Terraform Plugin Framework)
- `github.com/Zillaforge/cloud-sdk v0.0.0-20251209081935-79e26e215136` (Zillaforge API SDK)
- `github.com/hashicorp/terraform-plugin-testing v1.11.0` (acceptance testing)

**Storage**: N/A (read-only data sources, no state persistence)  
**Testing**: 
- Acceptance tests using `terraform-plugin-testing` framework
- Unit tests for filter logic and type conversions
- go test with standard Go testing package

**Target Platform**: Terraform providers (Linux, macOS, Windows cross-compiled binaries)  
**Project Type**: Single project (Terraform provider plugin)  
**Performance Goals**: 
- Data source queries < 3 seconds (per SC-002, SC-005)
- Support SDK pagination for large result sets
- Minimize API calls (single list call per data source read)

**Constraints**: 
- Must use Terraform Plugin Framework (not SDK v2) per constitution
- Must implement TDD workflow (acceptance tests before implementation)
- Must handle SDK pagination transparently
- Must use SDK default timeouts (no explicit timeout configuration)
- All attributes must have MarkdownDescription for documentation generation

**Scale/Scope**: 
- 2 data sources (flavors, networks)
- 6 filter attributes total (3 per data source)
- 11 computed attributes total (6 for flavors, 5 for networks)
- Expected result sets: 10-100 flavors, 10-50 networks per project

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Framework Compliance
- [x] All data sources use Terraform Plugin Framework (not Plugin SDK v2)
- [x] Schema definitions include MarkdownDescription for every attribute
- [x] Read methods validate inputs and return actionable diagnostics
- [x] State management handles null, unknown, and computed values correctly
- [x] All exported types have godoc comments

**Status**: ✅ PASS (verified in schema contracts - all attributes have MarkdownDescription, filter attributes marked Optional, result attributes marked Computed)

### II. Test-Driven Development
- [x] Acceptance tests written before implementation code
- [x] Tests fail initially (red), then pass after implementation (green)
- [x] Data source Read operations have acceptance test coverage
- [x] Filter behavior tested for all combinations
- [x] Empty result sets tested
- [x] API error scenarios tested with proper diagnostic messages

**Status**: ✅ PASS (verified in schema contracts - 8 test cases per data source covering basic queries, filtering, empty results, and error conditions)

### III. User Experience Consistency
- [x] Error messages are actionable (authentication failures, API errors)
- [x] Diagnostics use appropriate severity levels (Error for API failures)
- [x] Attribute naming follows Terraform conventions (snake_case: `vcpus`, `memory`, `status`)
- [x] Required vs Optional attributes clearly defined (all filters Optional, all results Computed)
- [x] Computed attributes explicitly marked in schema

**Status**: ✅ PASS (verified in data-model.md and schema contracts - consistent naming, clear attribute types, actionable error messages)

### IV. Performance & Resource Efficiency
- [x] API calls minimized (single list call per data source read)
- [x] SDK pagination handled transparently
- [x] SDK default timeouts used (no custom timeout configuration)
- [x] Context timeouts respected and propagated to SDK client
- [x] Logs use appropriate levels (Debug for API calls, Info for key events, Error for failures)

**Status**: ✅ PASS (verified in research.md - single List() call per read, SDK handles pagination internally, uses SDK default timeouts)

**Gate Result**: ✅ PASS - All constitutional requirements validated in Phase 1 design. Implementation must maintain compliance.

## Project Structure

### Documentation (this feature)

```text
specs/002-flavor-network-datasources/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── flavors-schema.md
│   └── networks-schema.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/provider/
├── flavor_data_source.go           # zillaforge_flavors data source implementation
├── flavor_data_source_test.go      # Acceptance and unit tests for flavors
├── network_data_source.go          # zillaforge_networks data source implementation
├── network_data_source_test.go     # Acceptance and unit tests for networks
├── provider.go                      # Provider registration (add new data sources)
└── provider_test.go                 # Provider-level test configuration

examples/data-sources/
├── zillaforge_flavor/
│   └── data-source.tf               # Usage examples for flavors data source
└── zillaforge_network/
    └── data-source.tf               # Usage examples for networks data source

docs/data-sources/
├── flavors.md                       # Generated documentation (tfplugindocs)
└── networks.md                      # Generated documentation (tfplugindocs)
```

**Structure Decision**: Single project structure following Terraform provider conventions. Data sources are implemented in `internal/provider/<resource>_data_source.go` files with colocated tests (`<resource>_data_source_test.go`). Examples use full data source name in directory path (`examples/data-sources/zillaforge_flavors/`). Documentation follows HashiCorp provider layout standards for registry publishing.

## Complexity Tracking

> **No violations detected. This section intentionally left empty.**

All design decisions comply with constitutional principles. No additional complexity justification required.

## Agent Context Update

**Updated**: 2025-12-11 (Post-Phase 1 Design)

### Key Design Patterns Discovered

1. **SDK Integration Pattern**:
   - Use `vpsClient.Flavors().List(ctx, nil)` for flavor retrieval
   - Use `vpsClient.Networks().List(ctx, nil)` for network retrieval
   - VPS SDK clients use `nil` for options parameter (no server-side filtering support)
   - SDK returns full lists without pagination in current implementation

2. **Hybrid Filtering Strategy**:
   - Server-side: Not supported by SDK (List methods accept nil options)
   - Client-side: All filtering implemented in Terraform data source code
   - Filter logic: AND semantics for multiple filters
   - String matching: Exact case-sensitive comparison for name filters

3. **Data Transformations**:
   - Memory conversion: SDK returns MiB, convert to GB (divide by 1024)
   - Field mapping: SDK `RAM` → Terraform `memory`
   - Type conversions: SDK integers → Terraform Int64 types
   - Network status: Pass through as-is (e.g., "ACTIVE", "ERROR")

4. **Error Handling Patterns**:
   - SDK client errors: Convert to Terraform Error diagnostics
   - Authentication failures: Return actionable diagnostic messages
   - Empty results: Return empty list (not an error condition)
   - Type conversion errors: Should not occur with SDK types (defensive programming)

5. **State Management**:
   - All filter attributes: Optional + schema validators
   - All result attributes: Computed
   - ID generation: Use timestamp-based ID for data source reads
   - Null handling: Filters support null/unknown states during plan phase

### Technology Additions

- **github.com/Zillaforge/cloud-sdk**: VPS client for API operations
  - Methods: `vpsClient.Flavors().List()`, `vpsClient.Networks().List()`
  - Types: Flavor, Network entities with standard fields
  - Error handling: Returns Go errors, convert to Terraform diagnostics

### Integration Points

1. **Provider Configuration**: 
   - VPS client initialized in provider Configure method
   - Client passed to data sources via ConfigureFunc
   - Client uses provider-level authentication (API key/endpoint)

2. **Schema Design**:
   - Use `schema.SingleNestedAttribute` for filter blocks
   - Use `schema.ListNestedAttribute` for result lists
   - All attributes include MarkdownDescription for doc generation
   - Validators: int64validator.AtLeast(1) for vcpus, memory, disk

3. **Testing Requirements**:
   - Acceptance tests use real API calls (requires test credentials)
   - Mock SDK responses for unit tests of filter logic
   - Test cases: basic query, name filter, multiple filters, empty results, API errors
