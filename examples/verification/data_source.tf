data "zillaforge_flavors" "small_falvor" {
  vcpus  = 1
  memory = 1
}

output "flavor_id" {
  value = data.zillaforge_flavors.small_falvor.flavors[0].id
}

data "zillaforge_networks" "default" {
  name = "default"
}

output "network_id" {
  value = data.zillaforge_networks.default.networks[0].id
}

data "zillaforge_images" "cirros" {
  repository = "cirros"
}

output "image_id" {
  value = data.zillaforge_images.cirros.images[0].id
}