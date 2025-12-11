# Query all available flavors (no filters)
data "zillaforge_flavors" "all" {}

output "all_flavors" {
  value = data.zillaforge_flavors.all.flavors
}

# Filter by exact name
data "zillaforge_flavors" "by_name" {
  name = "m1.large"
}

output "specific_flavor" {
  value = length(data.zillaforge_flavors.by_name.flavors) > 0 ? data.zillaforge_flavors.by_name.flavors[0] : null
}

# Filter by minimum vCPUs
data "zillaforge_flavors" "min_cpu" {
  vcpus = 4
}

output "high_cpu_flavors" {
  value = data.zillaforge_flavors.min_cpu.flavors
}

# Filter by minimum memory (GB)
data "zillaforge_flavors" "min_memory" {
  memory = 8
}

output "high_memory_flavors" {
  value = data.zillaforge_flavors.min_memory.flavors
}

# Multiple filters with AND logic
data "zillaforge_flavors" "specific_requirements" {
  vcpus  = 2
  memory = 4
}

output "matched_flavors" {
  value = data.zillaforge_flavors.specific_requirements.flavors
}

# Use flavor in resource configuration (example integration)
data "zillaforge_flavors" "compute" {
  vcpus  = 2
  memory = 4
}

# Example: Reference flavor ID in a resource
# resource "zillaforge_instance" "app_server" {
#   flavor_id = data.zillaforge_flavors.compute.flavors[0].id
#   name      = "app-server-01"
#   # ... other configuration
# }
