# Contract: zillaforge_images Data Source Schema

**Feature**: zillaforge_images data source  
**Version**: 1.0  
**Date**: December 15, 2025

## Schema Definition

### Terraform Plugin Framework Schema

```go
package data

import (
	"context"
	
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *ImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries VM images (represented as repository:tag pairs) from the ZillaForge VRM service. " +
			"Images can be filtered by repository name, exact tag name, or tag name pattern. " +
			"Each image includes metadata such as size, operating system, status, and a unique ID used for VM creation.",
		
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				MarkdownDescription: "Filter images by exact repository name (case-sensitive). " +
					"When combined with `tag`, returns a single image. " +
					"When used alone, returns all tags for the specified repository. " +
					"Optional - omit to query across all repositories.",
				Optional: true,
			},
			
			"tag": schema.StringAttribute{
				MarkdownDescription: "Filter images by exact tag name (case-sensitive). " +
					"When combined with `repository`, returns a single image. " +
					"When used alone, returns matching tags across all repositories. " +
					"Mutually exclusive with `tag_pattern`. " +
					"Optional - omit to list all tags (up to server limit).",
				Optional: true,
			},
			
			"tag_pattern": schema.StringAttribute{
				MarkdownDescription: "Filter images by tag name pattern using glob-style wildcards (`*` matches any characters, `?` matches single character). " +
					"Examples: `v1.*` matches all v1.x tags, `prod-*` matches tags starting with 'prod-'. " +
					"Mutually exclusive with `tag`. " +
					"Optional - omit for exact matching or no tag filtering.",
				Optional: true,
			},
			
			"images": schema.ListNestedAttribute{
				MarkdownDescription: "List of images matching the filter criteria. " +
					"Returns an empty list if no images match. " +
					"Sorted by creation time (newest first), then alphabetically by tag name. " +
					"Maximum 1000 images returned (server-enforced limit).",
				Computed: true,
				
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the image tag (UUID). " +
								"This is the Tag object ID from the cloud-sdk, used to create virtual machines. " +
								"Each repository:tag pair has a unique ID.",
							Computed: true,
						},
						
						"repository_name": schema.StringAttribute{
							MarkdownDescription: "Name of the repository containing this image tag. " +
								"Repository names are unique within a project.",
							Computed: true,
						},
						
						"tag_name": schema.StringAttribute{
							MarkdownDescription: "Tag name or label for this image version. " +
								"Examples: `latest`, `v1.0.0`, `prod-2024`, `ubuntu-22.04`. " +
								"Tag names follow image repository naming conventions.",
							Computed: true,
						},
						
						
						"size": schema.Int64Attribute{
							MarkdownDescription: "Image size in bytes. " +
								"Includes manifest and layer sizes as reported by the VRM API.",
							Computed: true,
						},
						
						"operating_system": schema.StringAttribute{
							MarkdownDescription: "Operating system type for this image. " +
								"Valid values: `linux`, `windows`.",
							Computed: true,
						},
						
						"description": schema.StringAttribute{
							MarkdownDescription: "Human-readable description of the image repository. " +
								"May be empty if no description is provided.",
							Computed: true,
						},
						
						"type": schema.StringAttribute{
							MarkdownDescription: "Tag type classification. " +
								"Valid values: `common` (standard image), `increase` (incremental image).",
							Computed: true,
						},
						
						"status": schema.StringAttribute{
							MarkdownDescription: "Current status of the image tag. " +
								"Common values: `active` (ready for use), `queued`, `saving`, `creating`, `error`, `deleted`. " +
								"Only `active` and `available` images are typically usable for VM creation.",
							Computed: true,
						},
					},
				},
			},
		},
	}
}
```

---

## Data Models

### ImagesDataSourceModel

```go
type ImagesDataSourceModel struct {
	Repository  types.String `tfsdk:"repository"`
	Tag         types.String `tfsdk:"tag"`
	TagPattern  types.String `tfsdk:"tag_pattern"`
	Images      types.List   `tfsdk:"images"`  // Element type: ImageModel
}
```

### ImageModel

```go
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

## Filter Behavior Specification

### Scenario 1: Both `repository` and `tag` Specified

**Input**:
```hcl
data "zillaforge_images" "specific" {
  repository = "ubuntu"
  tag        = "22.04"
}
```

**API Call**: `GET /vrm/api/v1/project/{project-id}/tags`

**Client-Side Logic**:
1. Prefer repository lookup via `vrmClient.Repositories().Get()` or `List()` and then `RepositoryResource.Tags().List()` for efficiency; fallback to filtering `Tag.Repository.Name == "ubuntu"` if repository lookup is not used
2. Filter `Tag.Name == "22.04"`

**Expected Output**:
```hcl
images = [
  {
    id                = "550e8400-e29b-41d4-a716-446655440000"
    repository_name   = "ubuntu"
    tag_name          = "22.04"
    size              = 2147483648
    operating_system  = "linux"
    description       = "Ubuntu 22.04 LTS"
    type              = "common"
    status            = "active"
  }
]
```

**Result**: List with exactly 1 element (or empty if not found)

---

### Scenario 2: Only `repository` Specified

**Input**:
```hcl
data "zillaforge_images" "ubuntu_all" {
  repository = "ubuntu"
}
```

**Client-Side Logic**:
1. Prefer repository lookup via `vrmClient.Repositories().Get()` or `List()` and then `RepositoryResource.Tags().List()` for efficiency; fallback to filtering `Tag.Repository.Name == "ubuntu"` if repository lookup is not used
2. Sort deterministically by repository_name asc, tag_name asc

**Expected Output**:
```hcl
images = [
  {
    id              = "uuid-1"
    repository_name = "ubuntu"
    tag_name        = "24.04"
    ...
  },
  {
    id              = "uuid-2"
    repository_name = "ubuntu"
    tag_name        = "22.04"
    ...
  },
  {
    id              = "uuid-3"
    repository_name = "ubuntu"
    tag_name        = "20.04"
    ...
  }
]
```

**Result**: All tags for "ubuntu" repository, sorted deterministically by repository name then tag name

---

### Scenario 3: Only `tag` Specified (Cross-Repository Search)

**Input**:
```hcl
data "zillaforge_images" "latest" {
  tag = "latest"
}
```

**Client-Side Logic**:
1. Filter `Tag.Name == "latest"`
2. Deterministic order: sort by repository name asc, tag name asc (repository name primary, tag name secondary)

**Expected Output**:
```hcl
images = [
  {
    id              = "uuid-1"
    repository_name = "nginx"
    tag_name        = "latest"
    ...
  },
  {
    id              = "uuid-2"
    repository_name = "ubuntu"
    tag_name        = "latest"
    ...
  },
  {
    id              = "uuid-3"
    repository_name = "postgres"
    tag_name        = "latest"
    ...
  }
]
```

**Result**: All "latest" tags across different repositories

---

### Scenario 4: `tag_pattern` with Wildcard

**Input**:
```hcl
data "zillaforge_images" "v1_series" {
  repository  = "myapp"
  tag_pattern = "v1.*"
}
```

**Client-Side Logic**:
1. Prefer repository lookup via `vrmClient.Repositories().Get()`/`List()` and then `RepositoryResource.Tags().List()` for efficiency; fallback to filtering `Tag.Repository.Name == "myapp"` if repository lookup is not used
2. Pattern match `filepath.Match("v1.*", Tag.Name)`
3. Deterministic order: sort by repository name asc, tag name asc

**Expected Output**:
```hcl
images = [
  {
    id              = "uuid-1"
    repository_name = "myapp"
    tag_name        = "v1.2.3"
    ...
  },
  {
    id              = "uuid-2"
    repository_name = "myapp"
    tag_name        = "v1.2.2"
    ...
  },
  {
    id              = "uuid-3"
    repository_name = "myapp"
    tag_name        = "v1.0.0"
    ...
  }
]
```

**Pattern Matching Rules**:
- `*` matches zero or more characters
- `?` matches exactly one character
- Case-sensitive matching
- Uses Go `path/filepath.Match()` function

---

### Scenario 5: No Filters (List All)

**Input**:
```hcl
data "zillaforge_images" "all" {
  # No filters specified
}
```

**Client-Side Logic**:
1. No filtering (return all tags from API)
2. Sort by `CreatedAt` desc, `Name` asc
3. Return up to server limit (1000 images)

**Expected Output**: All available images in project (truncated at 1000)

---

### Scenario 6: Empty Results

**Input**:
```hcl
data "zillaforge_images" "nonexistent" {
  repository = "does-not-exist"
}
```

**Expected Output**:
```hcl
images = []
```

**Behavior**: Returns empty list, NOT an error (FR-006)

---

### Scenario 7: Mutual Exclusivity Violation

**Input**:
```hcl
data "zillaforge_images" "invalid" {
  tag         = "latest"
  tag_pattern = "v1.*"
}
```

**Expected Error**:
```
Error: Invalid Filter Combination

Cannot specify both 'tag' and 'tag_pattern' filters.
Please use only one tag filter at a time.
```

**Severity**: Error (blocks Terraform plan)

---

## API Integration

### cloud-sdk Method Calls

```go
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    tagmodels "github.com/Zillaforge/cloud-sdk/models/vrm/tags"
    "github.com/Zillaforge/cloud-sdk/models/vrm/common"
)

// List all tags with pagination
func listImages(ctx context.Context, client *cloudsdk.ProjectClient) ([]*common.Tag, error) {
    vrmClient := client.VRM()
    
    opts := &tagmodels.ListTagsOptions{
        Limit:  1000,  // Server maximum
        Offset: 0,
    }
    
    tags, err := vrmClient.Tags().List(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to list tags: %w", err)
    }
    
    return tags, nil
}

// Efficiently list tags for a single repository by name
func listRepoTags(ctx context.Context, client *cloudsdk.ProjectClient, repoName string) ([]*tagmodels.Tag, error) {
    vrmClient := client.VRM()
    // List repositories and search by name (could also use server-side where filter if available)
    repos, err := vrmClient.Repositories().List(ctx, &repmod.ListRepositoriesOptions{Limit:1000})
    if err != nil {
        return nil, fmt.Errorf("failed to list repositories: %w", err)
    }
    var repoID string
    for _, r := range repos {
        if r.Repository.Name == repoName {
            repoID = r.Repository.ID
            break
        }
    }
    if repoID == "" {
        // Not found - return empty list (no error)
        return []*tagmodels.Tag{}, nil
    }
    // Use repository-scoped tags client for efficient per-repository listing
    repoRes, err := vrmClient.Repositories().Get(ctx, repoID)
    if err != nil {
        return nil, fmt.Errorf("failed to get repository %s: %w", repoID, err)
    }
    tags, err := repoRes.Tags().List(ctx, &tagmodels.ListTagsOptions{Limit:1000})
    if err != nil {
        return nil, fmt.Errorf("failed to list tags for repository %s: %w", repoName, err)
    }
    return tags, nil
}
```

### Request/Response Examples

**Request**:
```
GET /vrm/api/v1/project/550e8400-e29b-41d4-a716-446655440000/tags?limit=1000&offset=0
Authorization: Bearer <token>
```

**Response** (simplified):
```json
{
  "tags": [
    {
      "id": "tag-uuid-1",
      "name": "22.04",
      "repositoryID": "repo-uuid-1",
      "type": "common",
      "size": 2147483648,
      "status": "active",
      "extra": {
        "container_format": "bare",
        "disk_format": "qcow2"
      },
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z",
      "repository": {
        "id": "repo-uuid-1",
        "name": "ubuntu",
        "namespace": "public",
        "operatingSystem": "linux",
        "description": "Ubuntu Images",
        "count": 5
      }
    }
  ],
  "total": 47
}
```

---

## Validation Rules

### Schema-Level Validation

1. **Mutual Exclusivity**: `tag` and `tag_pattern` cannot both be specified
   - Checked in `Read()` method before API call
   - Returns diagnostic error if violated

### API-Level Validation

1. **Authentication**: Valid project-scoped credentials required
   - 401 error if token invalid/expired
2. **Project Access**: User must have read access to project images
   - 403 error if insufficient permissions
3. **Pagination Limits**: Server enforces maximum 1000 results
   - Additional results require filtering or not returned

---

## Performance Characteristics

### Expected Performance

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Query specific image (repository + tag) | <2s | SC-002 |
| List all tags in repository | <3s | SC-001 |
| Pattern matching | <3s | SC-003 |
| List all images (no filters) | <3s | SC-001 |

### Scaling Behavior

- **Up to 100 images**: Negligible client-side processing (<10ms)
- **100-500 images**: Client filtering + sorting <50ms
- **500-1000 images**: Client processing <100ms (acceptable)
- **>1000 images**: Server limit prevents; users must filter

---

## Error Handling

### Validation Errors

```go
// Mutual exclusivity check
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

### API Errors

```go
tags, err := vrmClient.Tags().List(ctx, opts)
if err != nil {
    resp.Diagnostics.AddError(
        "Failed to List Images",
        fmt.Sprintf("Unable to list images from ZillaForge VRM API: %s\n\n"+
            "Verify provider configuration (project_id, credentials) and network connectivity. "+
            "If the problem persists, check ZillaForge API status.",
            err.Error()),
    )
    return
}
```

### Empty Results (Not an Error)

```go
// No diagnostic error for empty results
if len(filteredTags) == 0 {
    data.Images, diags = types.ListValueFrom(ctx, imageObjectType, []ImageModel{})
    resp.Diagnostics.Append(diags...)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
    return
}
```

---

## Testing Contract

### Acceptance Test Requirements

```go
// T001: Repository + Tag filter returns length-1 list
func TestAccImagesDataSource_RepositoryAndTag(t *testing.T) {
    // Given: Repository "ubuntu" with tag "22.04" exists
    // When: Query with both filters
    // Then: images list length == 1
    // And: images[0].repository_name == "ubuntu"
    // And: images[0].tag_name == "22.04"
}

// T002: Repository-only filter returns multiple tags
func TestAccImagesDataSource_RepositoryOnly(t *testing.T) {
    // Given: Repository "ubuntu" with multiple tags
    // When: Query with repository filter only
    // Then: images list length > 1
    // And: All images[].repository_name == "ubuntu"
    // And: Results sorted by newest first
}

// T003: Tag-only filter returns cross-repository results
func TestAccImagesDataSource_TagOnly(t *testing.T) {
    // Given: Multiple repositories with "latest" tag
    // When: Query with tag="latest" only
    // Then: images list contains multiple repositories
    // And: All images[].tag_name == "latest"
}

// T004: Pattern matching with wildcards
func TestAccImagesDataSource_Pattern(t *testing.T) {
    // Given: Repository with tags v1.0.0, v1.1.0, v2.0.0
    // When: Query with tag_pattern="v1.*"
    // Then: images list length == 2
    // And: All matched tags start with "v1."
}

// T005: No filters returns all images
func TestAccImagesDataSource_NoFilters(t *testing.T) {
    // Given: Multiple images exist in project
    // When: Query with no filters
    // Then: images list length > 0
    // And: images list length <= 1000 (server limit)
}

// T006: Empty results for nonexistent repository
func TestAccImagesDataSource_EmptyResults(t *testing.T) {
    // Given: Repository "nonexistent" does not exist
    // When: Query with repository="nonexistent"
    // Then: images list length == 0
    // And: No error diagnostics
}

// T007: Mutual exclusivity validation
func TestAccImagesDataSource_MutualExclusivityError(t *testing.T) {
    // Given: Both tag and tag_pattern specified in config
    // When: Terraform plan executed
    // Then: Plan fails with validation error
    // And: Error message mentions mutual exclusivity
}

// T008: Use image ID in VM resource
func TestAccImagesDataSource_VMCreation(t *testing.T) {
    // Given: Image queried via data source
    // When: data.zillaforge_images.example.images[0].id used in VM resource
    // Then: VM created successfully with correct image
}
```

---

## Migration Path

Not applicable - this is a net-new data source with no prior implementation.

---

## Deprecation Policy

Not applicable - initial implementation.

Future changes will follow semantic versioning:
- **MINOR**: New optional attributes (backward compatible)
- **MAJOR**: Breaking changes to filter behavior or attribute removal
