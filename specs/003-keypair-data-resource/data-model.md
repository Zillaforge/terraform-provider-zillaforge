# Data Model: Keypair Data Source and Resource

**Feature**: 003-keypair-data-resource  
**Date**: December 13, 2025  
**Status**: Phase 1 Design

## Entity Overview

This document defines the data models for SSH keypair management in Terraform Provider Zillaforge. Two primary entities are defined: the Keypair resource for managing keypairs, and the Keypair data source for querying existing keypairs.

## Entity: Keypair (Resource)

Represents a managed SSH keypair for VPS instance access in ZillaForge.

### Attributes

| Attribute | Type | Mode | Validation | Description |
|-----------|------|------|------------|-------------|
| `id` | string | Computed | UUID format | Unique identifier assigned by API |
| `name` | string | Required | API-validated (FR-017) | Human-readable keypair name (immutable) |
| `description` | string | Optional, Computed | Max length per API | Optional description (updatable) |
| `public_key` | string | Optional, Computed | OpenSSH format (API-validated) | SSH public key. If omitted, system generates keypair |
| `private_key` | string | Computed, Sensitive | - | Private key (only for system-generated, returned once) |
| `fingerprint` | string | Computed | SHA256 or MD5 hash | Cryptographic fingerprint of public key |

### Immutability Rules

- **Immutable after creation**: `name`, `public_key`
  - Changes trigger `RequiresReplace` (resource recreation)
- **Updatable**: `description`
  - Uses API Update endpoint
- **Never updatable**: `id`, `fingerprint`, `private_key`
  - Managed by API/system

### State Transitions

```
[Configuration] 
    ↓
┌─────────────────────────┐
│ CREATE                  │
│ - name: "my-key"        │ → API POST /keypairs → [State: Created]
│ - public_key: (opt)     │                            ↓
└─────────────────────────┘                    ┌─────────────────┐
                                               │ id: "uuid"      │
                                               │ name: "my-key"  │
                                               │ public_key: "..." │
[Modify description]                           │ private_key: "..." (if generated) │
    ↓                                          │ fingerprint: "..." │
┌─────────────────────────┐                    └─────────────────┘
│ UPDATE                  │                            ↓
│ - description: "new"    │ → API PUT /keypairs/{id} → [State: Updated]
└─────────────────────────┘                            ↓
                                                       
[Modify name/public_key]                               
    ↓                                                   
┌─────────────────────────┐
│ REPLACE (Delete+Create) │
│ - Forces recreation     │ → DELETE old → CREATE new
└─────────────────────────┘

[terraform destroy]
    ↓
┌─────────────────────────┐
│ DELETE                  │ → API DELETE /keypairs/{id} → [State: Removed]
└─────────────────────────┘
```

### Validation Rules

1. **Name** (Enforced by API per FR-017):
   - Uniqueness within account (FR-006)
   - Format constraints defined by ZillaForge API
   - Duplicate names return error immediately (per clarification)

2. **Public Key** (Enforced by API per FR-007):
   - Must be valid OpenSSH format (ssh-rsa, ecdsa-sha2-*, ssh-ed25519)
   - Invalid format returns clear error message
   - Optional - omit for system-generated keypair

3. **Description**:
   - Optional
   - Maximum length enforced by API

### Relationships

- **Belongs to**: Project (via ProjectClient in provider configuration)
- **Used by**: VPS Instances (tracked externally, deletion warnings per FR-015)

### Examples

**System-Generated Keypair** (no public_key provided):
```hcl
resource "zillaforge_keypair" "auto" {
  name        = "auto-generated-key"
  description = "Automatically generated SSH keypair"
  # public_key omitted - system generates both keys
}

# Access generated keys
output "public_key" {
  value = zillaforge_keypair.auto.public_key
}

output "private_key" {
  value     = zillaforge_keypair.auto.private_key
  sensitive = true
}
```

**User-Provided Keypair** (import existing public key):
```hcl
resource "zillaforge_keypair" "imported" {
  name       = "my-existing-key"
  public_key = file("~/.ssh/id_ed25519.pub")
  # private_key will be null (not managed by provider)
}
```

---

## Entity: Keypair (Data Source)

Represents a query for existing SSH keypairs in ZillaForge. Supports both individual lookup and listing all keypairs.

### Attributes

| Attribute | Type | Mode | Validation | Description |
|-----------|------|------|------------|-------------|
| `id` | string | Optional | UUID format | Filter by specific keypair ID (mutually exclusive with name) |
| `name` | string | Optional | - | Filter by exact keypair name (mutually exclusive with id) |
| `keypairs` | list(object) | Computed | - | List of matching keypair objects |

### Nested Object: keypairs (each item)

| Attribute | Type | Mode | Description |
|-----------|------|------|-------------|
| `id` | string | Computed | Unique keypair identifier |
| `name` | string | Computed | Keypair name |
| `description` | string | Computed | Optional description |
| `public_key` | string | Computed | SSH public key |
| `fingerprint` | string | Computed | Public key fingerprint |

**Note**: `private_key` is NOT exposed in data source (security - only available at resource creation time).

### Query Modes

1. **Single Lookup by ID**:
   ```hcl
   data "zillaforge_keypairs" "specific" {
     id = "550e8400-e29b-41d4-a716-446655440000"
   }
   # Returns: list with 1 item if found, error if not found
   ```

2. **Single Lookup by Name**:
   ```hcl
   data "zillaforge_keypairs" "by_name" {
     name = "production-key"
   }
   # Returns: list with items matching exact name
   ```

3. **List All Keypairs**:
   ```hcl
   data "zillaforge_keypairs" "all" {
     # No filters - returns all keypairs in project
   }
   ```

4. **Invalid: Both Filters** (returns validation error):
   ```hcl
   data "zillaforge_keypairs" "invalid" {
     id   = "uuid"
     name = "name"  # ERROR: only one filter allowed
   }
   ```

### Validation Rules

1. **Mutual Exclusivity**: `id` and `name` filters cannot be used together
   - Error message: "Only one of 'id' or 'name' can be specified, not both"

2. **Not Found Handling**:
   - ID not found: Error diagnostic (FR-012)
   - Name not found: Empty list (consistent with flavors/networks pattern)

### Relationships

- **Belongs to**: Project (via ProjectClient)
- **Data Source**: Read-only, does not manage resources

### Examples

**List All and Filter Client-Side**:
```hcl
data "zillaforge_keypairs" "all" {}

locals {
  ed25519_keys = [
    for k in data.zillaforge_keypairs.all.keypairs :
    k if startswith(k.public_key, "ssh-ed25519")
  ]
}
```

**Reference Existing Keypair**:
```hcl
data "zillaforge_keypairs" "existing" {
  name = "shared-team-key"
}

resource "zillaforge_vps_instance" "server" {
  # ... other config ...
  keypair_id = data.zillaforge_keypairs.existing.keypairs[0].id
}
```

---

## Go Implementation Models

### Resource Models (in `internal/vps/resource/keypair_resource.go`)

```go
// KeypairResourceModel describes the Terraform resource data model
type KeypairResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    PublicKey   types.String `tfsdk:"public_key"`
    PrivateKey  types.String `tfsdk:"private_key"`  // Sensitive
    Fingerprint types.String `tfsdk:"fingerprint"`
}
```

### Data Source Models (in `internal/vps/data/keypair_data_source.go`)

```go
// KeypairDataSourceModel describes the data source config and filters
type KeypairDataSourceModel struct {
    ID       types.String     `tfsdk:"id"`       // Optional filter
    Name     types.String     `tfsdk:"name"`     // Optional filter
    Keypairs []KeypairModel   `tfsdk:"keypairs"` // Computed results
}

// KeypairModel represents a single keypair in the results list
type KeypairModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    PublicKey   types.String `tfsdk:"public_key"`
    Fingerprint types.String `tfsdk:"fingerprint"`
    // Note: PrivateKey intentionally omitted from data source
}
```

### Mapping from cloud-sdk Models

```go
// cloud-sdk Keypair struct (from github.com/Zillaforge/cloud-sdk/models/vps/keypairs)
type Keypair struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Description string         `json:"description,omitempty"`
    PublicKey   string         `json:"public_key"`
    Fingerprint string         `json:"fingerprint"`
    UserID      string         `json:"user_id"`
    PrivateKey  string         `json:"private_key,omitempty"` // Only in Create response
    User        *common.IDName `json:"user,omitempty"`
    CreatedAt   string         `json:"createdAt"`
    UpdatedAt   string         `json:"updatedAt"`
}

// Conversion function (example)
func keypairToModel(sdk *keypairs.Keypair) KeypairModel {
    return KeypairModel{
        ID:          types.StringValue(sdk.ID),
        Name:        types.StringValue(sdk.Name),
        Description: types.StringValue(sdk.Description),
        PublicKey:   types.StringValue(sdk.PublicKey),
        Fingerprint: types.StringValue(sdk.Fingerprint),
    }
}
```

## Data Flow Diagrams

### Resource Create Flow

```
User Config (name, public_key?) 
    ↓
Terraform Apply
    ↓
Provider: KeypairResource.Create()
    ↓
cloud-sdk: keypairs.Create(KeypairCreateRequest)
    ↓
ZillaForge API: POST /api/v1/project/{id}/keypairs
    ↓
API Response: Keypair (with private_key if generated)
    ↓
Provider: Map to KeypairResourceModel
    ↓
Terraform State: Save (private_key marked sensitive)
```

### Data Source Read Flow

```
User Config (id? / name? / neither)
    ↓
Terraform Plan/Refresh
    ↓
Provider: KeypairDataSource.Read()
    ↓
Validate: id and name not both set
    ↓
cloud-sdk: keypairs.Get(id) OR keypairs.List(opts)
    ↓
ZillaForge API: GET /api/v1/project/{id}/keypairs/{id} OR /keypairs?name=...
    ↓
API Response: Keypair OR KeypairListResponse
    ↓
Provider: Map to []KeypairModel
    ↓
Terraform State: Save to keypairs attribute
```

## Edge Cases Handling

### Duplicate Name on Create

```
API Error: 409 Conflict "Keypair name 'my-key' already exists"
    ↓
Provider Error Diagnostic: "Keypair name 'my-key' already exists. Choose a unique name."
    ↓
Terraform Apply: Fails with clear error (per FR-006)
```

### Invalid Public Key Format

```
API Error: 400 Bad Request "Invalid public key format"
    ↓
Provider Error Diagnostic: "Invalid public key format. Expected OpenSSH format (ssh-rsa, ecdsa-sha2-nistp256, ssh-ed25519)."
    ↓
Terraform Apply: Fails with actionable error (per FR-007)
```

### Import Without Private Key

```
terraform import zillaforge_keypair.example uuid
    ↓
Provider: Call keypairs.Get(uuid)
    ↓
API Response: Keypair (no private_key field - never stored)
    ↓
Provider: Set all fields except private_key (null/unknown)
    ↓
Terraform State: Imported successfully, private_key = null
```

### Data Source Not Found by ID

```
Data Source Config: id = "non-existent-uuid"
    ↓
cloud-sdk: keypairs.Get("non-existent-uuid")
    ↓
API Error: 404 Not Found
    ↓
Provider Error Diagnostic: "Keypair ID 'non-existent-uuid' not found. Verify the ID is correct."
    ↓
Terraform Plan: Fails with error (per FR-012)
```

### Data Source Not Found by Name

```
Data Source Config: name = "non-existent-name"
    ↓
cloud-sdk: keypairs.List(&ListKeypairsOptions{Name: "non-existent-name"})
    ↓
API Response: Empty list []
    ↓
Provider: Return empty keypairs list
    ↓
Terraform State: keypairs = [] (no error, consistent with flavors pattern)
```

## Conformance to Spec

| Requirement | Implementation |
|-------------|----------------|
| FR-001: Create with unique name | Enforced by API, provider passes validation errors |
| FR-002: Support user/system keys | Optional public_key field |
| FR-003: Private key once, sensitive | Returned in Create response only, marked Sensitive |
| FR-004: Delete keypairs | Delete() method implemented |
| FR-005: Query by name/ID | Data source with id/name filters |
| FR-006: Prevent duplicate names | API returns error, provider propagates clearly |
| FR-007: Validate public key | API validates, provider shows actionable errors |
| FR-008: Import support | ImportState() method implemented |
| FR-009: Track metadata | fingerprint in model |
| FR-010: CRUD lifecycle | Create, Read, Update(desc), Delete implemented |
| FR-011: Unique identifiers | ID field from API |
| FR-012: Error on not found | Data source returns error for ID lookup |
| FR-013: List all keypairs | Data source with no filters |
| FR-014: Indicate replacement | RequiresReplace plan modifiers |
| FR-015: Delete with warning | tflog.Warn() in Delete() method |
| FR-016: No blocking | Warning logged, deletion proceeds |
| FR-017: API validation | Name rules enforced by API |

**Status**: ✅ All functional requirements mapped to data model and implementation approach

---

## Next Steps

1. ✅ Phase 1.1: Generate contract schemas (see `contracts/` directory)
2. Phase 1.2: Create quickstart.md for developer reference
3. Phase 1.3: Update agent context with new technology
4. Phase 2: Generate tasks.md for implementation workflow
