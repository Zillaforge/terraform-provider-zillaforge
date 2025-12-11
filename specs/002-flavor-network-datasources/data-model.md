# Data Model: Flavor and Network Data Sources

**Feature**: 002-flavor-network-datasources  
**Date**: 2025-12-11  
**Purpose**: Define Terraform schema and state model for data sources

## Entity Definitions

### Flavor

**Description**: Represents a compute instance size template (VM flavor) available in Zillaforge VPS service.

**Attributes**:

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| `id` | string | - | Yes | Unique flavor identifier (UUID format) |
| `name` | string | - | Yes | Human-readable flavor name (e.g., "m1.small", "c2.large") |
| `vcpus` | int64 | - | Yes | Number of virtual CPUs allocated to instances using this flavor |
| `memory` | int64 | - | Yes | Memory size in GB (converted from SDK MiB: value / 1024) |
| `disk` | int64 | - | Yes | Root disk size in GB (optional in SDK, null if not set) |
| `description` | string | - | Yes | Optional human-readable description of flavor characteristics |

**Validation Rules**:
- `id`: Non-empty string
- `name`: Non-empty string
- `vcpus`: Positive integer (>= 1)
- `memory`: Positive integer (>= 1 GB)
- `disk`: Non-negative integer (>= 0), null allowed
- `description`: Any string, empty string allowed

**State Representation**:
```hcl
{
  id          = "550e8400-e29b-41d4-a716-446655440000"
  name        = "m1.large"
  vcpus       = 4
  memory      = 8  # GB
  disk        = 80 # GB
  description = "General purpose 4 vCPU, 8GB RAM, 80GB disk"
}
```

---

### Network

**Description**: Represents a virtual network segment for connecting compute instances.

**Attributes**:

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| `id` | string | - | Yes | Unique network identifier (UUID format) |
| `name` | string | - | Yes | Human-readable network name (e.g., "private-network", "dmz") |
| `cidr` | string | - | Yes | CIDR block defining network address range (e.g., "10.0.0.0/24") |
| `status` | string | - | Yes | Network operational status (e.g., "ACTIVE", "BUILD", "DOWN", "ERROR") |
| `description` | string | - | Yes | Optional human-readable description of network purpose |

**Validation Rules**:
- `id`: Non-empty string
- `name`: Non-empty string
- `cidr`: Valid CIDR notation (e.g., "10.0.0.0/8", "192.168.1.0/24")
- `status`: Non-empty string (API-controlled values)
- `description`: Any string, empty string allowed

**State Representation**:
```hcl
{
  id          = "660e8400-e29b-41d4-a716-446655440001"
  name        = "private-network"
  cidr        = "10.0.1.0/24"
  status      = "ACTIVE"
  description = "Private network for application servers"
}
```

---

## Data Source Schemas

### zillaforge_flavors

**Purpose**: Query and filter available compute flavors

**Filter Attributes** (Optional):

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Exact match filter for flavor name (case-sensitive) |
| `vcpus` | int64 | Minimum number of virtual CPUs (filters flavors with vcpus >= value) |
| `memory` | int64 | Minimum memory in GB (filters flavors with memory >= value) |

**Result Attributes** (Computed):

| Attribute | Type | Description |
|-----------|------|-------------|
| `flavors` | list of objects | List of matching flavor objects (see Flavor entity above) |

**Filter Behavior**:
- All filters optional (omitted filters match all)
- Multiple filters use AND logic (all must match)
- Empty filters return all available flavors
- No matches return empty list (not error)
- Name filter: exact match, case-sensitive
- vcpus/memory filters: inclusive minimum (>= comparison)

**Example Usage**:
```hcl
# Get all flavors
data "zillaforge_flavors" "all" {}

# Filter by name (exact match)
data "zillaforge_flavors" "specific" {
  name = "m1.large"
}

# Filter by minimum resources
data "zillaforge_flavors" "compute_optimized" {
  vcpus  = 4
  memory = 8  # >= 8 GB
}

# Multiple filters (AND logic)
data "zillaforge_flavors" "high_mem" {
  vcpus  = 2
  memory = 16
}
```

**State Model**:
```go
type FlavorsDataSourceModel struct {
    // Filters (input)
    Name   types.String `tfsdk:"name"`
    VCPUs  types.Int64  `tfsdk:"vcpus"`
    Memory types.Int64  `tfsdk:"memory"`
    
    // Results (output)
    Flavors []FlavorModel `tfsdk:"flavors"`
}

type FlavorModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    VCPUs       types.Int64  `tfsdk:"vcpus"`
    Memory      types.Int64  `tfsdk:"memory"`
    Disk        types.Int64  `tfsdk:"disk"`
    Description types.String `tfsdk:"description"`
}
```

---

### zillaforge_networks

**Purpose**: Query and filter available networks

**Filter Attributes** (Optional):

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Exact match filter for network name (case-sensitive) |
| `status` | string | Exact match filter for network status (e.g., "ACTIVE") |

**Result Attributes** (Computed):

| Attribute | Type | Description |
|-----------|------|-------------|
| `networks` | list of objects | List of matching network objects (see Network entity above) |

**Filter Behavior**:
- All filters optional (omitted filters match all)
- Multiple filters use AND logic (all must match)
- Empty filters return all available networks
- No matches return empty list (not error)
- All filters: exact match, case-sensitive

**Example Usage**:
```hcl
# Get all networks
data "zillaforge_networks" "all" {}

# Filter by name (exact match)
data "zillaforge_networks" "private" {
  name = "private-network"
}

# Filter by status
data "zillaforge_networks" "active" {
  status = "ACTIVE"
}

# Multiple filters (AND logic)
data "zillaforge_networks" "specific" {
  name   = "dmz"
  status = "ACTIVE"
}
```

**State Model**:
```go
type NetworksDataSourceModel struct {
    // Filters (input)
    Name   types.String `tfsdk:"name"`
    CIDR   types.String `tfsdk:"cidr"`
    Status types.String `tfsdk:"status"`
    
    // Results (output)
    Networks []NetworkModel `tfsdk:"networks"`
}

type NetworkModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    CIDR        types.String `tfsdk:"cidr"`
    Status      types.String `tfsdk:"status"`
    Description types.String `tfsdk:"description"`
}
```

---

## Data Transformations

### SDK to Terraform Type Conversions

**Flavor**:
```go
func sdkFlavorToModel(sdkFlavor *flavors.Flavor) FlavorModel {
    return FlavorModel{
        ID:          types.StringValue(sdkFlavor.ID),
        Name:        types.StringValue(sdkFlavor.Name),
        VCPUs:       types.Int64Value(int64(sdkFlavor.VCPU)),
        Memory:      types.Int64Value(int64(sdkFlavor.Memory / 1024)), // MiB → GB
        Disk:        types.Int64Value(int64(sdkFlavor.Disk)),          // Already GB
        Description: types.StringValue(sdkFlavor.Description),
    }
}
```

**Memory Conversion**:
- SDK: `Memory` field in MiB (mebibytes: 1 MiB = 1024 KiB = 1,048,576 bytes)
- Terraform: `memory` attribute in GB (gigabytes: 1 GB = 1024 MB)
- Conversion: `memory_gb = memory_mib / 1024`
- Example: 8192 MiB → 8 GB

**Network**:
```go
func sdkNetworkToModel(sdkNetwork *networks.Network) NetworkModel {
    return NetworkModel{
        ID:          types.StringValue(sdkNetwork.ID),
        Name:        types.StringValue(sdkNetwork.Name),
        CIDR:        types.StringValue(sdkNetwork.CIDR),
        Status:      types.StringValue(sdkNetwork.Status),
        Description: types.StringValue(sdkNetwork.Description),
    }
}
```

**Null Handling**:
```go
// Optional fields that may be empty/null in SDK
if sdkFlavor.Description == "" {
    model.Description = types.StringNull()
} else {
    model.Description = types.StringValue(sdkFlavor.Description)
}

if sdkFlavor.Disk == 0 {
    model.Disk = types.Int64Null()
} else {
    model.Disk = types.Int64Value(int64(sdkFlavor.Disk))
}
```

---

## Filtering Implementation

### Client-Side Filter Logic

**Flavor Filtering**:
```go
func filterFlavors(flavors []*flavors.Flavor, filters FlavorsDataSourceModel) []FlavorModel {
    var results []FlavorModel
    
    for _, flavor := range flavors {
        // Exact name match (case-sensitive)
        if !filters.Name.IsNull() && flavor.Name != filters.Name.ValueString() {
            continue
        }
        
        // Minimum vcpus
        if !filters.VCPUs.IsNull() && int64(flavor.VCPU) < filters.VCPUs.ValueInt64() {
            continue
        }
        
        // Minimum memory (convert SDK MiB to GB)
        if !filters.Memory.IsNull() {
            memoryGB := int64(flavor.Memory / 1024)
            if memoryGB < filters.Memory.ValueInt64() {
                continue
            }
        }
        
        // All filters passed, include in results
        results = append(results, sdkFlavorToModel(flavor))
    }
    
    return results
}
```

**Network Filtering**:
```go
func filterNetworks(networks []*networks.Network, filters NetworksDataSourceModel) []NetworkModel {
    var results []NetworkModel
    
    for _, network := range networks {
        // Exact name match
        if !filters.Name.IsNull() && network.Name != filters.Name.ValueString() {
            continue
        }
        
        // Exact status match
        if !filters.Status.IsNull() && network.Status != filters.Status.ValueString() {
            continue
        }
        
        // All filters passed, include in results
        results = append(results, sdkNetworkToModel(network))
    }
    
    return results
}
```

**Filter Semantics**:
- Null filter values ignored (no filtering applied)
- Non-null filters checked with AND logic (short-circuit on first mismatch)
- String comparisons are case-sensitive exact matches
- Numeric comparisons use inclusive minimum (>=)
- Empty result set returns empty list (not error)

---

## Relationships

### Usage in Resource Configurations

**Flavor Reference Pattern**:
```hcl
data "zillaforge_flavors" "compute" {
  vcpus  = 4
  memory = 8
}

resource "zillaforge_instance" "web" {
  name     = "web-server"
  flavor_id = data.zillaforge_flavors.compute.flavors[0].id
  # ...
}
```

**Network Reference Pattern**:
```hcl
data "zillaforge_networks" "private" {
  name = "private-network"
}

resource "zillaforge_instance" "app" {
  name       = "app-server"
  network_id = data.zillaforge_networks.private.networks[0].id
  # ...
}
```

**Multi-Match Handling**:
```hcl
# Get all active networks
data "zillaforge_networks" "active" {
  status = "ACTIVE"
}

# Loop over results
resource "zillaforge_port" "app_ports" {
  count      = length(data.zillaforge_networks.active.networks)
  network_id = data.zillaforge_networks.active.networks[count.index].id
  # ...
}
```

---

## State Management

### Read Operation Flow

1. **Parse filter inputs** from Terraform configuration
2. **Validate filters** (type checking handled by framework)
3. **Query SDK** via `List()` method (single API call)
4. **Apply client-side filters** using AND logic
5. **Convert SDK types** to Terraform models
6. **Set state** with filtered results list

### State Updates

- Data sources are **read-only** (no Create, Update, Delete)
- State refreshed on every `terraform plan` or `terraform apply`
- No incremental updates (full refresh pattern)
- Empty result lists preserved in state (not treated as error)

### Idempotency

- Same filter inputs always return same results (barring API changes)
- No side effects (read-only operations)
- Safe to run multiple times
- No resource creation or modification
