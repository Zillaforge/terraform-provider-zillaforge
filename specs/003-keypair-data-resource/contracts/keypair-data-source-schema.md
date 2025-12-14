# Contract: Keypair Data Source Schema

**Feature**: 003-keypair-data-resource  
**Type**: Data Source  
**Terraform Name**: `zillaforge_keypairs`  
**Date**: December 13, 2025

## Schema Definition

This contract defines the Terraform schema for the keypair data source following the Plugin Framework structure.

### Data Source Attributes

| Attribute | Type | Mode | Validators | Plan Modifiers | Description |
|-----------|------|------|------------|----------------|-------------|
| `id` | String | Optional | - | - | Filter by specific keypair ID (mutually exclusive with name) |
| `name` | String | Optional | - | - | Filter by exact keypair name (mutually exclusive with id) |
| `keypairs` | List(Nested) | Computed | - | - | List of matching keypair objects |

### Nested Object: keypairs

| Attribute | Type | Mode | Description |
|-----------|------|------|-------------|
| `id` | String | Computed | Unique keypair identifier |
| `name` | String | Computed | Keypair name |
| `description` | String | Computed | Optional description |
| `public_key` | String | Computed | SSH public key content |
| `fingerprint` | String | Computed | Public key fingerprint (SHA256 or MD5) |


## Go Schema Implementation

```go
func (d *KeypairDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Query available SSH keypairs in ZillaForge VPS service. Supports individual lookup by ID or name, and listing all keypairs when no filters are specified.",
        
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                MarkdownDescription: "Filter by specific keypair ID. Mutually exclusive with `name` filter. Returns single keypair if found, error if not found.",
                Optional:            true,
            },
            "name": schema.StringAttribute{
                MarkdownDescription: "Filter by exact keypair name (case-sensitive). Mutually exclusive with `id` filter. Returns all keypairs matching the name.",
                Optional:            true,
            },
            "keypairs": schema.ListNestedAttribute{
                MarkdownDescription: "List of matching keypair objects. Empty list if no matches found (for name filter) or error (for id filter).",
                Computed:            true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                            MarkdownDescription: "Unique identifier for the keypair (UUID format).",
                            Computed:            true,
                        },
                        "name": schema.StringAttribute{
                            MarkdownDescription: "Human-readable keypair name. Must be unique within the project.",
                            Computed:            true,
                        },
                        "description": schema.StringAttribute{
                            MarkdownDescription: "Optional description providing context about the keypair's purpose or usage.",
                            Computed:            true,
                        },
                        "public_key": schema.StringAttribute{
                            MarkdownDescription: "SSH public key in OpenSSH format (e.g., ssh-rsa, ecdsa-sha2-nistp256, ssh-ed25519).",
                            Computed:            true,
                        },
                        "fingerprint": schema.StringAttribute{
                            MarkdownDescription: "Cryptographic fingerprint of the public key (SHA256 or MD5 hash).",
                            Computed:            true,
                        },
                    },
                },
            },
        },
    }
}
```

## Usage Examples

### Example 1: List All Keypairs

```hcl
data "zillaforge_keypairs" "all" {
  # No filters - returns all keypairs in the project
}

output "keypair_count" {
  value = length(data.zillaforge_keypairs.all.keypairs)
}

output "keypair_names" {
  value = [for k in data.zillaforge_keypairs.all.keypairs : k.name]
}
```

### Example 2: Filter by Name

```hcl
data "zillaforge_keypairs" "production" {
  name = "production-keypair"
}

# Reference the first match
resource "zillaforge_vps_instance" "web" {
  # ... other config ...
  keypair_id = length(data.zillaforge_keypairs.production.keypairs) > 0 ? data.zillaforge_keypairs.production.keypairs[0].id : null
}
```

### Example 3: Lookup by ID

```hcl
data "zillaforge_keypairs" "specific" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "keypair_fingerprint" {
  value = data.zillaforge_keypairs.specific.keypairs[0].fingerprint
}
```

### Example 4: Invalid - Both Filters (Error)

```hcl
data "zillaforge_keypairs" "invalid" {
  id   = "uuid"
  name = "name"
  # ERROR: Only one of 'id' or 'name' can be specified
}
```

## Behavior Specifications

### Filter Logic

1. **No Filters** (both `id` and `name` are null):
   - Call: `client.List(ctx, nil)`
   - Returns: All keypairs in the project
   - Empty result: `keypairs = []` (no error)

2. **ID Filter Only**:
   - Call: `client.Get(ctx, id)`
   - Returns: Single keypair wrapped in list `[keypair]`
   - Not found: Error diagnostic (FR-012)

3. **Name Filter Only**:
   - Call: `client.List(ctx, &ListKeypairsOptions{Name: name})`
   - Returns: Keypairs matching exact name
   - No matches: `keypairs = []` (no error, consistent with flavors/networks)

4. **Both Filters**:
   - Validation error before API call
   - Error message: "Only one of 'id' or 'name' can be specified, not both"

### Error Handling

| Scenario | Behavior | Error Message |
|----------|----------|---------------|
| Both id and name set | Validation error | "Only one of 'id' or 'name' can be specified, not both" |
| ID not found | API error (404) | "Keypair ID '{id}' not found. Verify the ID is correct." |
| Name no matches | Empty list | No error, `keypairs = []` |
| API connection error | Connection error | "Failed to list keypairs: {error details}" |

## Validation Rules

### Runtime Validation (in Read() method)

```go
func (d *KeypairDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data KeypairDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Validate mutual exclusivity
    if !data.ID.IsNull() && !data.Name.IsNull() {
        resp.Diagnostics.AddError(
            "Invalid Filter Combination",
            "Only one of 'id' or 'name' can be specified, not both.",
        )
        return
    }

    // ... proceed with API calls
}
```

## API Integration

### cloud-sdk Method Calls

```go
// Import
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    keypairsmodels "github.com/Zillaforge/cloud-sdk/models/vps/keypairs"
)

// Get by ID
func getKeypairByID(ctx context.Context, client *cloudsdk.ProjectClient, id string) ([]KeypairModel, error) {
    keypair, err := client.VPS().Keypairs().Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get keypair %s: %w", id, err)
    }
    return []KeypairModel{keypairToModel(keypair)}, nil
}

// List with optional name filter
func listKeypairs(ctx context.Context, client *cloudsdk.ProjectClient, nameFilter string) ([]KeypairModel, error) {
    opts := &keypairsmodels.ListKeypairsOptions{}
    if nameFilter != "" {
        opts.Name = nameFilter
    }
    
    keypairList, err := client.VPS().Keypairs().List(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to list keypairs: %w", err)
    }
    
    results := make([]KeypairModel, 0, len(keypairList))
    for _, kp := range keypairList {
        // API-side filtering by name if opts.Name is set
        // Additional exact match validation could be added here if needed
        results = append(results, keypairToModel(kp))
    }
    return results, nil
}

// Conversion helper
func keypairToModel(kp *keypairsmodels.Keypair) KeypairModel {
    return KeypairModel{
        ID:          types.StringValue(kp.ID),
        Name:        types.StringValue(kp.Name),
        Description: types.StringValue(kp.Description),
        PublicKey:   types.StringValue(kp.PublicKey),
        Fingerprint: types.StringValue(kp.Fingerprint),
    }
}
```

## Testing Contract

### Acceptance Test Cases

1. **Test: List all keypairs**
   - Config: No filters
   - Expected: All project keypairs returned

2. **Test: Filter by name (exact match)**
   - Config: `name = "test-keypair"`
   - Expected: Keypairs with exact name "test-keypair"

3. **Test: Filter by ID**
   - Config: `id = "<valid-uuid>"`
   - Expected: Single keypair with matching ID

4. **Test: Both filters (error)**
   - Config: Both `id` and `name` set
   - Expected: Validation error

5. **Test: ID not found**
   - Config: `id = "non-existent-uuid"`
   - Expected: Error diagnostic

6. **Test: Name no matches**
   - Config: `name = "non-existent-name"`
   - Expected: Empty keypairs list, no error

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-12-13 | Initial contract definition |

---

**Status**: ✅ Contract complete and ready for implementation
