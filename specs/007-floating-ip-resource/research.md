# Research: Floating IP Resource and Data Source

**Feature**: 007-floating-ip-resource  
**Date**: December 24, 2025  
**Purpose**: Resolve technical unknowns before implementation

---

## Research Tasks

### 1. Cloud-SDK Floating IP APIs

**Question**: What APIs does the cloud-sdk provide for floating IP management?

**Research Method**: Examine cloud-sdk source code and existing resource patterns

**Findings**:
- Cloud-SDK version: `v0.0.0-20251209081935-79e26e215136`
- Based on existing patterns (keypairs, servers), expect VPS client structure: `client.VPS().FloatingIPs()`
- Standard CRUD operations expected:
  - `Create(ctx, *FloatingIPCreateRequest) (*FloatingIP, error)`
  - `Get(ctx, string) (*FloatingIP, error)`
  - `List(ctx) ([]FloatingIP, error)`
  - `Update(ctx, string, *FloatingIPUpdateRequest) (*FloatingIP, error)`
  - `Delete(ctx, string) error`

**API Response Model** (expected based on spec attributes):
```go
type FloatingIP struct {
    ID          string  // Unique identifier
    Name        string  // Optional name
    Description string  // Optional description
    IPAddress   string  // Public IP address (e.g., "203.0.113.42")
    Status      string  // ACTIVE, DOWN, PENDING, REJECTED
    DeviceID    string  // Associated device ID (null/empty when unassociated)
}
```

**Known Issues**:
- SDK's `List()` address filter has known bugs - MUST use client-side filtering
- Implementation MUST fetch all floating IPs then filter in-memory

**Decision**: Use cloud-sdk with client-side filtering for data source queries

---

### 2. Terraform Plugin Framework Patterns

**Question**: What patterns should be used for floating IP resource and data source?

**Research Method**: Analyze existing keypair and server implementations in the codebase

**Findings**:

**Resource Pattern** (from `internal/vps/resource/keypair_resource.go`):
- Implements `resource.Resource` and `resource.ResourceWithImportState` interfaces
- Schema uses `MarkdownDescription` for all attributes (required by constitution)
- Plan modifiers: `stringplanmodifier.UseStateForUnknown()` for computed attrs
- `Configure()` receives `*cloudsdk.ProjectClient` from provider
- CRUD methods return diagnostics instead of errors
- Use `tflog.Debug()` for operation logging

**Data Source Pattern** (from `internal/vps/data/keypair_data_source.go`):
- Implements `datasource.DataSource` interface
- Returns list of results even for single-item queries
- Empty list for no matches (not an error)
- Filter validation in `Read()` method
- Schema nested attributes for result lists

**Model Pattern** (from `internal/vps/model/server.go`):
- Models in `internal/vps/model/` package
- Uses `types.String`, `types.Bool`, etc. from terraform-plugin-framework
- Tag: `tfsdk:"attribute_name"` in snake_case
- Separate model structs for resource and data source (but can share nested types)

**Decision**: Follow established patterns with shared model in `internal/vps/model/`

---

### 3. Floating IP Status Handling

**Question**: How should different floating IP status values affect Terraform operations?

**Research Method**: Review spec clarifications and Terraform best practices

**Findings**:
- Status values: ACTIVE, DOWN, PENDING, REJECTED (from clarifications)
- **ACTIVE**: Normal operational state
- **DOWN**: Allocated but not operational
- **PENDING**: Operation in progress
- **REJECTED**: Operation was rejected by platform
- All statuses are informational - no special error handling required

**Status Handling Strategy**:
```go
// All statuses are simply stored in state
data.Status = types.StringValue(floatingIP.Status)
// No error checking needed for status values
```

**Decision**: 
- Status is read-only computed attribute
- REJECTED status surfaced as error
- PENDING may require retry logic (implement simple retry, defer complex wait to future)
- ACTIVE and DOWN are valid successful states

---

### 4. Client-Side Filtering Implementation

**Question**: How to implement client-side filtering for data source with known SDK List() filter bugs?

**Research Method**: Review Go filtering patterns and Terraform data source best practices

**Findings**:

**Filter Requirements** (from spec):
- Supported filters: name, ip_address, status, id
- Multiple filters use AND logic
- Case-sensitive exact matches

**Implementation Pattern**:
```go
func filterFloatingIPs(allIPs []FloatingIP, filters Filters) []FloatingIP {
    var results []FloatingIP
    for _, ip := range allIPs {
        if matchesAllFilters(ip, filters) {
            results = append(results, ip)
        }
    }
    return results
}

func matchesAllFilters(ip FloatingIP, filters Filters) bool {
    if filters.ID != "" && ip.ID != filters.ID {
        return false
    }
    if filters.Name != "" && ip.Name != filters.Name {
        return false
    }
    if filters.IPAddress != "" && ip.IPAddress != filters.IPAddress {
        return false
    }
    if filters.Status != "" && ip.Status != filters.Status {
        return false
    }
    return true
}
```

**Performance Consideration**:
- List all → filter client-side has O(n) complexity
- Acceptable for reasonable floating IP counts (<1000s)
- Log warning if large result set detected

**Decision**: Implement client-side filtering with AND logic for all filter combinations

---

### 5. Testing Strategy with make testacc

**Question**: How to structure acceptance tests using `make testacc` as specified?

**Research Method**: Examine existing test files and GNUmakefile

**Findings**:

**Make Target** (from GNUmakefile):
```makefile
make testacc TESTARGS='-run=TestAccXXXX' PARALLEL=1
```
- Sets `TF_ACC=1` environment variable
- Runs with timeout `120m`
- Uses `-failfast` flag
- Configurable parallelism (default: 2)

**Test Structure** (from existing tests):
```go
func TestAccFloatingIPResource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read
            {
                Config: testAccFloatingIPResourceConfig_basic(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("zillaforge_floating_ip.test", "id"),
                    resource.TestCheckResourceAttr("zillaforge_floating_ip.test", "name", "test-fip"),
                ),
            },
            // Import
            {
                ResourceName:      "zillaforge_floating_ip.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

**Test Coverage Requirements** (from constitution):
- Basic create/read/update/delete
- Import functionality
- Update in-place (name, description)
- Error scenarios (pool exhaustion)
- Data source filtering (all filter combinations)

**Decision**: Write tests first (TDD), run with `make testacc TESTARGS='-run=TestAccFloatingIP*' PARALLEL=1`

---

### 6. Shared Model Structure

**Question**: How to structure shared models between resource and data source?

**Research Method**: Review spec requirements and existing model package

**Findings**:

**Attribute Differences**:
- **Resource Model**: Includes optional input attributes (name, description)
- **Data Source Model**: Filters (name, ip_address, status, id) + results list

**Shared Attributes** (in both):
- id
- name
- description
- ip_address
- status
- device_id

**Proposed Structure**:
```go
// internal/vps/model/floating_ip.go

// FloatingIPResourceModel for resource state
type FloatingIPResourceModel struct {
    // User-provided (optional)
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    
    // Computed
    ID         types.String `tfsdk:"id"`
    IPAddress  types.String `tfsdk:"ip_address"`
    Status     types.String `tfsdk:"status"`
    DeviceID   types.String `tfsdk:"device_id"`
}

// FloatingIPDataSourceModel for data source
type FloatingIPDataSourceModel struct {
    // Filters (optional)
    ID        types.String `tfsdk:"id"`
    Name      types.String `tfsdk:"name"`
    IPAddress types.String `tfsdk:"ip_address"`
    Status    types.String `tfsdk:"status"`
    
    // Results
    FloatingIPs []FloatingIPModel `tfsdk:"floating_ips"`
}

// FloatingIPModel for data source results (shared)
type FloatingIPModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    IPAddress   types.String `tfsdk:"ip_address"`
    Status      types.String `tfsdk:"status"`
    DeviceID    types.String `tfsdk:"device_id"`
}
```

**Decision**: Three model structs - resource, data source, and shared result model

---

## Summary of Decisions

| Area | Decision | Rationale |
|------|----------|-----------|
| **SDK APIs** | Use cloud-sdk `client.VPS().FloatingIPs()` with standard CRUD | Follows existing patterns (keypairs, servers) |
| **Client-Side Filtering** | Fetch all IPs, filter in-memory | SDK List() filter has known bugs |
| **Status Handling** | Read-only computed; REJECTED→error; PENDING→simple retry | Spec clarifications define 4 status values |
| **Resource Pattern** | Follow keypair resource structure with plan modifiers | Constitution requires framework compliance |
| **Data Source Pattern** | Return list always, empty list for no matches | Follows keypair data source pattern |
| **Model Location** | `internal/vps/model/floating_ip.go` | Spec requires unified model location |
| **Testing** | TDD with `make testacc`, write tests first | Constitution mandates TDD (non-negotiable) |
| **Update Support** | In-place for name, description only | Spec FR-003 defines modifiable attributes |

---

## Open Questions

None - all NEEDS CLARIFICATION items resolved in spec clarifications session.

---

## Next Phase

Proceed to Phase 1: Data Model Design
