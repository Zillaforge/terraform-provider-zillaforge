# Schema Contract: zillaforge_flavors Data Source

**Version**: 1.0.0  
**Data Source**: `zillaforge_flavors`  
**Purpose**: Query available compute flavors with optional filtering

## Schema Definition

```go
func (d *FlavorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Query available compute flavors in the Zillaforge VPS service. " +
            "Flavors define instance sizes with specific CPU, memory, and disk allocations. " +
            "Use filters to narrow results by name or minimum resource requirements.",
        
        Attributes: map[string]schema.Attribute{
            // Filter: Name (exact match)
            "name": schema.StringAttribute{
                MarkdownDescription: "Filter flavors by exact name match (case-sensitive). " +
                    "Example: `m1.large` will match only flavors with that exact name.",
                Optional: true,
            },
            
            // Filter: Minimum vCPUs
            "vcpus": schema.Int64Attribute{
                MarkdownDescription: "Filter flavors with at least this many virtual CPUs. " +
                    "Example: `vcpus = 4` returns flavors with 4 or more vCPUs.",
                Optional: true,
            },
            
            // Filter: Minimum Memory (GB)
            "memory": schema.Int64Attribute{
                MarkdownDescription: "Filter flavors with at least this much memory in GB. " +
                    "Example: `memory = 8` returns flavors with 8GB or more RAM.",
                Optional: true,
            },
            
            // Result: Flavors List
            "flavors": schema.ListNestedAttribute{
                MarkdownDescription: "List of flavors matching the filter criteria. " +
                    "Returns empty list if no matches found.",
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                            MarkdownDescription: "Unique identifier for this flavor (UUID format).",
                            Computed:            true,
                        },
                        "name": schema.StringAttribute{
                            MarkdownDescription: "Human-readable flavor name (e.g., `m1.small`, `c2.large`).",
                            Computed:            true,
                        },
                        "vcpus": schema.Int64Attribute{
                            MarkdownDescription: "Number of virtual CPUs allocated to instances using this flavor.",
                            Computed:            true,
                        },
                        "memory": schema.Int64Attribute{
                            MarkdownDescription: "Memory size in GB allocated to instances using this flavor.",
                            Computed:            true,
                        },
                        "disk": schema.Int64Attribute{
                            MarkdownDescription: "Root disk size in GB. May be null if flavor uses separate volumes.",
                            Computed:            true,
                        },
                        "description": schema.StringAttribute{
                            MarkdownDescription: "Optional human-readable description of flavor characteristics.",
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
| `vcpus` | int64 | Optional, >= 1 | null | Minimum CPU count (inclusive) |
| `memory` | int64 | Optional, >= 1 | null | Minimum memory in GB (inclusive) |

**Filter Behavior**:
- All filters are optional
- Null/omitted filters match all values
- Multiple filters use AND logic (all must match)
- No filters returns all available flavors

**Validation Rules**:
- `name`: Any non-empty string accepted
- `vcpus`: Must be positive integer if provided
- `memory`: Must be positive integer if provided

### Result Attributes (Output)

| Attribute | Type | Constraints | Nullable | Description |
|-----------|------|-------------|----------|-------------|
| `flavors` | list | Computed | No | List of matching flavor objects |
| `flavors[].id` | string | Computed | No | UUID identifier |
| `flavors[].name` | string | Computed | No | Flavor name |
| `flavors[].vcpus` | int64 | Computed, >= 1 | No | Virtual CPU count |
| `flavors[].memory` | int64 | Computed, >= 1 | No | Memory in GB |
| `flavors[].disk` | int64 | Computed, >= 0 | Yes | Disk size in GB (null if not set) |
| `flavors[].description` | string | Computed | Yes | Optional description |

**Result Guarantees**:
- `flavors` list never null (empty list if no matches)
- Each flavor object has non-null `id`, `name`, `vcpus`, `memory`
- `disk` and `description` may be null/empty

## Examples

### Basic Usage

```hcl
# Get all available flavors
data "zillaforge_flavors" "all" {}

# Output: List of all flavors
output "flavor_count" {
  value = length(data.zillaforge_flavors.all.flavors)
}
```

### Filter by Name

```hcl
# Get specific flavor by exact name
data "zillaforge_flavors" "large" {
  name = "m1.large"
}

# Use in resource configuration
resource "zillaforge_instance" "web" {
  name     = "web-server"
  flavor_id = data.zillaforge_flavors.large.flavors[0].id
}
```

### Filter by Minimum Resources

```hcl
# Get flavors with at least 4 vCPUs
data "zillaforge_flavors" "compute" {
  vcpus = 4
}

# Get flavors with at least 16GB memory
data "zillaforge_flavors" "high_mem" {
  memory = 16
}

# Combine filters (AND logic)
data "zillaforge_flavors" "powerful" {
  vcpus  = 8
  memory = 32
}
```

### Access Specific Attributes

```hcl
data "zillaforge_flavors" "compute" {
  vcpus = 4
}

# Access first matching flavor
output "flavor_details" {
  value = {
    id     = data.zillaforge_flavors.compute.flavors[0].id
    name   = data.zillaforge_flavors.compute.flavors[0].name
    vcpus  = data.zillaforge_flavors.compute.flavors[0].vcpus
    memory = data.zillaforge_flavors.compute.flavors[0].memory
  }
}
```

### Loop Over Results

```hcl
# Get all flavors with at least 2 vCPUs
data "zillaforge_flavors" "compute_optimized" {
  vcpus = 2
}

# Display all matching flavors
output "available_flavors" {
  value = [
    for flavor in data.zillaforge_flavors.compute_optimized.flavors : {
      name   = flavor.name
      vcpus  = flavor.vcpus
      memory = flavor.memory
      disk   = flavor.disk
    }
  ]
}
```

## Error Scenarios

### Empty Result Set

```hcl
# Filter that matches no flavors
data "zillaforge_flavors" "nonexistent" {
  name = "does-not-exist"
}

# Result: flavors = [] (empty list, not error)
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

Failed to list flavors: Service temporarily unavailable (HTTP 503)
```

## State Representation

```json
{
  "name": {
    "value": "m1.large",
    "type": "string"
  },
  "vcpus": {
    "value": null,
    "type": "number"
  },
  "memory": {
    "value": null,
    "type": "number"
  },
  "flavors": {
    "value": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "m1.large",
        "vcpus": 4,
        "memory": 8,
        "disk": 80,
        "description": "General purpose 4 vCPU, 8GB RAM"
      }
    ],
    "type": [
      "list",
      [
        "object",
        {
          "id": "string",
          "name": "string",
          "vcpus": "number",
          "memory": "number",
          "disk": "number",
          "description": "string"
        }
      ]
    ]
  }
}
```

## Acceptance Test Requirements

### Test Cases

1. **No filters** - Returns all flavors
2. **Name filter (exact match)** - Returns only matching flavor
3. **Name filter (no match)** - Returns empty list
4. **vCPUs filter** - Returns flavors with vcpus >= specified
5. **Memory filter** - Returns flavors with memory >= specified
6. **Multiple filters (AND logic)** - Returns flavors matching all criteria
7. **Invalid auth** - Returns authentication error diagnostic
8. **API error** - Returns actionable error diagnostic

### Test Structure

```go
func TestAccFlavorsDataSource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccFlavorsDataSourceConfig_all,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.#"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.0.id"),
                    resource.TestCheckResourceAttrSet("data.zillaforge_flavors.test", "flavors.0.name"),
                ),
            },
        },
    })
}
```

## Versioning

**Current Version**: 1.0.0

**Breaking Changes** (require major version bump):
- Removing attributes
- Changing attribute types
- Changing filter semantics (exact → partial match)
- Changing AND/OR filter logic

**Non-Breaking Changes** (allow minor version bump):
- Adding new optional filter attributes
- Adding new computed result attributes
- Improving error messages
