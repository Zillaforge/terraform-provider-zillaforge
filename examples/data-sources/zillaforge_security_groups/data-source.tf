# Query security group by name
data "zillaforge_security_groups" "web" {
  name = "web-servers-prod"
}

# Access the security group (returns empty list if not found)
output "web_sg_id" {
  value = length(data.zillaforge_security_groups.web.security_groups) > 0 ? data.zillaforge_security_groups.web.security_groups[0].id : null
}

output "web_sg_rules" {
  value = length(data.zillaforge_security_groups.web.security_groups) > 0 ? {
    ingress = data.zillaforge_security_groups.web.security_groups[0].ingress_rule
    egress  = data.zillaforge_security_groups.web.security_groups[0].egress_rule
  } : null
}

# Query security group by ID
data "zillaforge_security_groups" "specific" {
  id = "sg-12345678-1234-1234-1234-123456789abc"
}

output "specific_sg_name" {
  value = data.zillaforge_security_groups.specific.security_groups[0].name
}

# List all security groups
data "zillaforge_security_groups" "all" {
  # No filters - lists all security groups
}

output "all_security_group_names" {
  value = [for sg in data.zillaforge_security_groups.all.security_groups : sg.name]
}

output "total_security_groups" {
  value = length(data.zillaforge_security_groups.all.security_groups)
}

# Use in resource configuration
# Query existing security group and reference in instance
data "zillaforge_security_groups" "shared_web" {
  name = "shared-web-servers"
}

# Reference in future instance resource (not yet implemented)
# resource "zillaforge_vps_instance" "app" {
#   name = "app-server-1"
#   # ... other attributes ...
#   
#   security_group_ids = [
#     data.zillaforge_security_groups.shared_web.security_groups[0].id
#   ]
# }
