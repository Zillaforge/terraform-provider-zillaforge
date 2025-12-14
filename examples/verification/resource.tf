resource "zillaforge_keypair" "auto_generated" {
  name        = "auto-generated-key"
  description = "System-generated SSH keypair for web servers"
}

# output "private_key" {
#     sensitive = true
#     value = resource.zillaforge_keypair.auto_generated.private_key
# }