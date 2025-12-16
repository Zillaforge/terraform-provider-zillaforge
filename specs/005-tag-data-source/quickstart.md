# Quickstart Guide: Images Data Source

**Feature**: zillaforge_images data source  
**For**: Developers implementing the data source  
**Date**: December 15, 2025

## Prerequisites

- Go 1.22.4+
- Terraform CLI 1.0+
- ZillaForge account with project access
- `github.com/Zillaforge/cloud-sdk` v0.0.0-20251209081935-79e26e215136

## Common cloud-sdk Operations

### List Tags (Images)

```go
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    tagmodels "github.com/Zillaforge/cloud-sdk/models/vrm/tags"
    repmod "github.com/Zillaforge/cloud-sdk/models/vrm/repositories"
    "github.com/Zillaforge/cloud-sdk/models/vrm/common"
)

// Access VRM Tags client
vrmClient := projectClient.VRM()
tagsClient := vrmClient.Tags()

// List all tags (project-wide)
opts := &tagmodels.ListTagsOptions{
    Limit:  1000,
    Offset: 0,
}
tags, err := tagsClient.List(ctx, opts)
if err != nil {
    return fmt.Errorf("failed to list tags: %w", err)
}

// tags is []*common.Tag
for _, tag := range tags {
    fmt.Printf("Tag: %s:%s (ID: %s)\n", tag.Repository.Name, tag.Name, tag.ID)
}

// --- Repository-scoped (more efficient) ---
// Find repository by name and list only its tags
repos, err := vrmClient.Repositories().List(ctx, &repmod.ListRepositoriesOptions{Limit:1000})
if err != nil {
    return fmt.Errorf("failed to list repositories: %w", err)
}
var repoID string
for _, r := range repos {
    if r.Repository.Name == "ubuntu" {
        repoID = r.Repository.ID
        break
    }
}
if repoID != "" {
    repoRes, err := vrmClient.Repositories().Get(ctx, repoID)
    if err != nil {
        return fmt.Errorf("failed to get repository %s: %w", repoID, err)
    }
    repoTags, err := repoRes.Tags().List(ctx, &tagmodels.ListTagsOptions{Limit:1000})
    if err != nil {
        return fmt.Errorf("failed to list tags for repository %s: %w", repoID, err)
    }
    for _, tag := range repoTags {
        fmt.Printf("Repo Tag: %s:%s (ID: %s)\n", tag.Repository.Name, tag.Name, tag.ID)
    }
}
```

### Get Single Tag

```go
tag, err := tagsClient.Get(ctx, "tag-uuid")
if err != nil {
    return fmt.Errorf("failed to get tag: %w", err)
}

fmt.Printf("Repository: %s, Tag: %s, Size: %d bytes\n",
    tag.Repository.Name, tag.Name, tag.Size)
```

### Extract Image Attributes

```go
func tagToImageModel(tag *common.Tag) ImageModel {
    return ImageModel{
        ID:              types.StringValue(tag.ID),
        RepositoryName:  types.StringValue(tag.Repository.Name),
        TagName:         types.StringValue(tag.Name),
        Size:            types.Int64Value(tag.Size),
        OperatingSystem: types.StringValue(tag.Repository.OperatingSystem),
        Description:     types.StringValue(tag.Repository.Description),
        Type:            types.StringValue(tag.Type.String()),
        Status:          types.StringValue(tag.Status.String()),
    }
}
```

## Implementation Steps (TDD Workflow)

### Step 1: Write Acceptance Tests (RED)

Create `internal/vrm/data/images_data_source_test.go`:

```go
func TestAccImagesDataSource_RepositoryAndTag(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { provider.TestAccPreCheck(t) },
        ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccImagesDataSourceConfig_repositoryAndTag,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("data.zillaforge_images.test", "images.#", "1"),
                    resource.TestCheckResourceAttr("data.zillaforge_images.test", "images.0.repository_name", "ubuntu"),
                    resource.TestCheckResourceAttr("data.zillaforge_images.test", "images.0.tag_name", "22.04"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_images.test", "images.0.id"),
                ),
            },
        },
    })
}

const testAccImagesDataSourceConfig_repositoryAndTag = `
data "zillaforge_images" "test" {
  repository = "ubuntu"
  tag        = "22.04"
}
`
```

Run test → **FAIL** (data source not implemented)

### Step 2: Implement Data Source (GREEN)

Create `internal/vrm/data/images_data_source.go`:

```go
package data

import (
    "context"
    "fmt"
    "path/filepath"
    "sort"
    
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    tagmodels "github.com/Zillaforge/cloud-sdk/models/vrm/tags"
    "github.com/Zillaforge/cloud-sdk/models/vrm/common"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

type ImagesDataSource struct {
    client *cloudsdk.ProjectClient
}

func NewImagesDataSource() datasource.DataSource {
    return &ImagesDataSource{}
}

func (d *ImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_images"
}

func (d *ImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    // See contracts/images-data-source-schema.md for full schema
}

func (d *ImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    
    client, ok := req.ProviderData.(*cloudsdk.ProjectClient)
    if !ok {
        resp.Diagnostics.AddError("Unexpected Data Source Configure Type", "...")
        return
    }
    
    d.client = client
}

func (d *ImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data ImagesDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }
    
    // 1. Validate mutual exclusivity
    hasExactTag := !data.Tag.IsNull() && data.Tag.ValueString() != ""
    hasPatternTag := !data.TagPattern.IsNull() && data.TagPattern.ValueString() != ""
    if hasExactTag && hasPatternTag {
        resp.Diagnostics.AddError("Invalid Filter Combination",
            "Cannot specify both 'tag' and 'tag_pattern' filters. Please use only one tag filter at a time.")
        return
    }
    
    // 2. Call API (prefer repository-scoped listing when repository filter specified)
    vrmClient := d.client.VRM()
    var tags []*common.Tag
    if data.Repository.ValueString() != "" {
        // Efficient path: look up repository by name and list tags for that repository
        repoTags, err := listRepoTags(ctx, d.client, data.Repository.ValueString())
        if err != nil {
            resp.Diagnostics.AddError("Failed to List Images", fmt.Sprintf("Unable to list images for repository %s: %s", data.Repository.ValueString(), err.Error()))
            return
        }
        tags = repoTags
    } else {
        opts := &tagmodels.ListTagsOptions{Limit: 1000, Offset: 0}
        allTags, err := vrmClient.Tags().List(ctx, opts)
        if err != nil {
            resp.Diagnostics.AddError("Failed to List Images",
                fmt.Sprintf("Unable to list images: %s", err.Error()))
            return
        }
        tags = allTags
    }

    // 3. Filter (if needed for tag or tag_pattern when using repo-scoped tags)
    filtered := filterTags(tags, data.Repository.ValueString(), data.Tag.ValueString(), data.TagPattern.ValueString())
    
    // 4. Sort
    sortTags(filtered)
    
    // 5. Convert to models
    images := make([]ImageModel, len(filtered))
    for i, tag := range filtered {
        images[i] = tagToImageModel(tag)
    }
    
    // 6. Set state
    imagesList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: imageAttrTypes}, images)
    resp.Diagnostics.Append(diags...)
    data.Images = imagesList
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func filterTags(tags []*common.Tag, repo, tag, pattern string) []*common.Tag {
    var filtered []*common.Tag
    for _, t := range tags {
        // Repository filter
        if repo != "" && t.Repository.Name != repo {
            continue
        }
        // Exact tag filter
        if tag != "" && t.Name != tag {
            continue
        }
        // Pattern filter
        if pattern != "" {
            matched, _ := filepath.Match(pattern, t.Name)
            if !matched {
                continue
            }
        }
        filtered = append(filtered, t)
    }
    return filtered
}

func sortTags(tags []*common.Tag) {
    sort.Slice(tags, func(i, j int) bool {
        // Deterministic sort by repository name then tag name
        if tags[i].Repository != nil && tags[j].Repository != nil {
            if tags[i].Repository.Name != tags[j].Repository.Name {
                return tags[i].Repository.Name < tags[j].Repository.Name
            }
        }
        return tags[i].Name < tags[j].Name
    })
}
```

Run test → **PASS**

### Step 3: Register Data Source

In `internal/provider/provider.go`:

```go
func (p *ZillaforgeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        data.NewFlavorsDataSource,
        data.NewNetworkDataSource,
        data.NewKeypairDataSource,
        data.NewSecurityGroupsDataSource,
        data.NewImagesDataSource,  // <-- Add this line
    }
}
```

### Step 4: Refactor (Code Quality)

- Extract helper functions
- Add comments
- Add logging with `tflog.Debug()`
- Add edge case handling

## Usage Examples

### Example 1: Find Ubuntu 22.04 Image for VM

```hcl
data "zillaforge_images" "ubuntu" {
  repository = "ubuntu"
  tag        = "22.04"
}

resource "zillaforge_vm" "web" {
  name     = "web-server"
  image_id = data.zillaforge_images.ubuntu.images[0].id
  flavor_id = "flavor-uuid"
  # ...
}
```

### Example 2: List Production Images

```hcl
data "zillaforge_images" "prod" {
  tag_pattern = "prod-*"
}

output "production_images" {
  value = {
    for img in data.zillaforge_images.prod.images :
    "${img.repository_name}:${img.tag_name}" => img.id
  }
}
```

### Example 3: Select Latest Ubuntu

```hcl
data "zillaforge_images" "ubuntu_all" {
  repository = "ubuntu"
}

locals {
  latest_ubuntu_id = data.zillaforge_images.ubuntu_all.images[0].id
}
```

## Debugging Tips

### Enable Debug Logging

```bash
export TF_LOG=DEBUG
terraform plan
```

### Check API Calls

```go
import "github.com/hashicorp/terraform-plugin-log/tflog"

func (d *ImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    tflog.Debug(ctx, "Listing images", map[string]interface{}{
        "repository": data.Repository.ValueString(),
        "tag": data.Tag.ValueString(),
    })
    
    tags, err := vrmClient.Tags().List(ctx, opts)
    tflog.Debug(ctx, "API response", map[string]interface{}{
        "tag_count": len(tags),
    })
}
```

### Test Individual Filters

```bash
# Test repository filter
terraform plan -var="test_repository=ubuntu"

# Test pattern
terraform plan -var="test_pattern=v1.*"
```

## Common Errors

### Error: Mutual Exclusivity

**Cause**: Both `tag` and `tag_pattern` specified  
**Fix**: Use only one tag filter

### Error: Failed to List Images

**Cause**: Authentication, network, or API error  
**Fix**: Check provider configuration, credentials, network connectivity

### Empty Results

**Cause**: No images match filters (not an error)  
**Fix**: Verify repository/tag names are correct

## Next Steps

1. Implement remaining acceptance tests (pattern matching, cross-repo, empty results)
2. Add unit tests for filter and sort logic
3. Generate documentation with `tfplugindocs`
4. Add examples to `examples/data-sources/zillaforge_images/`
5. Update CHANGELOG.md
