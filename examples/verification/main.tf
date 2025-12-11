terraform {
  required_providers {
    zillaforge = {
      source = "registry.terraform.io/Zillaforge/zillaforge"
    }
  }
}

provider "zillaforge" {
  api_endpoint     = var.api_endpoint
  api_key          = var.api_key
  project_sys_code = var.project_sys_code
}

data "zillaforge_flavors" "default" {
  name = "default"
}

data "zillaforge_networks" "default" {
  name = "default"
}

output "default_flavor_id" {
  value = data.zillaforge_flavors.default.flavors[0].id
}

output "default_network_id" {
  value = data.zillaforge_networks.default.networks[0].id
}