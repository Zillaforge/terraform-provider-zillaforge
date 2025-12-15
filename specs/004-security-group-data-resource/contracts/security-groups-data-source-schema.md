# Security Groups Data Source Schema

**Data Source Type**: `zillaforge_security_groups`  
**API Mapping**: `GET /api/v1/projects/{project_id}/vps/security-groups`

## Schema Definition

### Filter Attributes

| Attribute | Type | Required | Computed | Validators | Description |
|-----------|------|----------|----------|------------|-------------|
| `id` | String | No | No | UUID format | Optional filter to query a specific security group by ID. Mutually exclusive with `name`. |
| `name` | String | No | No | - | Optional filter to query security groups by exact name (case-sensitive). Mutually exclusive with `id`. |

### Result Attributes

| Attribute | Type | Computed | Description |
|-----------|------|----------|-------------|
| `security_groups` | List[Object] | Yes | List of security group objects matching the filter criteria. Empty list if no matches (for name filter) or error (for id filter). |

### Nested Attribute: `security_groups[]`

Each security group object contains:

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Unique identifier for the security group (UUID format) |
| `name` | String | Human-readable security group name |
| `description` | String | Optional description of security group purpose. Empty string if not set. |
| `ingress_rules` | List[Object] | List of inbound firewall rules. Empty list if no ingress rules defined. |
| `egress_rules` | List[Object] | List of outbound firewall rules. Empty list if no egress rules defined. |

### Nested Attribute: `security_groups[].ingress_rules[]` / `egress_rules[]`

Each rule object contains:

| Attribute | Type | Description |
|-----------|------|-------------|
| `protocol` | String | Network protocol: `tcp`, `udp`, `icmp`, or `any` |
| `port_range` | String | Port or port range: `22`, `80-443`, or `all` (literal `all` = `1-65535`) |
| `source_cidr` | String | Source CIDR block (ingress rules only). Example: `0.0.0.0/0`, `10.0.0.0/24` |
| `destination_cidr` | String | Destination CIDR block (egress rules only). Example: `0.0.0.0/0`, `192.168.1.0/24` |

## Terraform HCL Examples

### Query by ID

```hcl
data "zillaforge_security_groups" "specific" {
  id = "sg-12345678-1234-1234-1234-123456789abc"
}

output "security_group_name" {
  value = data.zillaforge_security_groups.specific.security_groups[0].name
}

output "ingress_rules" {
  value = data.zillaforge_security_groups.specific.security_groups[0].ingress_rules
}
```

### Query by Name

```hcl
data "zillaforge_security_groups" "web" {
  name = "web-servers-prod"
}

# Access first matching security group
output "web_sg_id" {
  value = length(data.zillaforge_security_groups.web.security_groups) > 0 ? data.zillaforge_security_groups.web.security_groups[0].id : null
}

# Check if security group exists
output "exists" {
  value = length(data.zillaforge_security_groups.web.security_groups) > 0
}
```

### List All Security Groups

```hcl
data "zillaforge_security_groups" "all" {
  # No filters = list all
}

output "all_security_group_names" {
  value = [for sg in data.zillaforge_security_groups.all.security_groups : sg.name]
}

output "total_count" {
  value = length(data.zillaforge_security_groups.all.security_groups)
}
```

### Use in Resource Configuration

```hcl
# Query existing security group
data "zillaforge_security_groups" "shared_web" {
  name = "shared-web-servers"
}

# Reference in instance configuration (future capability)
resource "zillaforge_vps_instance" "app" {
  name = "app-server-1"
  # ... other attributes ...
  
  # Attach queried security group to instance
  security_group_ids = [
    data.zillaforge_security_groups.shared_web.security_groups[0].id
  ]
}
```

## Filter Behavior

### ID Filter Specified

**API Call**: `GET /api/v1/projects/{project_id}/vps/security-groups/{id}`

**Behavior**:
- Returns single security group in `security_groups` list
- Errors if security group not found (404)
- `name` filter must NOT be specified (mutually exclusive)

**Example Response**:
```hcl
security_groups = [
  {
    id = "sg-12345678-1234-1234-1234-123456789abc"
    name = "web-servers-prod"
    description = "Production web tier"
    ingress_rules = [
      {
        protocol = "tcp"
        port_range = "443"
        source_cidr = "0.0.0.0/0"
      }
    ]
    egress_rules = [...]
  }
]
```

### Name Filter Specified

**API Call**: `GET /api/v1/projects/{project_id}/vps/security-groups?name={name}`

**Behavior**:
- Returns all security groups with exact name match (case-sensitive)
- Returns empty list `[]` if no matches found (NOT an error)
- `id` filter must NOT be specified (mutually exclusive)
- Client-side filtering if API doesn't support name parameter

**Example Response** (no matches):
```hcl
security_groups = []
```

**Example Response** (one match):
```hcl
security_groups = [
  {
    id = "sg-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
    name = "web-servers"
    description = ""
    ingress_rules = []
    egress_rules = []
  }
]
```

### No Filters Specified

**API Call**: `GET /api/v1/projects/{project_id}/vps/security-groups`

**Behavior**:
- Returns all security groups in the project
- Empty list `[]` if no security groups exist
- Supports pagination if account has many security groups

**Example Response**:
```hcl
security_groups = [
  {
    id = "sg-11111111-1111-1111-1111-111111111111"
    name = "web-servers"
    description = "Web tier"
    ingress_rules = [...]
    egress_rules = [...]
  },
  {
    id = "sg-22222222-2222-2222-2222-222222222222"
    name = "db-servers"
    description = "Database tier"
    ingress_rules = [...]
    egress_rules = [...]
  },
  # ... more security groups ...
]
```

### Both Filters Specified (Error Case)

```hcl
data "zillaforge_security_groups" "invalid" {
  id   = "sg-12345678-1234-1234-1234-123456789abc"
  name = "web-servers"  # ERROR: mutually exclusive with id
}
```

**Error**:
```
Error: Conflicting Filter Attributes

  on main.tf line X:
   X: data "zillaforge_security_groups" "invalid" {

Attributes "id" and "name" are mutually exclusive. Specify only one filter,
or omit both to list all security groups.
```

## API Response Format

### Single Security Group (by ID)

```json
{
  "id": "sg-12345678-1234-1234-1234-123456789abc",
  "name": "web-servers-prod",
  "description": "Production web tier",
  "ingress_rules": [
    {
      "protocol": "tcp",
      "port_range": "80",
      "source_cidr": "0.0.0.0/0"
    },
    {
      "protocol": "tcp",
      "port_range": "443",
      "source_cidr": "0.0.0.0/0"
    }
  ],
  "egress_rules": [
    {
      "protocol": "any",
      "port_range": "all",
      "destination_cidr": "0.0.0.0/0"
    }
  ],
  "created_at": "2025-12-14T10:30:00Z",
  "updated_at": "2025-12-14T11:00:00Z"
}
```

### List of Security Groups

```json
{
  "security_groups": [
    {
      "id": "sg-11111111-1111-1111-1111-111111111111",
      "name": "web-servers",
      "description": "Web tier",
      "ingress_rules": [...],
      "egress_rules": [...],
      "created_at": "2025-12-14T09:00:00Z"
    },
    {
      "id": "sg-22222222-2222-2222-2222-222222222222",
      "name": "db-servers",
      "description": "Database tier",
      "ingress_rules": [...],
      "egress_rules": [...],
      "created_at": "2025-12-14T09:15:00Z"
    }
  ],
  "pagination": {
    "total": 25,
    "page": 1,
    "per_page": 20,
    "next_page": 2
  }
}
```

**Pagination Handling**:
- If API supports pagination, iterate through all pages
- Consolidate results into single `security_groups` list
- Users don't see pagination details (abstracted by provider)

## Error Handling

### Security Group Not Found (ID filter)

**Scenario**: User specifies `id` for non-existent security group

**Error**:
```
Error: Security Group Not Found

  on main.tf line X:
   X: data "zillaforge_security_groups" "missing" {

Security group with ID "sg-99999999-9999-9999-9999-999999999999" was not
found. Verify the ID is correct and the security group exists in this project.
```

**Terraform Behavior**: Data source read fails; no state created

### Security Group Not Found (Name filter)

**Scenario**: User specifies `name` with no matches

**Behavior**: Returns empty list `security_groups = []` (NOT an error)

**Rationale**: Allows conditional logic like:
```hcl
locals {
  sg_exists = length(data.zillaforge_security_groups.optional.security_groups) > 0
}
```

### Multiple Filters Specified

**Scenario**: User provides both `id` and `name`

**Error**: Validation error during plan phase (before API call)

**Implementation**: Schema-level validation with `ConflictsWith`

```go
"id": schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        stringvalidator.ConflictsWith(path.Expressions{
            path.MatchRoot("name"),
        }...),
    },
},
"name": schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        stringvalidator.ConflictsWith(path.Expressions{
            path.MatchRoot("id"),
        }...),
    },
},
```

### API Rate Limiting

**Scenario**: Too many List requests in short time

**Behavior**: 
- Cloud-SDK automatically retries with exponential backoff
- Provider respects context timeout
- Returns error if retries exhausted

**Error**:
```
Error: Rate Limit Exceeded

Unable to list security groups: API rate limit exceeded. Please wait and try
again. The provider will automatically retry transient failures.
```

## Implementation Patterns

### Filter Logic

```go
func (d *SecurityGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data SecurityGroupDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    
    vpsClient := d.client.VPS()
    sgClient := vpsClient.SecurityGroups()
    
    // Filter by ID (single result)
    if !data.ID.IsNull() {
        sg, err := sgClient.Get(ctx, data.ID.ValueString())
        if err != nil {
            resp.Diagnostics.AddError(
                "Security Group Not Found",
                fmt.Sprintf("ID %s not found: %s", data.ID.ValueString(), err),
            )
            return
        }
        
        data.SecurityGroups = []SecurityGroupModel{convertToModel(*sg)}
        resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
        return
    }
    
    // List all, with optional name filter
    opts := &securitygroupsmodels.ListOptions{}
    if !data.Name.IsNull() {
        opts.Name = data.Name.ValueString()
    }
    
    securityGroups, err := sgClient.List(ctx, opts)
    if err != nil {
        resp.Diagnostics.AddError(
            "List Error",
            fmt.Sprintf("Failed to list security groups: %s", err),
        )
        return
    }
    
    // Client-side name filtering if API doesn't support it
    results := []SecurityGroupModel{}
    for _, sg := range securityGroups {
        if !data.Name.IsNull() && sg.Name != data.Name.ValueString() {
            continue
        }
        results = append(results, convertToModel(sg))
    }
    
    data.SecurityGroups = results
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

## Performance Considerations

### Caching

- Data sources are read during every `terraform plan` and `terraform apply`
- No built-in caching; API called each time
- For large lists (100+ security groups), ensure API response time meets SC-002 (< 5 seconds)

### Pagination

- Must handle paginated responses to avoid truncated results
- Cloud-SDK should abstract pagination details
- Provider consolidates all pages into single list

### Filtering Efficiency

- Prefer API-side filtering (`?name=xyz` query parameter) over client-side
- Client-side filtering acceptable if API doesn't support it
- Large result sets may impact Terraform performance

## Examples with Output

### Check if Security Group Exists

```hcl
data "zillaforge_security_groups" "check" {
  name = "optional-sg"
}

locals {
  sg_exists = length(data.zillaforge_security_groups.check.security_groups) > 0
  sg_id     = local.sg_exists ? data.zillaforge_security_groups.check.security_groups[0].id : null
}

output "security_group_status" {
  value = local.sg_exists ? "exists" : "not found"
}
```

### Find Security Groups with Specific Rules

```hcl
data "zillaforge_security_groups" "all" {}

locals {
  # Find security groups allowing SSH
  ssh_groups = [
    for sg in data.zillaforge_security_groups.all.security_groups :
    sg if anytrue([
      for rule in sg.ingress_rules :
      rule.protocol == "tcp" && rule.port_range == "22"
    ])
  ]
}

output "ssh_enabled_groups" {
  value = [for sg in local.ssh_groups : sg.name]
}
```

### List All Security Group Names

```hcl
data "zillaforge_security_groups" "all" {}

output "all_security_group_names" {
  value = [
    for sg in data.zillaforge_security_groups.all.security_groups : {
      id   = sg.id
      name = sg.name
    }
  ]
}
```
