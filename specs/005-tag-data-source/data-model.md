# Data Model: Image Data Source

**Feature**: zillaforge_images data source  
**Date**: December 15, 2025

## Entity Definitions

### Image

Represents a tagged version of a VM image within a repository (represented as a Tag object in the cloud-sdk). In the ZillaForge VRM API, this corresponds to a `Tag` object that references a specific image version used to create virtual machines.

**Source**: `github.com/Zillaforge/cloud-sdk/models/vrm/common.Tag`

#### Attributes

| Attribute | Type | Source | Required | Computed | Description |
|-----------|------|--------|----------|----------|-------------|
| `id` | string | `Tag.ID` | No | Yes | Unique identifier for the tag (Tag object ID from cloud-sdk). Used to create VMs. Unique per repository:tag pair. |
| `repository_name` | string | `Tag.Repository.Name` | No | Yes | Name of the repository containing this image tag |
| `tag_name` | string | `Tag.Name` | No | Yes | Tag name/label (e.g., "latest", "v1.0.0", "prod-2024") |

| `size` | int64 | `Tag.Size` | No | Yes | Image size in bytes |
| `operating_system` | string | `Tag.Repository.OperatingSystem` | No | Yes | Operating system type ("linux" or "windows") |
| `description` | string | `Tag.Repository.Description` | No | Yes | Human-readable description of the repository/image |
| `type` | string | `Tag.Type.String()` | No | Yes | Tag type ("common" or "increase") |
| `status` | string | `Tag.Status.String()` | No | Yes | Current status (e.g., "active", "queued", "error", "deleted") |

#### Optional Attributes (FR-011)

| Attribute | Type | Source | Description |
|-----------|------|--------|-------------|
| `updated_at` | string | `Tag.UpdatedAt.Format(RFC3339)` | Last update timestamp in ISO 8601 format |
| `image_format` | string | `Tag.Extra["container_format"]` or `Tag.Extra["disk_format"]` | Image format (e.g., "qcow2", "raw", "iso") |
| `platform` | string | `Tag.Extra["architecture"]` | CPU architecture (e.g., "x86_64", "aarch64") |

#### Validation Rules

**From SDK (`Tag.Validate()`):**
- `ID` must not be empty
- `Name` must not be empty
- `RepositoryID` must not be empty
- `Type` must be "common" or "increase"
- `Size` must be >= 0

**Terraform-specific:**
- All attributes are Computed (read-only from API)
- No user-provided values (data source queries existing data)

#### Relationships

- **Belongs to**: One Repository (via `Tag.RepositoryID` and embedded `Tag.Repository`)
- **Identity**: Unique by repository:tag pair (combination of `repository_name` + `tag_name`)
- **Immutability**: Multiple tags may reference the same underlying image content; the data source does not expose a `digest` attribute.

---

### Repository

Represents a virtual image repository within a project. Contains zero or more Tags (image versions).

**Source**: `github.com/Zillaforge/cloud-sdk/models/vrm/common.Repository`

#### Attributes (Embedded in Tag)

| Attribute | Type | Source | Description |
|-----------|------|--------|-------------|
| `ID` | string | `Repository.ID` | Unique repository identifier |
| `Name` | string | `Repository.Name` | Repository name (used for filtering) |
| `Namespace` | string | `Repository.Namespace` | "public" or "private" |
| `OperatingSystem` | string | `Repository.OperatingSystem` | "linux" or "windows" |
| `Description` | string | `Repository.Description` | Human-readable description |
| `Count` | int | `Repository.Count` | Number of tags in repository |

**Note**: The `Repository` is queryable via the VRM Repositories endpoint. For efficiency the data source SHOULD use `vrmClient.Repositories().Get()` or `vrmClient.Repositories().List()` to find repository metadata (and then `RepositoryResource.Tags().List()` to retrieve tags) when a `repository` filter is specified, instead of scanning all tags via `Tags().List()`.

---

## Data Source Schema

### Input Attributes (Filters)

| Attribute | Type | Required | Description | Validation |
|-----------|------|----------|-------------|------------|
| `repository` | string | Optional | Filter images by exact repository name | None |
| `tag` | string | Optional | Filter images by exact tag name | Mutually exclusive with `tag_pattern` |
| `tag_pattern` | string | Optional | Filter images by tag name pattern (glob-style: `*`, `?`) | Mutually exclusive with `tag`. Must use glob syntax only (no regex). |

**Filter Behavior** (per spec clarifications):

1. **Both `repository` and `tag`** → Returns list of length 1 (unique repository:tag)
2. **`repository` only** → Returns all tags for that repository
3. **`tag` only** → Returns matching tags across all repositories
4. **`tag_pattern` only** → Returns tags matching pattern across all repositories
5. **No filters** → Returns all images up to server limit (1000 max)

### Output Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `images` | List of Image | List of image objects matching filter criteria. Empty list if no matches. |

**List Behavior**:
- Always returns a list (even when filters uniquely identify one image)
- Empty list is not an error (FR-006)
- Deterministic ordering: sorted by `repository_name` ascending, then `tag_name` ascending (FR-015 updated)

---

## Terraform HCL Examples

### Example 1: Query specific image by repository and tag

```hcl
data "zillaforge_images" "ubuntu_2204" {
  repository = "ubuntu"
  tag        = "22.04"
}

# Access unique image
resource "zillaforge_vm" "web_server" {
  image_id = data.zillaforge_images.ubuntu_2204.images[0].id
  # ...
}
```

### Example 2: List all tags for a repository

```hcl
data "zillaforge_images" "ubuntu_all" {
  repository = "ubuntu"
}

# Find latest by sorted order (newest first)
locals {
  latest_ubuntu = data.zillaforge_images.ubuntu_all.images[0]
}
```

### Example 3: Pattern matching for versioned tags

```hcl
data "zillaforge_images" "prod_images" {
  tag_pattern = "prod-*"
}

output "production_images" {
  value = [
    for img in data.zillaforge_images.prod_images.images : {
      repo = img.repository_name
      tag  = img.tag_name
      id   = img.id
    }
  ]
}
```

### Example 4: Cross-repository tag search

```hcl
data "zillaforge_images" "latest_tags" {
  tag = "latest"
}

# Returns "latest" tag from multiple repositories
output "available_latest" {
  value = [for img in data.zillaforge_images.latest_tags.images : img.repository_name]
}
```

### Example 5: List all images (no filters)

```hcl
data "zillaforge_images" "all" {
  # No filters - returns up to 1000 images
}

output "total_images" {
  value = length(data.zillaforge_images.all.images)
}
```

---

## Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│ Terraform Configuration                                      │
│                                                              │
│  data "zillaforge_images" "example" {                        │
│    repository  = "ubuntu"          # Optional filter        │
│    tag_pattern = "22.*"            # Optional filter        │
│  }                                                           │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ 1. Read() called with filters
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ ImagesDataSource                                             │
│                                                              │
│  • Validate filters (mutual exclusivity)                    │
│  • Build ListTagsOptions                                    │
│  • If `repository` filter specified: use `vrmClient.Repositories().Get/List()` and `RepositoryResource.Tags().List()` for repository-scoped tags; otherwise call `vrmClient.Tags().List(ctx, opts)`                    │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ 2. GET /vrm/api/v1/project/{id}/tags
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ ZillaForge VRM API                                           │
│                                                              │
│  • Query database for tags matching project                 │
│  • Apply server-side filters (limit, offset, where)         │
│  • Return ListTagsResponse with Tag array                   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ 3. []*common.Tag (with embedded Repository)
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Data Source Processing                                       │
│                                                              │
│  • Client-side filter by repository name (exact)            │
│  • Client-side filter by tag name (exact or pattern)        │
│  • Deterministic sort by repository_name asc, tag_name asc    │
│  • Map Tag → ImageModel (extract Repository fields)         │
│  • Build types.List with ImageModel objects                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ 4. Set state with images list
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Terraform State                                              │
│                                                              │
│  images = [                                                  │
│    {                                                         │
│      id                = "tag-uuid-1"                        │
│      repository_name   = "ubuntu"                            │
│      tag_name          = "22.04"                             │
│      # digest not exposed by data source                    │
│      size              = 2147483648                          │
│      operating_system  = "linux"                             │
│      description       = "Ubuntu 22.04 LTS"                  │
│      type              = "common"                            │
│      status            = "active"                            │
│    }                                                         │
│  ]                                                           │
└─────────────────────────────────────────────────────────────┘
```

---

## State Transitions

Images data source is read-only and does not manage state transitions. However, Tag `status` field reflects image lifecycle:

```
queued → saving → creating → active
                           ↓
                        deleted
                        
    OR error path:
creating → error
         → killed
         → error_deleting
```

**Relevant Statuses for Data Source**:
- `active`: Image is ready for VM creation ✅
- `available`: Image is available ✅
- `queued`, `saving`, `creating`: Image is being prepared ⏳
- `error`, `killed`, `deleted`: Image is not usable ❌

**Recommendation**: Data source returns all statuses; users can filter in Terraform:
```hcl
locals {
  active_images = [
    for img in data.zillaforge_images.all.images :
    img if img.status == "active" || img.status == "available"
  ]
}
```

---

## Type Conversions

### SDK → Terraform

| SDK Type | Terraform Type | Conversion |
|----------|----------------|------------|
| `string` | `types.String` | `types.StringValue(sdkValue)` |
| `int64` | `types.Int64` | `types.Int64Value(sdkValue)` |
| `time.Time` | `types.String` | `types.StringValue(t.Format(time.RFC3339))` |
| `TagType` (enum) | `types.String` | `types.StringValue(tagType.String())` |
| `TagStatus` (enum) | `types.String` | `types.StringValue(status.String())` |
| `map[string]interface{}` | `types.String` | Extract specific key, type assert to string |
| `*Repository` (nested) | Flattened attributes | Extract fields individually |

### Null Handling

```go
// Handle empty description
var description types.String
if tag.Repository != nil && tag.Repository.Description != "" {
    description = types.StringValue(tag.Repository.Description)
} else {
    description = types.StringNull()
}
```

---

## Error Conditions

### Validation Errors (Before API Call)

| Condition | Error Message | Severity |
|-----------|---------------|----------|
| Both `tag` and `tag_pattern` specified | "Cannot specify both 'tag' and 'tag_pattern' filters. Please use only one tag filter at a time." | Error |

### API Errors (During List Call)

| SDK Error | Terraform Diagnostic |
|-----------|---------------------|
| Context timeout | "Failed to List Images: context deadline exceeded. Verify network connectivity and try again." |
| 401 Unauthorized | "Failed to List Images: authentication failed. Verify ZILLAFORGE_API_TOKEN environment variable or provider credentials configuration." |
| 404 Not Found | "Failed to List Images: project not found. Verify project ID in provider configuration." |
| 500 Server Error | "Failed to List Images: server error. The ZillaForge API is experiencing issues. Try again later or contact support." |

### Empty Results

Not an error - returns empty list:
```hcl
data "zillaforge_images" "nonexistent" {
  repository = "does-not-exist"
}

# data.zillaforge_images.nonexistent.images == []
# length(data.zillaforge_images.nonexistent.images) == 0
```

---

## Performance Considerations

### Query Optimization

- **Single API call**: One `List()` call regardless of filters
- **Client-side filtering**: Minimal overhead for exact string matching
- **Pattern matching**: `filepath.Match()` is O(n×m) where n=tag length, m=pattern length
- **Sorting**: O(n log n) where n=number of tags returned

### Memory Usage

- **Maximum result set**: 1000 images × ~500 bytes/image ≈ 500 KB
- **Embedded Repository**: Each Tag includes full Repository object (~200 bytes)
- **Extra map**: Variable size, typically <100 bytes per tag

### Scaling Limits

Per SC-008, must handle up to 1000 images without degradation:
- API call: <3 seconds (SC-001)
- Client-side processing: <100ms for 1000 items
- Total Read() time: <3.5 seconds worst case

---

## Testing Strategy

### Unit Tests

```go
func TestFilterByRepository(t *testing.T)  // Exact repository name match
func TestFilterByTag(t *testing.T)        // Exact tag name match
func TestFilterByPattern(t *testing.T)     // Glob pattern matching
func TestMutualExclusivity(t *testing.T)   // tag vs tag_pattern validation
func TestSortOrder(t *testing.T)           // Deterministic: repository_name asc, tag_name asc
// Digest not exposed: no dedicated digest extraction test required
```

### Acceptance Tests

```go
func TestAccImagesDataSource_RepositoryAndTag(t *testing.T)  // Both filters → length 1
func TestAccImagesDataSource_RepositoryOnly(t *testing.T)    // List all tags in repo
func TestAccImagesDataSource_TagOnly(t *testing.T)           // Cross-repo tag search
func TestAccImagesDataSource_Pattern(t *testing.T)           // Wildcard matching
func TestAccImagesDataSource_NoFilters(t *testing.T)         // List all (up to limit)
func TestAccImagesDataSource_EmptyResults(t *testing.T)      // No matches returns []
func TestAccImagesDataSource_VMCreation(t *testing.T)        // Use image.id in VM resource
```
