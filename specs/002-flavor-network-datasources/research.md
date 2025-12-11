# Research: Flavor and Network Data Sources

**Feature**: 002-flavor-network-datasources  
**Date**: 2025-12-11  
**Purpose**: Resolve technical unknowns before design phase

## Research Questions & Findings

### Q1: How to query flavors from Zillaforge cloud-sdk?

**Decision**: Use `vpsClient.Flavors().List(ctx, opts)` method

**Rationale**: 
- SDK provides direct `List()` method on flavors resource
- Returns `[]*flavors.Flavor` slice (not paginated wrapper)
- Supports server-side filtering via `ListFlavorsOptions`
- No pagination support (returns all results in single response)

**Implementation Pattern**:
```go
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    "github.com/Zillaforge/cloud-sdk/models/vps/flavors"
)

// Get VPS client from provider-configured SDK client
vpsClient := projectClient.VPS()

// List with optional server-side filtering
opts := &flavors.ListFlavorsOptions{
    Name: "flavor-name",  // Partial match
    Tags: []string{"tag1", "tag2"},
}
flavorList, err := vpsClient.Flavors().List(ctx, opts)
```

**Alternatives Considered**:
- Client-side filtering only → Rejected: SDK supports server-side filtering via query parameters
- Individual flavor Get() calls → Rejected: Inefficient, no bulk query support needed

---

### Q2: How to query networks from Zillaforge cloud-sdk?

**Decision**: Use `vpsClient.Networks().List(ctx, opts)` method

**Rationale**:
- SDK provides `List()` method on networks resource
- Returns `[]*NetworkResource` slice with sub-resource operations
- Supports server-side filtering via `ListNetworksOptions`
- No pagination support (returns all results)

**Implementation Pattern**:
```go
import "github.com/Zillaforge/cloud-sdk/models/vps/networks"

opts := &networks.ListNetworksOptions{
    Name:   "network-name",
    Status: "ACTIVE",
}
networkList, err := vpsClient.Networks().List(ctx, opts)
```

**Alternatives Considered**:
- Use router or subnet list endpoints → Rejected: Networks endpoint is the canonical source

---

### Q3: What field mappings are needed for flavor data source?

**Decision**: Map SDK `flavors.Flavor` fields to Terraform schema

| Terraform Attribute | SDK Field | Type | Notes |
|---------------------|-----------|------|-------|
| `id` | `ID` | string | Required |
| `name` | `Name` | string | Required |
| `vcpus` | `VCPU` | int64 | Required (type conversion needed) |
| `memory` | `Memory` | int64 | Required, SDK uses MiB, spec requires GB (divide by 1024) |
| `disk` | `Disk` | int64 | Optional, SDK uses GiB (matches spec) |
| `description` | `Description` | string | Optional |

**Rationale**:
- SDK `Memory` field is in MiB (mebibytes), specification requires GB
- Conversion: `memory_gb = memory_mib / 1024` (integer division acceptable)
- SDK `Disk` already in GiB, no conversion needed
- Type conversions from `int` to `int64` required for Terraform schema

**Ignored SDK Fields**: `GPU`, `Public`, `Tags`, `ProjectIDs`, `AZ`, timestamps - not in specification

---

### Q4: What field mappings are needed for network data source?

**Decision**: Map SDK `networks.Network` fields to Terraform schema

| Terraform Attribute | SDK Field | Type | Notes |
|---------------------|-----------|------|-------|
| `id` | `ID` | string | Required |
| `name` | `Name` | string | Required |
| `cidr` | `CIDR` | string | Required |
| `status` | `Status` | string | Optional |
| `description` | `Description` | string | Optional |

**Rationale**:
- Direct field mapping, no type conversions needed
- All fields are strings, matching Terraform schema requirements
- Status is optional in SDK but should always be present in API responses

**Ignored SDK Fields**: `Bonding`, `Gateway`, `Shared`, `SubnetID`, `UserID`, `ProjectID`, nested objects - not in specification

---

### Q5: How to implement filter logic (server-side vs. client-side)?

**Decision**: Hybrid approach - leverage SDK server-side filtering where available, add client-side filters for specification requirements

**SDK Server-Side Filtering**:

**Flavors** (`ListFlavorsOptions`):
- `Name` (string) - partial match support
- `Tags` ([]string) - multiple tags
- `Public` (*bool) - true/false/nil
- `ResizeServerID` (string) - resize compatibility

**Networks** (`ListNetworksOptions`):
- `Name` (string) - name filter
- `Status` (string) - status filter
- `UserID`, `RouterID` (string) - relationship filters
- `Detail` (*bool) - detail level

**Client-Side Filtering Required**:

**Flavors**:
- **Exact name match**: SDK supports partial match, need exact match per clarification
- **Minimum vcpus**: SDK doesn't support numeric comparisons
- **Minimum memory**: SDK doesn't support numeric comparisons

**Networks**:
- **Exact name match**: SDK supports name filter but unclear if exact or partial
- **Status match**: SDK supports status filtering

**Implementation Strategy**:
1. Use SDK server-side filters to reduce result set (optimization)
2. Apply client-side filters in Read() method for exact specification compliance
3. Filter logic uses AND semantics (all filters must match per clarification)

**Example**:
```go
// Step 1: Server-side filter (optimization)
opts := &flavors.ListFlavorsOptions{}
if nameFilter != "" {
    opts.Name = nameFilter  // Partial match, will narrow results
}
allFlavors, err := vpsClient.Flavors().List(ctx, opts)

// Step 2: Client-side exact filters
filtered := []*flavors.Flavor{}
for _, flavor := range allFlavors {
    // Exact name match
    if nameFilter != "" && flavor.Name != nameFilter {
        continue
    }
    // Minimum vcpus
    if vcpusMin > 0 && flavor.VCPU < vcpusMin {
        continue
    }
    // Minimum memory (convert MiB to GB)
    if memoryMinGB > 0 && (flavor.Memory / 1024) < memoryMinGB {
        continue
    }
    filtered = append(filtered, flavor)
}
```

**Rationale**:
- Server-side filtering reduces network payload and improves performance
- Client-side filtering ensures specification compliance (exact match, numeric comparisons)
- AND logic: filter stack short-circuits on first non-match

**Alternatives Considered**:
- Pure server-side → Rejected: SDK doesn't support all required filters
- Pure client-side → Rejected: Inefficient for large flavor/network lists

---

### Q6: How to handle SDK errors in Terraform diagnostics?

**Decision**: Map SDK errors to Terraform diagnostics with actionable messages

**SDK Error Type**: `types.SDKError`
```go
type SDKError struct {
    StatusCode int                    // HTTP status (0 for client errors)
    ErrorCode  int                    // API error code
    Message    string                 // Human-readable message
    Meta       map[string]interface{} // Additional context
    Cause      error                  // Wrapped error
}
```

**Error Mapping Strategy**:

| HTTP Status | Terraform Severity | Diagnostic Summary | Detail Message |
|-------------|-------------------|-------------------|----------------|
| 401 | Error | Authentication failed | Check api_key configuration and ensure token is valid |
| 403 | Error | Permission denied | Insufficient permissions to list flavors/networks for this project |
| 404 | Error | Project not found | Project ID/code not found, verify provider configuration |
| 429 | Error | Rate limit exceeded | Too many requests, SDK retry logic exhausted |
| 500/502/503/504 | Error | API unavailable | Zillaforge API returned server error, retry operation |
| 0 (client-side) | Error | Network error | Failed to connect to Zillaforge API: {cause} |

**Implementation Pattern**:
```go
import (
    "errors"
    "github.com/hashicorp/terraform-plugin-framework/diag"
    "github.com/Zillaforge/cloud-sdk/types"
)

flavors, err := vpsClient.Flavors().List(ctx, opts)
if err != nil {
    var sdkErr *types.SDKError
    if errors.As(err, &sdkErr) {
        switch sdkErr.StatusCode {
        case 401:
            resp.Diagnostics.AddError(
                "Authentication Failed",
                "Check api_key configuration and ensure token is valid. "+
                "Error: "+sdkErr.Message,
            )
        case 403:
            resp.Diagnostics.AddError(
                "Permission Denied",
                "Insufficient permissions to list flavors for this project. "+
                "Verify project access and API key permissions.",
            )
        default:
            resp.Diagnostics.AddError(
                "API Error",
                fmt.Sprintf("Failed to list flavors: %s (HTTP %d)",
                    sdkErr.Message, sdkErr.StatusCode),
            )
        }
    } else {
        resp.Diagnostics.AddError(
            "Request Failed",
            "Failed to list flavors: "+err.Error(),
        )
    }
    return
}
```

**Rationale**:
- Terraform users need actionable guidance, not raw HTTP errors
- AddError() severity appropriate (prevents plan/apply continuation)
- Error messages explain what failed and how to fix it
- Preserves underlying error details for debugging

**Alternatives Considered**:
- AddWarning() for API errors → Rejected: List failures should block plan
- Generic "API error" message → Rejected: Not actionable per UX principle

---

### Q7: How to handle pagination for large result sets?

**Decision**: No pagination handling needed - SDK returns complete result sets

**Rationale**:
- SDK `Flavors().List()` and `Networks().List()` do not support pagination
- API may paginate, but SDK wraps and returns all results in single slice
- No `offset`, `limit`, or `page` parameters in `ListFlavorsOptions` or `ListNetworksOptions`
- Other SDK modules (IAM Projects) do support pagination via `ListProjectsOptions`, but VPS module does not

**Performance Considerations**:
- Expected result sets: 10-100 flavors, 10-50 networks per project (per spec)
- Single API call acceptable for these volumes
- If pagination is added to SDK later, no Terraform schema changes needed (transparent)

**Alternatives Considered**:
- Implement pagination via multiple SDK calls → Rejected: SDK doesn't expose pagination controls
- Limit result set size → Rejected: Would break functionality, no specification requirement

---

### Q8: Best practices for Terraform Plugin Framework data source implementation

**Decision**: Follow framework idioms for list-based data sources

**Schema Pattern**:
```go
schema.Schema{
    Attributes: map[string]schema.Attribute{
        // Filter attributes (Optional)
        "name": schema.StringAttribute{
            MarkdownDescription: "Filter flavors by exact name match",
            Optional: true,
        },
        "vcpus": schema.Int64Attribute{
            MarkdownDescription: "Minimum number of virtual CPUs",
            Optional: true,
        },
        
        // Result attribute (Computed)
        "flavors": schema.ListNestedAttribute{
            MarkdownDescription: "List of matching flavors",
            Computed: true,
            NestedObject: schema.NestedAttributeObject{
                Attributes: map[string]schema.Attribute{
                    "id": schema.StringAttribute{
                        MarkdownDescription: "Unique flavor identifier",
                        Computed: true,
                    },
                    // ... other flavor attributes
                },
            },
        },
    },
}
```

**Read Method Pattern**:
```go
func (d *FlavorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data FlavorsDataSourceModel
    
    // Read filter configuration
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }
    
    // Get SDK client from provider
    client := req.ProviderData.(*ZillaforgeClient)
    vpsClient := client.ProjectClient.VPS()
    
    // List from API
    flavors, err := vpsClient.Flavors().List(ctx, nil)
    if err != nil {
        // Handle error (see Q6)
        return
    }
    
    // Apply client-side filters (see Q5)
    filtered := applyFlavorFilters(flavors, data)
    
    // Convert to Terraform model
    data.Flavors = convertFlavorsToModel(filtered)
    
    // Save to state
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Rationale**:
- Filters as top-level Optional attributes (user input)
- Results in nested list attribute (computed output)
- MarkdownDescription on all attributes (enables tfplugindocs generation)
- All result attributes marked Computed (read-only from API)
- Filter attributes marked Optional (allow unfiltered queries)

**Alternatives Considered**:
- Filters in nested block → Rejected: Top-level attributes more idiomatic
- Single object return type → Rejected: Specification requires list return (clarification Q1)

---

## Technology Decisions Summary

### Chosen Technologies
- **SDK Client**: `github.com/Zillaforge/cloud-sdk` (already in go.mod)
- **API Access**: `projectClient.VPS().Flavors().List()` and `projectClient.VPS().Networks().List()`
- **Filtering**: Hybrid server-side + client-side approach
- **Error Handling**: SDK error type assertions with Terraform diagnostics mapping
- **Testing**: terraform-plugin-testing framework with acceptance tests

### Key Constraints
- No pagination support in SDK (acceptable for expected data volumes)
- Memory unit conversion required: SDK MiB → Terraform GB
- Exact match filtering requires client-side logic (SDK supports partial match)
- SDK retry logic handles transient errors (429, 5xx) automatically

### Integration Points
- Provider configuration supplies authenticated SDK client
- VPS client scoped to project (from provider config)
- Context propagation from Terraform to SDK for timeout handling
- Error wrapping for diagnostic clarity

## Remaining Unknowns

**None.** All technical questions resolved. Ready for Phase 1 design.
