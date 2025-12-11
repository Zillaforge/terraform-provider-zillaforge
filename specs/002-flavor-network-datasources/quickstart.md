# Quick Start Guide: Flavor and Network Data Sources

**Feature**: 002-flavor-network-datasources  
**Audience**: Terraform users and developers  
**Purpose**: Get started with querying flavors and networks in Zillaforge provider

## Overview

The Zillaforge provider includes two data sources for discovering available infrastructure resources:

- **`zillaforge_flavors`** - Query compute instance sizes (CPU, memory, disk specs)
- **`zillaforge_networks`** - Query virtual networks (name, status, connectivity)

Both data sources support optional filtering and return lists of matching resources.

## Prerequisites

1. **Zillaforge Provider Configured**:
   ```hcl
   terraform {
     required_providers {
       zillaforge = {
         source = "Zillaforge/zillaforge"
       }
     }
   }

   provider "zillaforge" {
     api_endpoint = "https://api.zillaforge.com"
     api_key      = var.zillaforge_api_key
     project_id   = "your-project-id"
   }
   ```

2. **Valid API Credentials**: Obtain API key from Zillaforge console
3. **Project Access**: Ensure API key has permissions to list VPS resources

## 5-Minute Quick Start

### Step 1: Query All Flavors

Create `flavors.tf`:

```hcl
data "zillaforge_flavors" "all" {}

output "available_flavors" {
  value = [
    for flavor in data.zillaforge_flavors.all.flavors : {
      name   = flavor.name
      vcpus  = flavor.vcpus
      memory = flavor.memory
      disk   = flavor.disk
    }
  ]
}
```

Run:
```bash
terraform init
terraform plan
```

**Expected Output**:
```
available_flavors = [
  {
    name   = "m1.small"
    vcpus  = 1
    memory = 2
    disk   = 20
  },
  {
    name   = "m1.medium"
    vcpus  = 2
    memory = 4
    disk   = 40
  },
  # ... more flavors
]
```

---

### Step 2: Query All Networks

Create `networks.tf`:

```hcl
data "zillaforge_networks" "all" {}

output "available_networks" {
  value = [
    for network in data.zillaforge_networks.all.networks : {
      name   = network.name
      cidr   = network.cidr
      status = network.status
    }
  ]
}
```

Run:
```bash
terraform plan
```

**Expected Output**:
```
available_networks = [
  {
    name   = "default-network"
    cidr   = "10.0.0.0/16"
    status = "ACTIVE"
  },
  {
    name   = "private-network"
    cidr   = "10.1.0.0/24"
    status = "ACTIVE"
  },
]
```

---

### Step 3: Use Data Sources in Resources

Create `instance.tf`:

```hcl
# Find suitable flavor (at least 2 vCPUs, 4GB RAM)
data "zillaforge_flavors" "compute" {
  vcpus  = 2
  memory = 4
}

# Find private network
data "zillaforge_networks" "private" {
  name = "private-network"
}

# Create instance using discovered resources
resource "zillaforge_instance" "web_server" {
  name     = "web-server-01"
  flavor_id = data.zillaforge_flavors.compute.flavors[0].id
  network_id = data.zillaforge_networks.private.networks[0].id
  
  # ... other configuration
}
```

Run:
```bash
terraform plan
```

**Terraform will**:
1. Query flavors with >= 2 vCPUs and >= 4GB memory
2. Query networks named "private-network"
3. Use the first matching flavor and network to configure the instance

---

## Common Usage Patterns

### Pattern 1: Find Specific Flavor by Name

```hcl
data "zillaforge_flavors" "large" {
  name = "m1.large"
}

# Access flavor attributes
output "large_flavor_id" {
  value = data.zillaforge_flavors.large.flavors[0].id
}

output "large_flavor_specs" {
  value = "${data.zillaforge_flavors.large.flavors[0].vcpus} vCPUs, ${data.zillaforge_flavors.large.flavors[0].memory}GB RAM"
}
```

---

### Pattern 2: Filter Flavors by Minimum Resources

```hcl
# High-memory workloads
data "zillaforge_flavors" "high_mem" {
  memory = 16  # At least 16GB
}

# Compute-intensive workloads
data "zillaforge_flavors" "cpu_intensive" {
  vcpus = 8  # At least 8 vCPUs
}

# Balanced workloads
data "zillaforge_flavors" "balanced" {
  vcpus  = 4
  memory = 16
}
```

---

### Pattern 3: Find Active Networks

```hcl
data "zillaforge_networks" "active" {
  status = "ACTIVE"
}

# Create ports on all active networks
resource "zillaforge_port" "multi_nic" {
  count      = length(data.zillaforge_networks.active.networks)
  network_id = data.zillaforge_networks.active.networks[count.index].id
}
```

---

### Pattern 4: Conditional Resource Creation

```hcl
# Check if GPU flavors exist
data "zillaforge_flavors" "gpu" {
  name = "g1.large"
}

# Create GPU instance only if flavor exists
resource "zillaforge_instance" "gpu_worker" {
  count = length(data.zillaforge_flavors.gpu.flavors) > 0 ? 1 : 0
  
  name     = "gpu-worker"
  flavor_id = data.zillaforge_flavors.gpu.flavors[0].id
}
```

---

### Pattern 5: Display Available Options to Users

```hcl
data "zillaforge_flavors" "all" {}

output "flavor_menu" {
  description = "Available instance sizes"
  value = {
    for flavor in data.zillaforge_flavors.all.flavors :
    flavor.name => "${flavor.vcpus} vCPUs / ${flavor.memory}GB RAM / ${flavor.disk}GB Disk"
  }
}
```

**Output**:
```
flavor_menu = {
  "m1.small"  = "1 vCPUs / 2GB RAM / 20GB Disk"
  "m1.medium" = "2 vCPUs / 4GB RAM / 40GB Disk"
  "m1.large"  = "4 vCPUs / 8GB RAM / 80GB Disk"
  # ...
}
```

---

## Troubleshooting

### No Results Returned

**Problem**: Data source returns empty list

```hcl
data "zillaforge_flavors" "test" {
  name = "nonexistent-flavor"
}

# Error when accessing [0]:
# Error: Invalid index - list has 0 elements
```

**Solutions**:
1. Check filter values match exactly (case-sensitive)
2. Query without filters to see all available options
3. Add validation:
   ```hcl
   locals {
     flavor_id = length(data.zillaforge_flavors.test.flavors) > 0 ? 
                 data.zillaforge_flavors.test.flavors[0].id : null
   }
   ```

---

### Authentication Errors

**Problem**: "Authentication Failed" error

```
Error: Authentication Failed

Check api_key configuration and ensure token is valid.
```

**Solutions**:
1. Verify `api_key` in provider configuration
2. Check API key hasn't expired in Zillaforge console
3. Ensure API key has project access permissions

---

### Multiple Matches When Expecting One

**Problem**: Filter returns multiple results, unsure which to use

```hcl
data "zillaforge_flavors" "compute" {
  vcpus = 4  # Returns multiple 4-vCPU flavors
}

resource "zillaforge_instance" "app" {
  flavor_id = data.zillaforge_flavors.compute.flavors[0].id  # Which one?
}
```

**Solutions**:
1. Use exact name match for specific flavor:
   ```hcl
   data "zillaforge_flavors" "specific" {
     name = "m1.large"  # Exact match
   }
   ```

2. Add more filters to narrow results:
   ```hcl
   data "zillaforge_flavors" "specific" {
     vcpus  = 4
     memory = 8  # Now more specific
   }
   ```

3. Use `for` expression to choose:
   ```hcl
   locals {
     # Choose smallest matching flavor
     flavor_id = sort([
       for f in data.zillaforge_flavors.compute.flavors : f.id
     ])[0]
   }
   ```

---

## Best Practices

### 1. Use Exact Name Filters for Production

```hcl
# ✅ Good: Explicit, predictable
data "zillaforge_flavors" "prod" {
  name = "m1.large"
}

# ⚠️ Risky: May return different flavor if new ones added
data "zillaforge_flavors" "prod" {
  vcpus = 4
}
```

### 2. Validate Data Source Results

```hcl
data "zillaforge_networks" "required" {
  name = "production-network"
}

# Add validation
resource "null_resource" "validate" {
  lifecycle {
    precondition {
      condition     = length(data.zillaforge_networks.required.networks) > 0
      error_message = "Production network not found. Ensure network exists before deploying."
    }
  }
}
```

### 3. Use Locals for Reusable Selections

```hcl
data "zillaforge_flavors" "compute" {
  vcpus  = 4
  memory = 8
}

data "zillaforge_networks" "private" {
  name   = "private-network"
  status = "ACTIVE"
}

locals {
  standard_flavor_id  = data.zillaforge_flavors.compute.flavors[0].id
  private_network_id = data.zillaforge_networks.private.networks[0].id
}

# Reuse in multiple resources
resource "zillaforge_instance" "app1" {
  name       = "app-server-1"
  flavor_id  = local.standard_flavor_id
  network_id = local.private_network_id
}

resource "zillaforge_instance" "app2" {
  name       = "app-server-2"
  flavor_id  = local.standard_flavor_id
  network_id = local.private_network_id
}
```

### 4. Document Expected Results

```hcl
# Expected: 1 result (production network)
data "zillaforge_networks" "prod" {
  name = "production-network"
}

# Expected: 3-5 results (all compute flavors)
data "zillaforge_flavors" "compute_family" {
  vcpus = 2
}
```

---

## Next Steps

1. **Review Full Schema**: See [contracts/flavors-schema.md](./contracts/flavors-schema.md) and [contracts/networks-schema.md](./contracts/networks-schema.md)
2. **Explore Filtering**: Experiment with different filter combinations
3. **Integrate with Resources**: Use data sources in instance, port, and security group configurations
4. **Monitor Changes**: Run `terraform plan` regularly to detect infrastructure changes

---

## Additional Resources

- [Terraform Data Sources Documentation](https://www.terraform.io/language/data-sources)
- [Zillaforge Provider Documentation](../../../docs/)
- [Feature Specification](./spec.md)
- [Data Model Reference](./data-model.md)
