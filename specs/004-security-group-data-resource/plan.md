# Implementation Plan: Security Group Data Source and Resource

**Branch**: `004-security-group-data-resource` | **Date**: 2025-12-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-security-group-data-resource/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement Terraform data source and resource for ZillaForge security groups with stateful firewall rules. Enables infrastructure engineers to declaratively manage network access control for VPS instances through security groups containing inbound/outbound rules (protocol, port range, CIDR). Supports CRUD operations, import, and querying via data source. Uses `github.com/Zillaforge/cloud-sdk` for API interactions, following existing provider patterns established in keypair/network/flavor implementations.

## Technical Context

**Language/Version**: Go 1.22.4  
**Primary Dependencies**: 
- `github.com/Zillaforge/cloud-sdk` (v0.0.0-20251209081935-79e26e215136) - ZillaForge API client
- `github.com/hashicorp/terraform-plugin-framework` (v1.14.1) - Terraform Plugin Framework
- `github.com/hashicorp/terraform-plugin-testing` (v1.11.0) - Acceptance testing framework
- `github.com/hashicorp/terraform-plugin-log` (v0.9.0) - Structured logging

**Storage**: N/A (stateless provider, all state managed by Terraform and ZillaForge API)  
**Testing**: Terraform Plugin Testing framework with acceptance tests (`make testacc`), table-driven unit tests for validation logic  
**Target Platform**: Linux/macOS/Windows (cross-platform Terraform provider binary)  
**Project Type**: Single project (Terraform provider plugin)  
**Performance Goals**: 
- Data source queries < 5 seconds for 100 security groups (per SC-002)
- Rule updates < 30 seconds (per SC-003)
- Minimal API calls (batch operations where possible, cache provider config)

**Constraints**: 
- Terraform Plugin Framework semantics (null/unknown/computed value handling)
- Stateful firewall behavior (return traffic auto-allowed, per FR-027)
- Deletion blocked if attached to instances (per FR-007, clarification answer B)
- API rate limits (implement retry with exponential backoff)

**Scale/Scope**: 
- Support accounts with up to 100+ security groups
- Security groups with 50+ rules each
- Multiple security groups per VPS instance (union evaluation, per FR-026)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Code Quality & Framework Compliance
- ✅ **Plugin Framework**: Will use Terraform Plugin Framework v1.14.1 (not SDK v2)
- ✅ **Interface Implementation**: Resource will implement `resource.Resource` and `resource.ResourceWithImportState`; data source implements `datasource.DataSource`
- ✅ **MarkdownDescription**: All schema attributes will have markdown descriptions for documentation generation
- ✅ **Input Validation**: Schema will validate protocol (TCP/UDP/ICMP/ANY), port ranges (1-65535), CIDR notation
- ✅ **Diagnostics**: All errors will include actionable context (e.g., "Security group attached to instances: [list]")
- ✅ **Godoc Comments**: All exported types will be documented

### Test-Driven Development (NON-NEGOTIABLE)
- ✅ **Acceptance Tests First**: Tests written before implementation (red-green-refactor cycle)
- ✅ **CRUD Coverage**: Create, Read, Update, Delete operations all tested
- ✅ **Import Testing**: Import functionality tested with `ImportStateVerify: true`
- ✅ **Data Source Testing**: Read operations validated for single lookup and list scenarios
- ✅ **Validation Testing**: Unit tests for CIDR validation, port range validation, protocol validation

### User Experience Consistency
- ✅ **Actionable Errors**: "Cannot delete security group sg-xxx: attached to instances [i-abc, i-def]. Detach first."
- ✅ **Error vs Warning**: Errors for failures, warnings for delete operations on attached groups
- ✅ **Snake Case Attributes**: `security_group`, `port_range`, `source_cidr` (not camelCase)
- ✅ **Required/Optional Clear**: Name (required), description (optional), rules (optional but computed)
- ✅ **Computed Attributes**: ID, fingerprint marked as computed
- ✅ **Timeouts**: Configurable for long-running operations (if needed)
- ✅ **Import Examples**: Documentation includes `terraform import zillaforge_security_group.example sg-12345`
- ✅ **Semantic Versioning**: New feature, minor version bump

### Performance & Resource Efficiency
- ✅ **Minimize API Calls**: Batch rule operations where possible; Read only if state is stale
- ✅ **Context Timeouts**: Respect ctx.Done() in all API calls
- ✅ **Pagination**: Support pagination if API returns large security group lists
- ✅ **Provider Config Cached**: Client initialized once, shared across resources
- ✅ **Structured Logging**: Debug logs for API calls, Info for key events, Error for failures
- ✅ **Retry Logic**: HTTP client uses exponential backoff for transient failures (cloud-sdk handles this)

**GATE STATUS**: ✅ **PASS** - All constitution requirements satisfied. No violations to justify.

---

## Post-Design Constitution Re-Check

*Re-evaluated after Phase 1 design completion*

### Code Quality & Framework Compliance
- ✅ **Schema Contracts**: All attributes documented with MarkdownDescription in [contracts/](contracts/)
- ✅ **Nested Attributes**: Uses `ListNestedAttribute` for rules (validated in data-model.md)
- ✅ **Validation**: Custom validators defined for port ranges, CIDR blocks, protocols (research.md section 3-4)
- ✅ **Plan Modifiers**: `UseStateForUnknown` for ID, `RequiresReplace` for name (schema contract)

### Test-Driven Development
- ✅ **Acceptance Test Scenarios**: Defined in quickstart.md (create, read, update, delete, import)
- ✅ **Test Coverage Plan**: CRUD + import + data source queries (schema contracts specify all operations)
- ✅ **Red-Green-Refactor**: Workflow established (tests before implementation)

### User Experience Consistency  
- ✅ **Error Messages**: Actionable diagnostics specified (e.g., "attached to instances [i-abc, i-def]" - schema contract)
- ✅ **Attribute Naming**: snake_case used throughout (security_group, ingress_rules, port_range)
- ✅ **Import Documentation**: Examples with concrete IDs in quickstart.md
- ✅ **Breaking Changes**: None (new feature, no existing resources affected)

### Performance & Resource Efficiency
- ✅ **API Call Optimization**: Research determined full replacement strategy for rules (optimizable later)
- ✅ **Pagination Support**: Data source implementation includes pagination handling (schema contract)
- ✅ **Retry Logic**: Cloud-SDK handles exponential backoff (confirmed in research)
- ✅ **Context Timeouts**: All API calls respect ctx.Done() (standard pattern)

**FINAL GATE STATUS**: ✅ **PASS** - Design fully compliant with constitution. Ready for Phase 2 task breakdown.

## Project Structure

### Documentation (this feature)

```text
specs/004-security-group-data-resource/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── security-group-resource-schema.md
│   ├── security-group-data-source-schema.md
│   └── security-rule-schema.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/vps/
├── data/
│   ├── flavor_data_source.go          # Existing
│   ├── keypair_data_source.go         # Existing
│   ├── network_data_source.go         # Existing
│   ├── securitygroup_data_source.go   # NEW: Security group data source
│   ├── flavor_data_source_test.go     # Existing
│   ├── keypair_data_source_test.go    # Existing
│   ├── network_data_source_test.go    # Existing
│   └── securitygroup_data_source_test.go  # NEW: Acceptance tests
└── resource/
    ├── keypair_resource.go            # Existing
    ├── keypair_resource_test.go       # Existing
    ├── securitygroup_resource.go      # NEW: Security group resource
    └── securitygroup_resource_test.go # NEW: Acceptance tests

docs/
├── data-sources/
│   └── security_groups.md             # NEW: Generated by tfplugindocs
└── resources/
    └── security_group.md              # NEW: Generated by tfplugindocs

examples/
├── data-sources/
│   └── zillaforge_security_groups/
│       └── data-source.tf             # NEW: Example queries
└── resources/
    └── zillaforge_security_group/
        ├── resource.tf                # NEW: Example create
        └── import.sh                  # NEW: Import example
```

**Structure Decision**: Single project (Terraform provider). Following established pattern where VPS resources/data sources live in `internal/vps/resource/` and `internal/vps/data/` respectively. Tests colocated with implementation (`*_test.go`). Documentation generated from schema descriptions via `tfplugindocs`, examples written as HCL in `examples/`.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations - constitution check passed. All requirements align with established provider patterns and Terraform Plugin Framework best practices.
