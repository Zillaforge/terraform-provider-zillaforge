# Basic security group resource
resource "zillaforge_security_group" "basic" {
  name        = "basic-sg"
  description = "Basic security group with minimal rules"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "0.0.0.0/0"
  }

  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "0.0.0.0/0"
  }
}

# Web server security group with HTTP/HTTPS/SSH
resource "zillaforge_security_group" "web_servers" {
  name        = "web-servers-prod"
  description = "Security group for production web tier"

  # Allow HTTP from anywhere
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }

  # Allow HTTPS from anywhere
  ingress_rule {
    protocol    = "tcp"
    port_range  = "443"
    source_cidr = "0.0.0.0/0"
  }

  # Allow SSH from admin network only
  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "203.0.113.0/24"
  }

  # Allow all outbound traffic (IPv4)
  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "0.0.0.0/0"
  }

  # Allow all outbound traffic (IPv6)
  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "::/0"
  }
}

# Database security group with port range
resource "zillaforge_security_group" "database" {
  name        = "database-tier"
  description = "Security group for database servers"

  # Allow PostgreSQL from application subnet
  ingress_rule {
    protocol    = "tcp"
    port_range  = "5432"
    source_cidr = "10.0.2.0/24"
  }

  # Allow MySQL from application subnet
  ingress_rule {
    protocol    = "tcp"
    port_range  = "3306"
    source_cidr = "10.0.2.0/24"
  }

  # Allow Redis cluster ports
  ingress_rule {
    protocol    = "tcp"
    port_range  = "6379-6389"
    source_cidr = "10.0.2.0/24"
  }

  # Allow ICMP (ping) for monitoring
  ingress_rule {
    protocol    = "icmp"
    port_range  = "all"
    source_cidr = "10.0.0.0/16"
  }

  # Allow outbound to application subnet only
  egress_rule {
    protocol         = "tcp"
    port_range       = "all"
    destination_cidr = "10.0.2.0/24"
  }
}

# IPv6-enabled security group
resource "zillaforge_security_group" "ipv6_enabled" {
  name        = "ipv6-web"
  description = "Security group with IPv6 support"

  # HTTP IPv4
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "0.0.0.0/0"
  }

  # HTTP IPv6
  ingress_rule {
    protocol    = "tcp"
    port_range  = "80"
    source_cidr = "::/0"
  }

  # Egress IPv4
  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "0.0.0.0/0"
  }

  # Egress IPv6
  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "::/0"
  }
}

# Output security group ID for reference
output "web_sg_id" {
  description = "Security group ID for web servers"
  value       = zillaforge_security_group.web_servers.id
}

# Example: Referencing security group in VPS instance (future resource)
# This demonstrates the pattern for attaching security groups to instances
# Note: zillaforge_vps_instance resource is not yet implemented
#
# resource "zillaforge_vps_instance" "web_server" {
#   name              = "web-prod-01"
#   flavor_id         = "flavor-uuid-here"
#   security_group_id = zillaforge_security_group.web_servers.id
#
#   # Alternative: Reference by data source lookup
#   # security_group_id = data.zillaforge_security_groups.by_name.security_groups[0].id
# }
