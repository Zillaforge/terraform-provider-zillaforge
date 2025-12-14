# Contract: Keypair Resource Schema

**Feature**: 003-keypair-data-resource  
**Type**: Resource  
**Terraform Name**: `zillaforge_keypair`  
**Date**: December 13, 2025

## Schema Definition

This contract defines the Terraform schema for the keypair resource following the Plugin Framework structure.

### Resource Attributes

| Attribute | Type | Mode | Sensitive | Validators | Plan Modifiers | Description |
|-----------|------|------|-----------|------------|----------------|-------------|
| `id` | String | Computed | No | - | UseStateForUnknown | Unique keypair identifier (UUID) |
| `name` | String | Required | No | - | RequiresReplace | Keypair name (immutable) |
| `description` | String | Optional, Computed | No | - | - | Optional description (updatable) |
| `public_key` | String | Optional, Computed | No | - | RequiresReplace, UseStateForUnknown | SSH public key (immutable, generated if omitted) |
| `private_key` | String | Computed | **Yes** | - | UseStateForUnknown | Private key (only for generated, sensitive) |
| `fingerprint` | String | Computed | No | - | UseStateForUnknown | Public key fingerprint |

## Go Schema Implementation

```go
package resource

import (
    "context"
    
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func (r *KeypairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Manages SSH keypairs for VPS instance access in ZillaForge. Supports both user-provided public keys and system-generated keypairs. Note: keypair name and public key are immutable after creation.",
        
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                MarkdownDescription: "Unique identifier for the keypair (UUID format). Assigned by the API upon creation.",
                Computed:            true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "name": schema.StringAttribute{
                MarkdownDescription: "Human-readable name for the keypair. Must be unique within the project. **Immutable** - changing this value forces resource replacement.",
                Required:            true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                },
            },
            "description": schema.StringAttribute{
                MarkdownDescription: "Optional description providing context about the keypair's purpose or usage. This is the only updatable attribute.",
                Optional:            true,
                Computed:            true,
            },
            "public_key": schema.StringAttribute{
                MarkdownDescription: "SSH public key in OpenSSH format (ssh-rsa, ecdsa-sha2-*, ssh-ed25519). If omitted, the system generates a keypair automatically and returns both public and private keys. **Immutable** - changing this value forces resource replacement.",
                Optional:            true,
                Computed:            true, // Computed if user doesn't provide (system-generated)
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "private_key": schema.StringAttribute{
                MarkdownDescription: "Private key for SSH authentication. **Only available for system-generated keypairs** (when `public_key` is not provided). The private key is returned only once during creation and marked as sensitive to prevent exposure in logs or console output. For user-provided public keys, this field remains null.",
                Computed:            true,
                Sensitive:           true, // Prevents exposure in logs/plan output
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "fingerprint": schema.StringAttribute{
                MarkdownDescription: "Cryptographic fingerprint of the public key (SHA256 or MD5 hash format).",
                Computed:            true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            
        },
    }
}
```

## Usage Examples

### Example 1: System-Generated Keypair

```hcl
resource "zillaforge_keypair" "auto" {
  name        = "auto-generated-key"
  description = "System-generated SSH keypair for web servers"
}

# Access the generated keys
output "public_key" {
  description = "Public key to share with team"
  value       = zillaforge_keypair.auto.public_key
}

output "private_key" {
  description = "Private key for SSH access (save securely)"
  value       = zillaforge_keypair.auto.private_key
  sensitive   = true  # Prevents display in console
}

output "fingerprint" {
  description = "Key fingerprint for verification"
  value       = zillaforge_keypair.auto.fingerprint
}
```

### Example 2: User-Provided Public Key

```hcl
resource "zillaforge_keypair" "imported" {
  name        = "team-shared-key"
  description = "Shared team key for production servers"
  public_key  = file("~/.ssh/id_ed25519.pub")
}

# private_key will be null for user-provided keys
output "keypair_id" {
  value = zillaforge_keypair.imported.id
}
```

### Example 3: Update Description

```hcl
resource "zillaforge_keypair" "example" {
  name        = "example-key"
  description = "Updated description"  # Only this field is updatable
}

# Changing name or public_key triggers replacement:
# Terraform will show: # zillaforge_keypair.example must be replaced
```

### Example 4: Import Existing Keypair

```bash
# Import by keypair ID
terraform import zillaforge_keypair.existing 550e8400-e29b-41d4-a716-446655440000
```

```hcl
# Configuration after import (private_key will be null)
resource "zillaforge_keypair" "existing" {
  name        = "existing-keypair-name"
  description = "Imported from manual creation"
  # public_key automatically populated from import
  # private_key will be null (not available after creation)
}
```

## Behavior Specifications

### Create Operation

**Input**: Configuration with `name` (required) and optional `public_key`, `description`

**Process**:
1. Validate `name` not empty
2. Construct `KeypairCreateRequest`:
   - If `public_key` provided: include in request
   - If `public_key` omitted: omit field (API generates keypair)
3. Call `client.VPS().Keypairs().Create(ctx, request)`
4. Map API response to resource model
5. Save to state (including `private_key` if generated)

**Output**: Resource in state with all computed fields populated

### Read Operation

**Input**: Resource ID from state

**Process**:
1. Call `client.VPS().Keypairs().Get(ctx, id)`
2. Map API response to resource model
3. Preserve `private_key` from state (not in API response)

**Output**: Updated state with current API values

### Update Operation

**Input**: Modified `description` field

**Process**:
1. Check plan for changes
2. If only `description` changed:
   - Call `client.VPS().Keypairs().Update(ctx, id, &KeypairUpdateRequest{Description: desc})`
   - Map response to state
3. If `name` or `public_key` changed:
   - Terraform triggers Delete + Create (RequiresReplace)

**Output**: Updated state

### Delete Operation

**Input**: Resource ID from state

**Process**:
1. Log warning about potential instance access loss (FR-015)
2. Call `client.VPS().Keypairs().Delete(ctx, id)`
3. Remove from state

**Output**: Resource removed from state

### Import Operation

**Input**: Keypair ID from command line

**Process**:
1. Call `client.VPS().Keypairs().Get(ctx, id)`
2. Map all available fields to state
3. Set `private_key` to null (not available from API)

**Output**: Resource in state without `private_key`

## API Integration

### cloud-sdk Method Calls

```go
import (
    cloudsdk "github.com/Zillaforge/cloud-sdk"
    keypairsmodels "github.com/Zillaforge/cloud-sdk/models/vps/keypairs"
)

// Create
func (r *KeypairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan KeypairResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    createReq := &keypairsmodels.KeypairCreateRequest{
        Name:        plan.Name.ValueString(),
        Description: plan.Description.ValueString(),
    }
    
    // Only include public_key if user provided it
    if !plan.PublicKey.IsNull() {
        createReq.PublicKey = plan.PublicKey.ValueString()
    }

    keypair, err := r.client.VPS().Keypairs().Create(ctx, createReq)
    if err != nil {
        resp.Diagnostics.AddError("Create Error", 
            fmt.Sprintf("Unable to create keypair '%s': %s. Verify the public key format is valid OpenSSH (ssh-rsa, ecdsa-sha2-*, ssh-ed25519).", 
                plan.Name.ValueString(), err))
        return
    }

    // Map response to state
    plan.ID = types.StringValue(keypair.ID)
    plan.Name = types.StringValue(keypair.Name)
    plan.Description = types.StringValue(keypair.Description)
    plan.PublicKey = types.StringValue(keypair.PublicKey)
    plan.Fingerprint = types.StringValue(keypair.Fingerprint)
    
    // PrivateKey only in Create response (if generated)
    if keypair.PrivateKey != "" {
        plan.PrivateKey = types.StringValue(keypair.PrivateKey)
    } else {
        plan.PrivateKey = types.StringNull()
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read
func (r *KeypairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state KeypairResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    keypair, err := r.client.VPS().Keypairs().Get(ctx, state.ID.ValueString())
    if err != nil {
        // Handle 404 by removing from state
        resp.State.RemoveResource(ctx)
        return
    }

    // Update state (preserve private_key from existing state)
    privateKey := state.PrivateKey // Preserve
    
    state.Name = types.StringValue(keypair.Name)
    state.Description = types.StringValue(keypair.Description)
    state.PublicKey = types.StringValue(keypair.PublicKey)
    state.Fingerprint = types.StringValue(keypair.Fingerprint)
    state.PrivateKey = privateKey // Restore preserved value

    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update
func (r *KeypairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan, state KeypairResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Only description can be updated
    updateReq := &keypairsmodels.KeypairUpdateRequest{
        Description: plan.Description.ValueString(),
    }

    keypair, err := r.client.VPS().Keypairs().Update(ctx, state.ID.ValueString(), updateReq)
    if err != nil {
        resp.Diagnostics.AddError("Update Error", 
            fmt.Sprintf("Unable to update keypair '%s': %s", state.Name.ValueString(), err))
        return
    }

    // Update state
    plan.PrivateKey = state.PrivateKey // Preserve private key

    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete
func (r *KeypairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state KeypairResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Log warning (FR-015, FR-016)
    tflog.Warn(ctx, "Deleting keypair that may be in use by VPS instances",
        map[string]interface{}{
            "keypair_id":   state.ID.ValueString(),
            "keypair_name": state.Name.ValueString(),
        })

    err := r.client.VPS().Keypairs().Delete(ctx, state.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Delete Error", 
            fmt.Sprintf("Unable to delete keypair '%s': %s", state.Name.ValueString(), err))
        return
    }
}

// ImportState
func (r *KeypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    keypairID := req.ID

    keypair, err := r.client.VPS().Keypairs().Get(ctx, keypairID)
    if err != nil {
        resp.Diagnostics.AddError("Import Error", 
            fmt.Sprintf("Unable to read keypair '%s': %s. Verify the ID is correct.", keypairID, err))
        return
    }

    state := KeypairResourceModel{
        ID:          types.StringValue(keypair.ID),
        Name:        types.StringValue(keypair.Name),
        Description: types.StringValue(keypair.Description),
        PublicKey:   types.StringValue(keypair.PublicKey),
        Fingerprint: types.StringValue(keypair.Fingerprint),
        PrivateKey:  types.StringNull(), // Not available from API
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

## Error Handling

| Scenario | API Response | Provider Behavior | User Message |
|----------|--------------|-------------------|--------------|
| Duplicate name | 409 Conflict | AddError | "Keypair name '{name}' already exists. Choose a unique name." |
| Invalid public key | 400 Bad Request | AddError | "Invalid public key format. Expected OpenSSH format (ssh-rsa, ecdsa-sha2-*, ssh-ed25519)." |
| Keypair not found (Read) | 404 Not Found | RemoveResource | Silent removal from state |
| Keypair not found (Import) | 404 Not Found | AddError | "Keypair ID '{id}' not found. Verify the ID is correct." |
| Quota exceeded | 403 Forbidden / 429 | AddError | "Account keypair limit reached. Delete unused keypairs or contact support to increase quota." |

## Testing Contract

### Acceptance Test Cases

1. **Test: Create with system-generated keypair**
   - Config: name only
   - Expected: private_key populated and sensitive

2. **Test: Create with user-provided public key**
   - Config: name + public_key
   - Expected: private_key null

3. **Test: Update description**
   - Config: Change description
   - Expected: In-place update, no replacement

4. **Test: Change name triggers replacement**
   - Config: Change name
   - Expected: Plan shows "must be replaced"

5. **Test: Import by ID**
   - Command: `terraform import zillaforge_keypair.test <id>`
   - Expected: State populated, private_key null

6. **Test: Delete keypair**
   - Command: `terraform destroy`
   - Expected: Resource deleted, warning logged

7. **Test: Duplicate name error**
   - Config: Create with existing name
   - Expected: Error with actionable message

## State File Example

```json
{
  "mode": "managed",
  "type": "zillaforge_keypair",
  "name": "example",
  "provider": "provider[\"registry.terraform.io/zillaforge/zillaforge\"]",
  "instances": [
    {
      "schema_version": 0,
      "attributes": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "my-keypair",
        "description": "Example keypair",
        "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
        "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
        "fingerprint": "SHA256:abcd1234..."
      },
      "sensitive_attributes": [
        {
          "type": "get_attr",
          "value": "private_key"
        }
      ]
    }
  ]
}
```

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-12-13 | Initial contract definition |

---

**Status**: ✅ Contract complete and ready for implementation
