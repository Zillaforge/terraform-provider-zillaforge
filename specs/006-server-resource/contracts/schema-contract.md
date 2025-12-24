# Schema Contract: zillaforge_server Resource

**Resource Name**: `zillaforge_server`  
**Purpose**: Manage ZillaForge VPS virtual machine instances  
**API Endpoint**: `POST/GET/PATCH/DELETE /v1/projects/{project_id}/vps/servers`

---

## Resource Schema

### Required Attributes

#### `name` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `true`
- **Computed**: `false`
- **Sensitive**: `false`
- **Validators**: Length 1-255 characters
- **Plan Modifiers**: None (updateable in-place)
- **Description**: Display name for the server instance. Must be unique within the project.
- **MarkdownDescription**: The name of the server instance. Must be unique within the project and between 1-255 characters.

#### `flavor` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `true`
- **Computed**: `false`
- **Sensitive**: `false`
- **Validators**: Must reference valid flavor ID from `zillaforge_flavors` data source
- **Plan Modifiers**: `stringplanmodifier.RequiresReplace()`
- **Description**: Flavor ID defining CPU, RAM, and disk configuration. Cannot be changed after creation (requires instance recreation).
- **MarkdownDescription**: The ID of the flavor (instance type) to use for this server. Defines the virtual CPU count, memory, and root disk size. **Changing this attribute will force recreation of the instance.** Use the `zillaforge_flavors` data source to list available flavors.

#### `image` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `true`
- **Computed**: `false`
- **Sensitive**: `false`
- **Validators**: Must reference valid image ID from `zillaforge_images` data source
- **Plan Modifiers**: `stringplanmodifier.RequiresReplace()`
- **Description**: Image ID for the operating system. Cannot be changed after creation (requires instance recreation).
- **MarkdownDescription**: The ID of the image to use for the server's operating system. **Changing this attribute will force recreation of the instance.** Use the `zillaforge_images` data source to list available images.

#### `network_attachment` (List of Objects)
- **Type**: `schema.ListNestedAttribute`
- **Required**: `true`
- **Computed**: `false`
- **Sensitive**: `false`
- **Validators**: 
  - `listvalidator.SizeAtLeast(1)` - At least one network attachment required
  - Custom validator: `validators.NetworkAttachmentPrimaryConstraint()` - At most one `primary=true`
- **Plan Modifiers**: None (updateable in-place)
- **Description**: List of network interfaces to attach to the server. At least one network is required. Use the `primary` attribute to designate the primary network interface.
- **MarkdownDescription**: Network interfaces to attach to the server. Each block defines a network connection. At least one network attachment is required, and at most one can be marked as `primary=true`.

**Nested Attributes**:

- **`network_id` (String)**
  - **Type**: `schema.StringAttribute`
  - **Required**: `true`
  - **Computed**: `false`
  - **Description**: ID of the network to attach
  - **MarkdownDescription**: The ID of the network to attach. Use the `zillaforge_networks` data source to list available networks.

- **`ip_address` (String)**
  - **Type**: `schema.StringAttribute`
  - **Required**: `false`
  - **Computed**: `false`
  - **Validators**: `stringvalidator.RegexMatches()` for IPv4 format
  - **Description**: Optional fixed IP address. If not specified, IP is assigned via DHCP.
  - **MarkdownDescription**: Optional fixed IPv4 address to assign to this network interface. If not specified, an IP address will be automatically assigned via DHCP. Must be a valid IPv4 address within the network's CIDR range.

- **`primary` (Boolean)**
  - **Type**: `schema.BoolAttribute`
  - **Required**: `false`
  - **Computed**: `false`
  - **Default**: `false`
  - **Description**: Whether this is the primary network interface. At most one attachment can be primary.
  - **MarkdownDescription**: Whether this is the primary network interface for the server. At most one network attachment can have `primary=true`. The primary interface is used for default routing.

#### `security_group_ids` (List of Strings) — nested inside `network_attachment`
- **Type**: `schema.ListAttribute` with `ElementType: types.StringType`
- **Required**: `true` (per NIC)
- **Computed**: `false`
- **Sensitive**: `false`
- **Validators**: `listvalidator.SizeAtLeast(1)` - At least one security group required per NIC
- **Plan Modifiers**: None (updateable in-place)
- **Description**: List of security group IDs to apply to this network interface (NIC). At least one security group is required per network attachment.
- **MarkdownDescription**: Assign security groups to a specific network attachment using `security_group_ids` (list of security group IDs). Use the `zillaforge_security_groups` data source to list available security groups.

---

### Optional Attributes

#### `description` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `false`
- **Computed**: `false`
- **Sensitive**: `false`
- **Validators**: Max length 1000 characters
- **Plan Modifiers**: None (updateable in-place)
- **Description**: Optional description for the server instance.
- **MarkdownDescription**: A human-readable description of the server. Maximum 1000 characters.

#### `keypair` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `false`
- **Computed**: `false`
- **Sensitive**: `false`
- **Validators**: Must reference valid keypair name from `zillaforge_keypairs` data source
- **Plan Modifiers**: `stringplanmodifier.RequiresReplace()`
- **Description**: Name of SSH keypair to inject into the instance. Cannot be changed after creation.
- **MarkdownDescription**: The name of the SSH keypair to inject into the server for authentication. **Changing this attribute will force recreation of the instance.** Use the `zillaforge_keypairs` data source to list available keypairs or create a new one with the `zillaforge_keypair` resource.

#### `user_data` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `false`
- **Computed**: `false`
- **Sensitive**: `true`
- **Validators**: Max 64KB, must be base64-encoded
- **Plan Modifiers**: `stringplanmodifier.RequiresReplace()`
- **Description**: Cloud-init user data for instance initialization. Must be base64-encoded. Cannot be changed after creation.
- **MarkdownDescription**: Cloud-init user data for configuring the server on first boot. Must be base64-encoded (use Terraform's `base64encode()` function). Maximum size 64KB. **Changing this attribute will force recreation of the instance.** The user data is not returned by the API for security reasons, so it will not appear in state after import.

#### `wait_for_active` (Boolean)
- **Type**: `schema.BoolAttribute`
- **Required**: `false`
- **Computed**: `false`
- **Default**: `true`
- **Sensitive**: `false`
- **Validators**: None
- **Plan Modifiers**: None
- **Description**: Whether to wait for the server to reach 'active' status after creation. If false, Terraform will return as soon as the API responds without polling for status.
- **MarkdownDescription**: Whether to wait for the server to reach `active` status after creation. When set to `true` (default), Terraform will poll the server status until it reaches `active` state or the timeout is exceeded. When set to `false`, Terraform will return immediately after the API responds, without waiting for the server to become active. Default is `true`.


---

### Computed Attributes (Read-Only)

#### `id` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `false`
- **Computed**: `true`
- **Sensitive**: `false`
- **Plan Modifiers**: `stringplanmodifier.UseStateForUnknown()`
- **Description**: Unique identifier for the server instance (platform-generated).
- **MarkdownDescription**: The unique identifier for the server instance. Generated by the platform.

#### `status` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `false`
- **Computed**: `true`
- **Sensitive**: `false`
- **Description**: Current status of the server. Possible values: "building", "active", "error", "deleted".
- **MarkdownDescription**: The current status of the server. Possible values: `building` (instance is being created), `active` (instance is running and ready), `error` (instance entered an error state), `deleted` (instance has been deleted).

#### `ip_addresses` (List of Strings)
- **Type**: `schema.ListAttribute` with `ElementType: types.StringType`
- **Required**: `false`
- **Computed**: `true`
- **Sensitive**: `false`
- **Description**: List of IP addresses assigned to the server (one per network attachment).
- **MarkdownDescription**: List of IP addresses assigned to the server. The order corresponds to the order of `network_attachment` blocks. Includes both DHCP-assigned and fixed IP addresses.

#### `created_at` (String)
- **Type**: `schema.StringAttribute`
- **Required**: `false`
- **Computed**: `true`
- **Sensitive**: `false`
- **Description**: Timestamp when the server was created (RFC3339 format).
- **MarkdownDescription**: The timestamp when the server was created, in RFC3339 format (e.g., `2023-10-15T14:30:00Z`).

---

### Timeouts Block

#### `timeouts` (Object)
- **Type**: `schema.SingleNestedAttribute`
- **Required**: `false`
- **Computed**: `false`
- **Description**: Configurable timeouts for resource operations.
- **MarkdownDescription**: Configurable timeouts for create, update, and delete operations.

**Nested Attributes**:

- **`create` (String)**
  - **Type**: `schema.StringAttribute`
  - **Required**: `false`
  - **Default**: `"10m"`
  - **Description**: Timeout for create operations (e.g., "10m", "1h")
  - **MarkdownDescription**: Maximum time to wait for server creation to complete. Default is `10m` (10 minutes). Use Go duration syntax (e.g., `15m`, `1h`).

- **`update` (String)**
  - **Type**: `schema.StringAttribute`
  - **Required**: `false`
  - **Default**: `"10m"`
  - **Description**: Timeout for update operations (e.g., "10m", "1h")
  - **MarkdownDescription**: Maximum time to wait for server updates to complete. Default is `10m` (10 minutes). Use Go duration syntax (e.g., `15m`, `1h`).

- **`delete` (String)**
  - **Type**: `schema.StringAttribute`
  - **Required**: `false`
  - **Default**: `"10m"`
  - **Description**: Timeout for delete operations (e.g., "10m", "1h")
  - **MarkdownDescription**: Maximum time to wait for server deletion to complete. Default is `10m` (10 minutes). Use Go duration syntax (e.g., `15m`, `1h`).

---

## Validation Rules

### Schema Validation (Framework)
1. **network_attachment**: At least 1 block required (`listvalidator.SizeAtLeast(1)`)
2. **network_attachment**: At most 1 block with `primary=true` (custom validator)
3. **network_attachment.ip_address**: Valid IPv4 format (`stringvalidator.RegexMatches()`)
4. **security_groups**: At least 1 security group required (`listvalidator.SizeAtLeast(1)`)

### API Validation (Cloud-SDK)
1. **name**: Unique within project, 1-255 characters
2. **flavor**: Must exist and be available in project
3. **image**: Must exist and be compatible with flavor
4. **network_attachment.network_id**: Must exist in project
5. **network_attachment.ip_address**: Must be within network CIDR, not already allocated
6. **security_groups**: All IDs must exist in project
7. **keypair**: Must exist in project (if specified)
8. **user_data**: Max 64KB, valid base64 encoding

### State Validation (Timeouts)
1. **timeouts.create**: Valid Go duration string (e.g., "10m", "1h")
2. **timeouts.update**: Valid Go duration string
3. **timeouts.delete**: Valid Go duration string

---

## Plan Modifiers

### RequiresReplace (Force Recreation)
Applied to attributes that cannot be changed in-place:
- `flavor` - Instance type cannot be resized (flavor resize is out of scope)
- `image` - Operating system image cannot be changed after creation
- `keypair` - SSH keypair cannot be changed after injection
- `user_data` - Cloud-init data only applied at creation time

### UseStateForUnknown
Applied to computed attributes to preserve values during plan:
- `id` - Instance ID remains stable after creation

---

## Import Support

### Import ID Format
```
terraform import zillaforge_server.example srv-abc123def456
```

Where `srv-abc123def456` is the server ID.

### Import Behavior
1. Fetch server details from API using `GET /servers/{id}`
2. Map all attributes to Terraform state
3. Map `network_attachment` blocks from API `network_ports` array
4. Map `security_groups` list from API response
5. **Ignore `user_data`**: API does not return user_data for security reasons

### Import Verification
Use `ImportStateVerify: true` in acceptance tests, with:
```go
ImportStateVerifyIgnore: []string{"user_data"}
```

---

## Error Handling

### HTTP Error Codes

| Code | Scenario | Terraform Diagnostic |
|------|----------|----------------------|
| 400 | Bad Request (validation error) | "Invalid configuration: {api_error_message}" |
| 404 | Resource not found | "Server not found. It may have been deleted outside Terraform." |
| 409 | Conflict (e.g., name already exists) | "Server name already exists: {name}. Choose a different name." |
| 422 | Unprocessable Entity (quota limit) | "Quota exceeded: {api_error_message}. Contact support to increase limits." |
| 429 | Rate limit exceeded | "API rate limit exceeded. Retrying in {backoff}s..." |
| 500 | Internal server error | "API error: {api_error_message}. Please try again or contact support." |

### Timeout Errors

| Operation | Error Message |
|-----------|---------------|
| Create | "Timeout waiting for server to become active. Current status: {status}. Increase timeout with `timeouts.create` or check server console logs." |
| Update | "Timeout waiting for server update to complete. Increase timeout with `timeouts.update`." |
| Delete | "Timeout waiting for server to be deleted. Increase timeout with `timeouts.delete`." |

### Validation Errors

| Validation | Error Message |
|------------|---------------|
| Multiple primary networks | "Only one network attachment can have primary=true, found {count}." |
| Invalid IP address | "Invalid IP address '{ip}'. Must be a valid IPv4 address." |
| No network attachments | "At least one network_attachment is required." |
| No security groups | "At least one security group is required." |

---

## State Transitions

### Create Flow
```
User: terraform apply
  ↓
[Validate] Schema validation (framework)
  ↓
[Validate] Custom validators (primary constraint, IP format)
  ↓
[API] POST /servers → Response: {"id": "srv-123", "status": "building"}
  ↓
[Poll] GET /servers/srv-123 → {"status": "building"} (wait, retry with backoff)
  ↓
[Poll] GET /servers/srv-123 → {"status": "active"} ✓
  ↓
[State] Write to Terraform state
  ↓
User: Server created successfully
```

### Update Flow
```
User: terraform apply (after config change)
  ↓
[Diff] Compare plan vs state
  ↓
[Check] Plan modifiers: RequiresReplace?
  ├─ YES → Destroy + Create (force replacement)
  └─ NO → In-place update
       ↓
      [API] PATCH /servers/srv-123 → Response: {"status": "active"}
       ↓
      [State] Update Terraform state
       ↓
      User: Server updated successfully
```

### Delete Flow
```
User: terraform destroy
  ↓
[API] DELETE /servers/srv-123 → Response: 204 No Content
  ↓
[Poll] GET /servers/srv-123 → Response: 404 Not Found ✓
  ↓
[State] Remove from Terraform state
  ↓
User: Server destroyed successfully
```

---

## Example Schema Implementation

```go
func (r *ServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Manages a ZillaForge VPS server instance.",
        
        Attributes: map[string]schema.Attribute{
            // Required attributes
            "name": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "The name of the server instance. Must be unique within the project and between 1-255 characters.",
            },
            
            "flavor": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "The ID of the flavor (instance type) to use for this server. **Changing this attribute will force recreation of the instance.**",
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                },
            },
            
            "image": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "The ID of the image to use for the server's operating system. **Changing this attribute will force recreation of the instance.**",
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                },
            },
            
            "network_attachment": schema.ListNestedAttribute{
                Required:            true,
                MarkdownDescription: "Network interfaces to attach to the server. At least one network attachment is required.",
                Validators: []validator.List{
                    listvalidator.SizeAtLeast(1),
                    validators.NetworkAttachmentPrimaryConstraint(),
                },
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "network_id": schema.StringAttribute{
                            Required:            true,
                            MarkdownDescription: "The ID of the network to attach.",
                        },
                        "ip_address": schema.StringAttribute{
                            Optional:            true,
                            MarkdownDescription: "Optional fixed IPv4 address. Must be valid IPv4 within network CIDR.",
                            Validators: []validator.String{
                                validators.IPv4Address(),
                            },
                        },
                        "primary": schema.BoolAttribute{
                            Optional:            true,
                            MarkdownDescription: "Whether this is the primary network interface. At most one can be primary.",
                        },
                    },
                },
            },
            
            "security_groups": schema.ListAttribute{
                Required:            true,
                ElementType:         types.StringType,
                MarkdownDescription: "List of security group IDs. At least one is required.",
                Validators: []validator.List{
                    listvalidator.SizeAtLeast(1),
                },
            },
            
            // Computed attributes
            "id": schema.StringAttribute{
                Computed:            true,
                MarkdownDescription: "The unique identifier for the server instance.",
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            
            "status": schema.StringAttribute{
                Computed:            true,
                MarkdownDescription: "The current status of the server (building, active, error, deleted).",
            },
            
            // ... other attributes ...
        },
    }
}
```

---

## Testing Contract

### Acceptance Test Cases

1. **TestAccServerResource_basic**: Create server with minimum required fields
2. **TestAccServerResource_complete**: Create server with all optional fields
3. **TestAccServerResource_multiNetwork**: Create server with multiple network attachments
4. **TestAccServerResource_updateName**: Update name in-place
5. **TestAccServerResource_updateNetworks**: TODO - in-place network updates planned but not yet implemented (expect a TODO warning)
6. **TestAccServerResource_updateSecurityGroups**: TODO - in-place security group updates planned but not yet implemented (expect a TODO warning)
7. **TestAccServerResource_updateFlavor**: Verify force replacement when flavor changes
8. **TestAccServerResource_import**: Import existing server
9. **TestAccServerResource_multiplePrimary**: Validate error for multiple primary networks
10. **TestAccServerResource_invalidIP**: Validate error for invalid IP address format

### Test Configuration Template

```hcl
resource "zillaforge_server" "test" {
  name   = "test-server-%s"
  flavor = data.zillaforge_flavors.test.flavors[0].id
  image  = data.zillaforge_images.test.images[0].id
  
  network_attachment {
    network_id = data.zillaforge_networks.test.networks[0].id
    primary    = true
    sg_ids     = [data.zillaforge_security_groups.test.security_groups[0].id]
  }
}
```

---

## Summary

- **12 Total Attributes**: 5 required, 4 optional, 4 computed, 1 nested block, 1 timeouts block
- **Plan Modifiers**: RequiresReplace on flavor, image, keypair, user_data
- **Validators**: Custom network_attachment validator, IPv4 validator, size validators
- **Import Support**: By server ID, ignore user_data in verification
- **Timeout Support**: Configurable create/update/delete timeouts (default 10m)
- **Error Handling**: HTTP error mapping, validation errors, timeout errors
