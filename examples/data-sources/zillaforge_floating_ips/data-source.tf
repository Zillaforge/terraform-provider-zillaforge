# Query floating IPs by name
data "zillaforge_floating_ips" "web_server" {
  name = "web-server-public-ip"
}

# Access the floating IP (returns empty list if not found)
output "web_fip_address" {
  description = "IP address of the web server floating IP"
  value       = length(data.zillaforge_floating_ips.web_server.floating_ips) > 0 ? data.zillaforge_floating_ips.web_server.floating_ips[0].ip_address : null
}

output "web_fip_id" {
  description = "ID of the web server floating IP"
  value       = length(data.zillaforge_floating_ips.web_server.floating_ips) > 0 ? data.zillaforge_floating_ips.web_server.floating_ips[0].id : null
}

# Query floating IP by ID
data "zillaforge_floating_ips" "specific" {
  id = "fip-12345678-1234-1234-1234-123456789abc"
}

output "specific_fip_details" {
  description = "Details of the specific floating IP"
  value = length(data.zillaforge_floating_ips.specific.floating_ips) > 0 ? {
    id         = data.zillaforge_floating_ips.specific.floating_ips[0].id
    name       = data.zillaforge_floating_ips.specific.floating_ips[0].name
    ip_address = data.zillaforge_floating_ips.specific.floating_ips[0].ip_address
    status     = data.zillaforge_floating_ips.specific.floating_ips[0].status
    device_id  = data.zillaforge_floating_ips.specific.floating_ips[0].device_id
  } : null
}

# Query floating IPs by IP address
data "zillaforge_floating_ips" "by_ip" {
  ip_address = "203.0.113.42"
}

output "fip_by_ip_found" {
  description = "Whether the specific IP address was found"
  value       = length(data.zillaforge_floating_ips.by_ip.floating_ips) > 0
}

# Query floating IPs by status
data "zillaforge_floating_ips" "active_ips" {
  status = "ACTIVE"
}

output "active_fip_count" {
  description = "Number of active floating IPs"
  value       = length(data.zillaforge_floating_ips.active_ips.floating_ips)
}

output "active_fip_addresses" {
  description = "List of active floating IP addresses"
  value       = [for fip in data.zillaforge_floating_ips.active_ips.floating_ips : fip.ip_address]
}

# Multiple filters (AND logic)
data "zillaforge_floating_ips" "web_active" {
  name   = "web-server-public-ip"
  status = "ACTIVE"
}

output "web_active_fip" {
  description = "Web server floating IP if active"
  value       = length(data.zillaforge_floating_ips.web_active.floating_ips) > 0 ? data.zillaforge_floating_ips.web_active.floating_ips[0] : null
}

# List all floating IPs
data "zillaforge_floating_ips" "all" {
  # No filters - lists all floating IPs
}

output "all_floating_ips" {
  description = "All floating IPs in the project"
  value = [for fip in data.zillaforge_floating_ips.all.floating_ips : {
    id         = fip.id
    name       = fip.name
    ip_address = fip.ip_address
    status     = fip.status
  }]
}

output "total_floating_ips" {
  description = "Total number of floating IPs"
  value       = length(data.zillaforge_floating_ips.all.floating_ips)
}