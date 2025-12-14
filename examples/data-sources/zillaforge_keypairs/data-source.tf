# Query all keypairs in the project
data "zillaforge_keypairs" "all" {
  # No filters - returns all keypairs
}

output "keypair_count" {
  description = "Total number of keypairs in the project"
  value       = length(data.zillaforge_keypairs.all.keypairs)
}

output "keypair_names" {
  description = "List of all keypair names"
  value       = [for k in data.zillaforge_keypairs.all.keypairs : k.name]
}

# Query keypairs by exact name
data "zillaforge_keypairs" "production" {
  name = "production-keypair"
}

# Reference the first match from name filter
resource "zillaforge_vps_instance" "web" {
  # ... other configuration ...
  keypair_id = length(data.zillaforge_keypairs.production.keypairs) > 0 ? data.zillaforge_keypairs.production.keypairs[0].id : null
}

# Query specific keypair by ID
data "zillaforge_keypairs" "specific" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "keypair_fingerprint" {
  description = "Fingerprint of the specific keypair"
  value       = length(data.zillaforge_keypairs.specific.keypairs) > 0 ? data.zillaforge_keypairs.specific.keypairs[0].fingerprint : null
}

output "public_key" {
  description = "Public key of the specific keypair"
  value       = length(data.zillaforge_keypairs.specific.keypairs) > 0 ? data.zillaforge_keypairs.specific.keypairs[0].public_key : null
}
