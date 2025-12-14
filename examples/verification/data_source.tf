data "zillaforge_flavors" "default" {
  name = "default"
}

data "zillaforge_networks" "default" {
  name = "default"
}

data "zillaforge_keypairs" "generated" {
  name = resource.zillaforge_keypair.auto_generated.name
}

output "default_flavor_id" {
  value = data.zillaforge_flavors.default.flavors[0].id
}

output "default_network_id" {
  value = data.zillaforge_networks.default.networks[0].id
}

output "keypair_names_id" {
  description = "generated keypair names"
  value       = data.zillaforge_keypairs.generated.keypairs[0].id
}