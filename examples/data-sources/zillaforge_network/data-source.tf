# Query all available networks (no filters)
data "zillaforge_networks" "all" {}

output "all_networks" {
  value = data.zillaforge_networks.all.networks
}

# Filter by exact name
data "zillaforge_networks" "by_name" {
  name = "private-network"
}

output "specific_network" {
  value = length(data.zillaforge_networks.by_name.networks) > 0 ? data.zillaforge_networks.by_name.networks[0] : null
}

# Filter by status
data "zillaforge_networks" "active" {
  status = "ACTIVE"
}

output "active_networks" {
  value = data.zillaforge_networks.active.networks
}

# Multiple filters with AND logic
data "zillaforge_networks" "dmz_active" {
  name   = "dmz"
  status = "ACTIVE"
}

output "dmz_network" {
  value = length(data.zillaforge_networks.dmz_active.networks) > 0 ? data.zillaforge_networks.dmz_active.networks[0] : null
}

# Use network in resource configuration (example integration)
data "zillaforge_networks" "app_network" {
  name   = "app-private-network"
  status = "ACTIVE"
}

# Example: Reference network ID in a resource
# resource "zillaforge_instance" "app_server" {
#   name      = "app-server-01"
#   flavor_id = "some-flavor-id"
#   network_ids = [
#     data.zillaforge_networks.app_network.networks[0].id
#   ]
#   # ... other configuration
# }

# Access network CIDR for security group rules
output "network_cidr" {
  value       = length(data.zillaforge_networks.app_network.networks) > 0 ? data.zillaforge_networks.app_network.networks[0].cidr : null
  description = "CIDR block of the application network"
}
