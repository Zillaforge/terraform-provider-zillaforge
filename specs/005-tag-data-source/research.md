# Research: Image Data Source

**Feature**: zillaforge_images data source  
**Date**: December 15, 2025  
**Status**: Complete

## Research Questions & Findings

### Q1: How to query VM images (Tags) from Zillaforge cloud-sdk?

**Decision**: Use `vrmClient.Tags().List(ctx, opts)` method from VRM module. For repository-scoped queries, prefer `vrmClient.Repositories().Get()/List()` and then `RepositoryResource.Tags().List()` to fetch tags for a specific repository efficiently.

**Rationale**:
- SDK provides `List()` method on Tags resource in VRM module
- Returns `[]*common.Tag` with embedded Repository information
- Supports server-side filtering via `ListTagsOptions` (limit, offset, where, namespace)
- API endpoint: `GET /vrm/api/v1/project/{project-id}/tags`
- Tags represent repository:tag pairs used to create VMs

**Implementation Pattern**:
```go
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    tagmodels "github.com/Zillaforge/cloud-sdk/models/vrm/tags"
)

// Get VRM client from provider-configured SDK client
vrmClient := projectClient.VRM()

// List with optional filtering
opts := &tagmodels.ListTagsOptions{
    Limit:  1000,  // Server max limit
    Offset: 0,
    Where:  []string{},  // Filter conditions
}
tagList, err := vrmClient.Tags().List(ctx, opts)
```

**SDK Tag Structure** (`github.com/Zillaforge/cloud-sdk/models/vrm/common`):
```go
type Tag struct {
    ID           string                 // Unique tag ID (used for VM creation)
    Name         string                 // Tag name (e.g., "latest", "v1.0.0")
    RepositoryID string                 // Parent repository ID
    Type         TagType                // "common" or "increase"
    Size         int64                  // Image size in bytes
    Status       TagStatus              // e.g., "active", "queued", "error"
    Extra        map[string]interface{} // Additional properties
    CreatedAt    time.Time              // ISO 8601 timestamp
    UpdatedAt    time.Time              // ISO 8601 timestamp
    Repository   *Repository            // Embedded repository info
}

type Repository struct {
    ID              string   // Repository ID
    Name            string   // Repository name
    Namespace       string   // "public" or "private"
    OperatingSystem string   // "linux" or "windows"
    Description     string   // Human-readable description
    Tags            []*Tag   // Associated tags
    Count           int      // Tag count
    Creator         *IDName  // Creator reference
    Project         *IDName  // Project reference
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Alternatives Considered**:
- Use repository list endpoint and extract tags → Accepted for repository-scoped queries: preferred for efficiency (use `Repositories().Get()` / `RepositoryResource.Tags().List()`)
- Use separate Get() calls per tag → Rejected: List() is more efficient for bulk queries

---

### Q2: How to implement filter logic (repository and tag filters)?

**Decision**: Hybrid approach - use SDK `Where` parameter if supported by API, fall back to client-side filtering for exact matches and pattern matching

**Server-Side Filtering** (`ListTagsOptions`):
- `Limit` (int) - pagination limit (max 1000)
- `Offset` (int) - pagination offset
- `Where` ([]string) - filter conditions (syntax to be determined from API docs)
- `Namespace` (string) - namespace filter ("public"/"private")

**Repository-Scoped Listing**:
- Use `vrmClient.Repositories().Get(repositoryID)` or `vrmClient.Repositories().List()` to locate repository and then `RepositoryResource.Tags().List()` to retrieve tags limited to that repository (endpoint: `/api/v1/project/{project-id}/repository/{repository-id}/tags`). This is more efficient for repository-only queries and avoids project-wide tag scans.

**Client-Side Filtering Required**:
- **Exact repository match**: Prefer repository lookup via `vrmClient.Repositories().Get()`/`List()` and use `RepositoryResource.Tags().List()` to fetch repository-scoped tags efficiently. As a fallback, filter `Tag.Repository.Name` from `Tags().List()` if repository lookup is unavailable.
- **Exact tag match**: Filter `Tag.Name` to match user-specified tag
- **Pattern matching**: Implement glob-style wildcard matching (* and ?) for tag names
- **Cross-repository tag queries**: When only tag filter specified, return matching tags across all repositories

**Filter Scenarios**:
1. **Both repository & tag**: Return single image (repository:tag uniquely identifies one Tag)
2. **Repository only**: Return all tags for that repository
3. **Tag only**: Return matching tags across all repositories
4. **No filters**: Return all images (up to server limit)

**Pattern Matching Implementation**:
```go
import "path/filepath"

func matchesPattern(tagName, pattern string) bool {
    matched, _ := filepath.Match(pattern, tagName)
    return matched
}
```

---

### Q3: How are Tag attributes mapped to Terraform schema?

**Decision**: Direct mapping with type conversions and computed attribute extraction

**Attribute Mappings**:

| Terraform Attribute | SDK Source | Type Conversion |
|---------------------|------------|-----------------|
| `id` | `Tag.ID` | string (no conversion) |
| `tag_name` | `Tag.Name` | string (no conversion) |
| `repository_name` | `Tag.Repository.Name` | string (extracted from nested Repository) |
| `size` | `Tag.Size` | int64 → types.Int64Value |
| `operating_system` | `Tag.Repository.OperatingSystem` | string (from Repository) |
| `description` | `Tag.Repository.Description` | string (from Repository) |
| `type` | `Tag.Type` | string (TagType.String()) |
| `status` | `Tag.Status` | string (TagStatus.String()) |

**Optional Attributes** (FR-011):
- `updated_at`: `Tag.UpdatedAt` → RFC3339 string
- `image_format`: `Tag.Extra["container_format"]` or `Tag.Extra["disk_format"]`
- `platform`: `Tag.Extra["architecture"]` or derived from OS



---

### Q4: How to handle sorting and deterministic ordering?

**Decision**: Return results in a deterministic order to avoid non-deterministic Terraform behavior; implement a stable sort by repository name ascending, then tag name ascending.

**Rationale**:
- Avoids exposing internal timestamps and minimizes reliance on fields not part of the data source contract
- Provides deterministic ordering for Terraform plans and state comparisons

**Implementation**:
```go
import "sort"

func sortTagsDeterministic(tags []*tagmodels.Tag) {
    sort.SliceStable(tags, func(i, j int) bool {
        if tags[i].Repository != nil && tags[j].Repository != nil {
            if tags[i].Repository.Name != tags[j].Repository.Name {
                return tags[i].Repository.Name < tags[j].Repository.Name
            }
        }
        return tags[i].Name < tags[j].Name
    })
}
```

---

### Q5: How to handle pagination and server limits?

**Decision**: Use SDK pagination with server-enforced maximum limit

**Server Limit**:
- Maximum 1000 images per query (Clarification Q3)
- SDK `ListTagsOptions.Limit` set to 1000
- `ListTagsResponse.Total` provides total count for awareness

**Pagination Approach**:
- FR-016: Handle pagination transparently
- Single List() call with Limit=1000 for initial implementation
- If Total > 1000, log warning but return first 1000 (sorted)
- Users must use repository/tag filters to narrow results if hitting limit

**Future Enhancement** (not required for v1):
```go
// Paginated retrieval (if needed)
func listAllTags(ctx context.Context, client *tags.Client) ([]*tagmodels.Tag, error) {
    var allTags []*tagmodels.Tag
    offset := 0
    limit := 100
    
    for {
        opts := &tagmodels.ListTagsOptions{
            Limit:  limit,
            Offset: offset,
        }
        resp, err := client.List(ctx, opts)
        if err != nil {
            return nil, err
        }
        
        allTags = append(allTags, resp...)
        
        if len(resp) < limit {
            break  // Last page
        }
        offset += limit
    }
    
    return allTags, nil
}
```

---

### Q6: How to handle API errors and edge cases?

**Decision**: Map SDK errors to Terraform diagnostics with actionable messages

**Error Scenarios**:

| Scenario | SDK Behavior | Terraform Response |
|----------|--------------|-------------------|
| Repository not found | List returns empty array | Return empty list (not error) - only error if explicitly checking repository existence |
| Authentication failed | SDK returns 401 error | AddError diagnostic with authentication guidance |
| Network timeout | SDK context timeout | AddError with retry suggestion |
| Invalid filter | SDK returns 400 error | AddError with filter validation message |
| Server error (5xx) | SDK returns error | AddError with "try again later" message |

**Error Handling Pattern**:
```go
tagList, err := vrmClient.Tags().List(ctx, opts)
if err != nil {
    resp.Diagnostics.AddError(
        "Failed to List Images",
        fmt.Sprintf("Unable to list images: %s\n\n"+
            "Verify provider configuration and network connectivity. "+
            "If the problem persists, check ZillaForge API status.",
            err.Error()),
    )
    return
}
```

**Edge Case: Empty Results**:
- FR-006: Return empty list when no matches found
- Not an error condition
- Allows Terraform plans to complete successfully
- User Story 1, Acceptance Scenario 4: Empty list for pattern with no matches

---

### Q7: How to validate filter mutual exclusivity?

**Decision**: Implement schema-level validation in Read() method

**Pattern from Existing Code** (keypairs, security groups):
```go
// T035 pattern: Validate mutual exclusivity
hasExactTag := !data.Tag.IsNull() && data.Tag.ValueString() != ""
hasPatternTag := !data.TagPattern.IsNull() && data.TagPattern.ValueString() != ""

if hasExactTag && hasPatternTag {
    resp.Diagnostics.AddError(
        "Invalid Filter Combination",
        "Cannot specify both 'tag' and 'tag_pattern' filters. "+
        "Please use only one tag filter at a time.",
    )
    return
}
```

**FR-014**: Tag name and pattern filters are Optional and mutually exclusive  
**FR-018**: Data source MUST validate only one specified

---

### Q8: Best practices for Terraform Plugin Framework data source implementation

**Decision**: Follow established patterns from existing provider data sources

**Implementation Checklist**:

1. **Interface Implementation**:
   ```go
   type ImagesDataSource struct {
       client *cloudsdk.ProjectClient
   }
   
   func (d *ImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
       resp.TypeName = req.ProviderTypeName + "_images"
   }
   ```

2. **Schema Definition**:
   ```go
   func (d *ImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
       resp.Schema = schema.Schema{
           MarkdownDescription: "Queries VM images (repository:tag pairs) from ZillaForge VRM",
           Attributes: map[string]schema.Attribute{
               "repository": schema.StringAttribute{
                   MarkdownDescription: "Filter images by repository name (exact match). Optional.",
                   Optional: true,
               },
               // ... other attributes
           },
       }
   }
   ```

3. **Client Configuration**:
   ```go
   func (d *ImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
       if req.ProviderData == nil {
           return
       }
       
       client, ok := req.ProviderData.(*cloudsdk.ProjectClient)
       if !ok {
           resp.Diagnostics.AddError("Unexpected Data Source Configure Type", ...)
           return
       }
       
       d.client = client
   }
   ```

4. **Read Implementation**:
   ```go
   func (d *ImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
       var data ImagesDataSourceModel
       resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
       
       // Validation
       // API call
       // Filtering
       // Sorting
       // Model mapping
       
       resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
   }
   ```

5. **Model Definition**:
   ```go
   type ImagesDataSourceModel struct {
       Repository  types.String `tfsdk:"repository"`
       Tag         types.String `tfsdk:"tag"`
       TagPattern  types.String `tfsdk:"tag_pattern"`
       Images      types.List   `tfsdk:"images"`  // List of ImageModel
   }
   
   type ImageModel struct {
       ID              types.String `tfsdk:"id"`
       RepositoryName  types.String `tfsdk:"repository_name"`
       TagName         types.String `tfsdk:"tag_name"`

       Size            types.Int64  `tfsdk:"size"`
       OperatingSystem types.String `tfsdk:"operating_system"`
       Description     types.String `tfsdk:"description"`
       Type            types.String `tfsdk:"type"`
       Status          types.String `tfsdk:"status"`
   }
   ```

---

## Technology Decisions Summary

### Chosen Technologies
- **SDK Client**: `github.com/Zillaforge/cloud-sdk/modules/vrm/tags`
- **API Access**: `projectClient.VRM().Tags().List(ctx, opts)`
- **Model Types**: `github.com/Zillaforge/cloud-sdk/models/vrm/common` (Tag, Repository)
- **Filtering**: Hybrid server-side (Where) + client-side (exact match, patterns)
- **Pattern Matching**: `path/filepath.Match()` for glob-style wildcards
- **Sorting**: Deterministic sort (repository name asc, tag name asc)
- **Error Handling**: SDK error wrapping with Terraform diagnostics mapping
- **Testing**: terraform-plugin-testing framework with acceptance tests

### Key Constraints
- Server limit: 1000 images maximum per query
- No regex support: glob wildcards only (* and ?)
- Memory size: SDK returns bytes (no conversion needed)
- Digest information may exist in `Tag.Extra` but the data source does not expose a `digest` attribute (image content may be identified by an immutable cryptographic hash in the VRM service)
- Project-scoped: All queries scoped to authenticated project
- VRM module: Different from VPS module (uses /vrm/api/v1/project path)

### Integration Points
- Provider configuration supplies authenticated SDK client
- VRM client scoped to project (from provider config via `ProjectClient.VRM()`)
- Context propagation from Terraform to SDK for timeout handling
- Error wrapping for diagnostic clarity
- Registration in `provider.go` DataSources() method

### Differences from VPS Data Sources
- **Module**: VRM (not VPS) - uses `projectClient.VRM().Tags()` instead of `projectClient.VPS()`
- **API Path**: `/vrm/api/v1/project/{id}/tags` instead of `/vps/api/v1/project/{id}`
- **Nested Data**: Tag includes embedded Repository information
- **Filter Semantics**: Cross-repository queries supported (tag-only filter)
- **Extra Fields**: Tag.Extra map contains additional metadata (formats, architecture, etc.)

---

## Open Questions for Implementation Phase

1. **Where clause syntax**: How to use `ListTagsOptions.Where` parameter? (Can implement without it initially using client-side filtering only)
2. **Namespace handling**: Should we expose namespace filter? (Deferred: not in FR, can add later)
3. **Pagination behavior**: Does ListTagsResponse include Total field? (Assume yes based on model definition)
4. **Digest field location**: Digest may appear in `Extra["digest"]` or `Extra["checksum"]` in the API. Implementation note: data source will not expose a `digest` attribute; SDK may surface it for internal use.

These can be resolved during implementation with SDK testing or deferred to v2 if not immediately needed for core functionality.
