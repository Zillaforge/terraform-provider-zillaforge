# Basic Floating IP Allocation
# This example shows how to allocate a floating IP address from the shared pool.

resource "zillaforge_floating_ip" "basic" {
  # No additional attributes - allocates with default settings
  # The IP address will be automatically assigned from the pool
}

# Output the allocated IP address
output "floating_ip_address" {
  description = "The allocated floating IP address"
  value       = zillaforge_floating_ip.basic.ip_address
}

output "floating_ip_id" {
  description = "The unique ID of the floating IP"
  value       = zillaforge_floating_ip.basic.id
}

# Floating IP with Name and Description
# This example shows how to allocate a floating IP with metadata.

resource "zillaforge_floating_ip" "with_metadata" {
  name        = "web-server-public-ip"
  description = "Public IP address for the main web server"
}

# Output the metadata
output "floating_ip_with_metadata" {
  description = "Details of the floating IP with metadata"
  value = {
    id          = zillaforge_floating_ip.with_metadata.id
    name        = zillaforge_floating_ip.with_metadata.name
    description = zillaforge_floating_ip.with_metadata.description
    ip_address  = zillaforge_floating_ip.with_metadata.ip_address
    status      = zillaforge_floating_ip.with_metadata.status
  }
}