# Quickstart Guide: Keypair Data Source and Resource

**Feature**: 003-keypair-data-resource  
**For**: Developers implementing this feature  
**Date**: December 13, 2025

## Overview

This guide provides a quick reference for implementing the keypair data source and resource in the Terraform Provider Zillaforge. Follow the TDD workflow and use existing patterns from flavor/network implementations.

## Implementation Checklist

- [ ] Phase 1: Data Source Implementation
  - [ ] Write acceptance tests (failing)
  - [ ] Implement schema
  - [ ] Implement Read() method
  - [ ] Verify tests pass
  - [ ] Add examples
- [ ] Phase 2: Resource Implementation  
  - [ ] Write acceptance tests (failing)
  - [ ] Implement schema
  - [ ] Implement Create() method
  - [ ] Implement Read() method
  - [ ] Implement Update() method
  - [ ] Implement Delete() method
  - [ ] Implement ImportState() method
  - [ ] Verify tests pass
  - [ ] Add examples
- [ ] Phase 3: Documentation & Finalization
  - [ ] Generate docs with `tfplugindocs`
  - [ ] Update provider registration
  - [ ] Run acceptance tests
  - [ ] Update CHANGELOG.md

## File Locations

```
internal/vps/data/keypair_data_source.go          # NEW - Data source implementation
internal/vps/data/keypair_data_source_test.go     # NEW - Data source tests
internal/vps/resource/keypair_resource.go         # NEW - Resource implementation  
internal/vps/resource/keypair_resource_test.go    # NEW - Resource tests
examples/data-sources/zillaforge_keypairs/        # NEW - Data source examples
examples/resources/zillaforge_keypair/            # NEW - Resource examples
docs/data-sources/keypairs.md                     # Generated
docs/resources/keypair.md                         # Generated
```

## Quick Reference Patterns

### Data Source Structure (Follow flavor_data_source.go)

```go
package data

import (
    "context"
    "fmt"
    
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    keypairsmodels "github.com/Zillaforge/cloud-sdk/models/vps/keypairs"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &KeypairDataSource{}

func NewKeypairDataSource() datasource.DataSource {
    return &KeypairDataSource{}
}

type KeypairDataSource struct {
    client *cloudsdk.ProjectClient
}

type KeypairDataSourceModel struct {
    ID       types.String     `tfsdk:"id"`
    Name     types.String     `tfsdk:"name"`
    Keypairs []KeypairModel   `tfsdk:"keypairs"`
}

type KeypairModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    PublicKey   types.String `tfsdk:"public_key"`
    Fingerprint types.String `tfsdk:"fingerprint"`
}

func (d *KeypairDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_keypairs"
}

func (d *KeypairDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    // See contracts/keypair-data-source-schema.md
}

func (d *KeypairDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
    if ok {
        d.client = projectClient
    }
}

func (d *KeypairDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // See data-model.md for logic
    var data KeypairDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    
    // 1. Validate mutual exclusivity
    // 2. Call appropriate SDK method (Get or List)
    // 3. Map results to KeypairModel
    // 4. Set state
}
```

### Resource Structure

```go
package resource

import (
    "context"
    "fmt"
    
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    keypairsmodels "github.com/Zillaforge/cloud-sdk/models/vps/keypairs"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &KeypairResource{}
var _ resource.ResourceWithImportState = &KeypairResource{}

func NewKeypairResource() resource.Resource {
    return &KeypairResource{}
}

type KeypairResource struct {
    client *cloudsdk.ProjectClient
}

type KeypairResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    PublicKey   types.String `tfsdk:"public_key"`
    PrivateKey  types.String `tfsdk:"private_key"` // Sensitive
    Fingerprint types.String `tfsdk:"fingerprint"`
}

func (r *KeypairResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_keypair"
}

func (r *KeypairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    // See contracts/keypair-resource-schema.md
}

func (r *KeypairResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
    if ok {
        r.client = projectClient
    }
}

func (r *KeypairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // See contracts/keypair-resource-schema.md
}

func (r *KeypairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    // See contracts/keypair-resource-schema.md
}

func (r *KeypairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // See contracts/keypair-resource-schema.md
}

func (r *KeypairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // Log warning then delete - see contracts/keypair-resource-schema.md
}

func (r *KeypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // See contracts/keypair-resource-schema.md
}
```

### Provider Registration

Update `internal/provider/provider.go`:

```go
import (
    keypairdata "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/data"
    keypairresource "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/resource"
)

func (p *ZillaforgeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        data.NewFlavorDataSource,
        data.NewNetworkDataSource,
        keypairdata.NewKeypairDataSource,  // NEW
    }
}

func (p *ZillaforgeProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        keypairresource.NewKeypairResource,  // NEW
    }
}
```

## Testing Quick Reference

### Acceptance Test Structure

```go
package data

import (
    "testing"
    
    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKeypairDataSource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccKeypairDataSourceConfig_basic(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "keypairs.#", "1"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_keypairs.test", "keypairs.0.id"),
                    resource.TestCheckResourceAttr("data.zillaforge_keypairs.test", "keypairs.0.name", "test-keypair"),
                ),
            },
        },
    })
}

func testAccKeypairDataSourceConfig_basic() string {
    return `
data "zillaforge_keypairs" "test" {
  name = "test-keypair"
}
`
}
```

### Resource Test with Create/Delete

```go
func TestAccKeypairResource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read
            {
                Config: testAccKeypairResourceConfig_basic("test-keypair"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("zillaforge_keypair.test", "name", "test-keypair"),
                    resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "id"),
                    resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "public_key"),
                    resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "private_key"),
                    resource.TestCheckResourceAttrSet("zillaforge_keypair.test", "fingerprint"),
                ),
            },
            // Import
            {
                ResourceName:      "zillaforge_keypair.test",
                ImportState:       true,
                ImportStateVerify: true,
                ImportStateVerifyIgnore: []string{"private_key"}, // Not available after creation
            },
        },
    })
}

func testAccKeypairResourceConfig_basic(name string) string {
    return fmt.Sprintf(`
resource "zillaforge_keypair" "test" {
  name = %[1]q
}
`, name)
}
```

## Common cloud-sdk Operations

```go
// List keypairs
vpsClient := projectClient.VPS()
keypairs, err := vpsClient.Keypairs().List(ctx, &keypairsmodels.ListKeypairsOptions{
    Name: "optional-filter",
})

// Get single keypair
keypair, err := vpsClient.Keypairs().Get(ctx, "keypair-id")

// Create keypair
createReq := &keypairsmodels.KeypairCreateRequest{
    Name:        "my-keypair",
    Description: "optional",
    PublicKey:   "optional - omit for generation",
}
keypair, err := vpsClient.Keypairs().Create(ctx, createReq)

// Update description
updateReq := &keypairsmodels.KeypairUpdateRequest{
    Description: "new description",
}
keypair, err := vpsClient.Keypairs().Update(ctx, "keypair-id", updateReq)

// Delete keypair
err := vpsClient.Keypairs().Delete(ctx, "keypair-id")
```

## Running Tests

```bash
# Run acceptance tests
TF_ACC=1 go test ./internal/vps/data/... -v -run TestAccKeypairDataSource
TF_ACC=1 go test ./internal/vps/resource/... -v -run TestAccKeypairResource

# Generate documentation
make generate

# Run all acceptance tests
make testacc

# Lint
golangci-lint run
```

## Common Pitfalls

1. **Sensitive Attribute**: Don't forget `Sensitive: true` on private_key
2. **RequiresReplace**: Add to name and public_key plan modifiers
3. **UseStateForUnknown**: Add to computed fields to preserve state
4. **Mutual Exclusivity**: Validate id/name filters in Read()
5. **Private Key Preservation**: In Read(), preserve private_key from state
6. **Import State**: Set private_key to null (not available from API)
7. **Error Messages**: Make them actionable with specific guidance

## Reference Documents

- **Detailed Spec**: [spec.md](spec.md)
- **Research**: [research.md](research.md)
- **Data Model**: [data-model.md](data-model.md)
- **Data Source Contract**: [contracts/keypair-data-source-schema.md](contracts/keypair-data-source-schema.md)
- **Resource Contract**: [contracts/keypair-resource-schema.md](contracts/keypair-resource-schema.md)
- **Existing Patterns**: 
  - [internal/vps/data/flavor_data_source.go](../../internal/vps/data/flavor_data_source.go)
  - [internal/vps/data/network_data_source.go](../../internal/vps/data/network_data_source.go)

## TDD Workflow Reminder

```
1. Write test (RED) → Test fails
2. Write minimal code (GREEN) → Test passes  
3. Refactor (REFACTOR) → Test still passes
4. Repeat for next behavior
```

**Constitution**: TDD is NON-NEGOTIABLE. Write tests first!

---

**Ready to implement!** Start with data source tests, then resource tests. Follow the contracts closely.
