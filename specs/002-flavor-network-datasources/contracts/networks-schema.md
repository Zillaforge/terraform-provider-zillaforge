# Schema Contract: zillaforge_networks Data Source

**Version**: 1.0.0  
**Data Source**: `zillaforge_networks`  
**Purpose**: Query available networks with optional filtering

## Schema Definition

```go
func (d *NetworksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Query available networks in the Zillaforge VPS service. " +
            "Networks define virtual network segments for connecting compute instances. " +
            "Use filters to narrow results by name or status.",
        
        Attributes: map[string]schema.Attribute{
            // Filter: Name (exact match)
            "name": schema.StringAttribute{
                MarkdownDescription: "Filter networks by exact name match (case-sensitive). " +
                    "Example: `private-network` will match only networks with that exact name.",
                Optional: true,
            },
            
            // Filter: Status (exact match)
            "status": schema.StringAttribute{
                MarkdownDescription: "Filter networks by status (case-sensitive). " +
                    "Common values: `ACTIVE`, `BUILD`, `DOWN`, `ERROR`. " +
                    "Example: `status = \"ACTIVE\"` returns only active networks.",
                Optional: true,
            },
            
            // Result: Networks List
            "networks": schema.ListNestedAttribute{
                MarkdownDescription: "List of networks matching the filter criteria. " +
                    "Returns empty list if no matches found.",
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                            MarkdownDescription: "Unique identifier for this network (UUID format).",
                            Computed:            true,
                        },
                        "name": schema.StringAttribute{
                            MarkdownDescription: "Human-readable network name (e.g., `private-network`, `dmz`).",
                            Computed:            true,
                        },
                        "cidr": schema.StringAttribute{
                            MarkdownDescription: "CIDR block defining the network address range (e.g., `10.0.0.0/24`).",
                            Computed:            true,
                        },
                        "status": schema.StringAttribute{
                            MarkdownDescription: "Network operational status. " +
                                "Common values: `ACTIVE` (operational), `BUILD` (creating), " +
                                "`DOWN` (offline), `ERROR` (failed).",
                            Computed: true,
                        },
                        "description": schema.StringAttribute{
                            MarkdownDescription: "Optional human-readable description of network purpose.",
                            Computed:            true,
                        },
                    },
                },
            },
        },
    }
}
```

## Attribute Specifications

### Filter Attributes (Input)

| Attribute | Type | Constraints | Default | Validation |
|-----------|------|-------------|---------|------------|
| `name` | string | Optional | null | Case-sensitive exact match |
| `status` | string | Optional | null | Case-sensitive exact match |

**Filter Behavior**:
- All filters are optional
- Null/omitted filters match all values
- Multiple filters use AND logic (all must match)
- No filters returns all available networks

**Validation Rules**:
- `name`: Any non-empty string accepted
- `status`: Any string accepted (status values controlled by API)

### Result Attributes (Output)

| Attribute | Type | Constraints | Nullable | Description |
|-----------|------|-------------|----------|-------------|
| `networks` | list | Computed | No | List of matching network objects |
| `networks[].id` | string | Computed | No | UUID identifier |
| `networks[].name` | string | Computed | No | Network name |
| `networks[].cidr` | string | Computed | No | CIDR block |
| `networks[].status` | string | Computed | No | Operational status |
| `networks[].description` | string | Computed | Yes | Optional description |

**Result Guarantees**:
- `networks` list never null (empty list if no matches)
- Each network object has non-null `id`, `name`, `cidr`, `status`
- `description` may be null/empty

**Status Values** (API-controlled):
- `ACTIVE` - Network is operational and ready for use
- `BUILD` - Network is being created/configured
- `DOWN` - Network is offline or disabled
- `ERROR` - Network creation or operation failed

## Examples

### Basic Usage

```hcl
# Get all available networks
data "zillaforge_networks" "all" {}

# Output: List of all networks
output "network_count" {
  value = length(data.zillaforge_networks.all.networks)
}
```

### Filter by Name

```hcl
# Get specific network by exact name
data "zillaforge_networks" "private" {
  name = "private-network"
}

# Use in resource configuration
resource "zillaforge_instance" "app" {
  name       = "app-server"
  network_id = data.zillaforge_networks.private.networks[0].id
}
```

### Filter by Status

```hcl
# Get only active networks
data "zillaforge_networks" "active" {
  status = "ACTIVE"
}

# Loop over active networks
output "active_networks" {
  value = [
    for network in data.zillaforge_networks.active.networks : {
      name = network.name
      cidr = network.cidr
    }
  ]
}
```

### Multiple Filters (AND Logic)

```hcl
# Get active private network
data "zillaforge_networks" "private_active" {
  name   = "private-network"
  status = "ACTIVE"
}

# Verify network exists and is active
resource "zillaforge_instance" "secure_app" {
  name       = "secure-app"
  network_id = data.zillaforge_networks.private_active.networks[0].id
}
```

### Access Specific Attributes

```hcl
data "zillaforge_networks" "dmz" {
  name = "dmz"
}

# Display network details
output "dmz_details" {
  value = {
    id          = data.zillaforge_networks.dmz.networks[0].id
    name        = data.zillaforge_networks.dmz.networks[0].name
    cidr        = data.zillaforge_networks.dmz.networks[0].cidr
    status      = data.zillaforge_networks.dmz.networks[0].status
    description = data.zillaforge_networks.dmz.networks[0].description
  }
}
```

### Conditional Resource Creation

```hcl
# Get network if it exists
data "zillaforge_networks" "optional" {
  name = "optional-network"
}

# Create instance only if network exists
resource "zillaforge_instance" "conditional" {
  count = length(data.zillaforge_networks.optional.networks) > 0 ? 1 : 0
  
  name       = "conditional-instance"
  network_id = data.zillaforge_networks.optional.networks[0].id
}
```

## Error Scenarios

### Empty Result Set

```hcl
# Filter that matches no networks
data "zillaforge_networks" "nonexistent" {
  name = "does-not-exist"
}

# Result: networks = [] (empty list, not error)
# Accessing [0] will cause Terraform error:
# Error: Invalid index - list has 0 elements
```

### API Authentication Error

```
Error: Authentication Failed

Check api_key configuration and ensure token is valid.
Error: API returned 401 Unauthorized
```

### API Unavailable

```
Error: API Error

Failed to list networks: Service temporarily unavailable (HTTP 503)
```

### Permission Denied

```
Error: Permission Denied

Insufficient permissions to list networks for this project.
Verify project access and API key permissions.
```

## State Representation

```json
{
  "name": {
    "value": "private-network",
    "type": "string"
  },
  "status": {
    "value": "ACTIVE",
    "type": "string"
  },
  "networks": {
    "value": [
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "private-network",
        "cidr": "10.0.1.0/24",
        "status": "ACTIVE",
        "description": "Private network for application servers"
      }
    ],
    "type": [
      "list",
      [
        "object",
        {
          "id": "string",
          "name": "string",
          "cidr": "string",
          "status": "string",
          "description": "string"
        }
      ]
    ]
  }
}
```

## Acceptance Test Requirements

### Test Cases

1. **No filters** - Returns all networks
2. **Name filter (exact match)** - Returns only matching network
3. **Name filter (no match)** - Returns empty list
4. **Status filter** - Returns networks with matching status
5. **Multiple filters (AND logic)** - Returns networks matching all criteria
6. **Invalid auth** - Returns authentication error diagnostic
7. **API error** - Returns actionable error diagnostic

### Test Structure

```go
func TestAccNetworksDataSource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccNetworksDataSourceConfig_all,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.#"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.0.id"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.0.name"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.0.cidr"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_networks.test", "networks.0.status"),
                ),
            },
        },
    })
}
```

## Integration Patterns

### With Instance Resources

```hcl
# Find network by name
data "zillaforge_networks" "backend" {
  name = "backend-network"
}

# Attach instance to network
resource "zillaforge_instance" "database" {
  name       = "db-server"
  network_id = data.zillaforge_networks.backend.networks[0].id
  flavor_id  = "..."
}
```

### With Port/NIC Resources

```hcl
# Get all active networks
data "zillaforge_networks" "active" {
  status = "ACTIVE"
}

# Create port on each network
resource "zillaforge_port" "multi_nic" {
  count      = length(data.zillaforge_networks.active.networks)
  network_id = data.zillaforge_networks.active.networks[count.index].id
  # ...
}
```

### With Security Groups

```hcl
# Get DMZ network
data "zillaforge_networks" "dmz" {
  name = "dmz"
  status = "ACTIVE"
}

# Create security group for DMZ
resource "zillaforge_security_group" "dmz_sg" {
  name       = "dmz-sg"
  network_id = data.zillaforge_networks.dmz.networks[0].id
  # ...
}
```

## Versioning

**Current Version**: 1.0.0

**Breaking Changes** (require major version bump):
- Removing attributes
- Changing attribute types
- Changing filter semantics (exact → partial match)
- Changing AND/OR filter logic
- Removing status values

**Non-Breaking Changes** (allow minor version bump):
- Adding new optional filter attributes
- Adding new computed result attributes
- Adding new status values
- Improving error messages
