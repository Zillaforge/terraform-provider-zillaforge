# Example 1: System-generated keypair (recommended for most use cases)
resource "zillaforge_keypair" "auto_generated" {
  name        = "auto-generated-key"
  description = "System-generated SSH keypair for web servers"
}

# Output the generated keys (save private_key securely!)
output "auto_public_key" {
  description = "Public key to share with team or external services"
  value       = zillaforge_keypair.auto_generated.public_key
}

output "auto_private_key" {
  description = "Private key for SSH access (SAVE THIS SECURELY - shown once)"
  value       = zillaforge_keypair.auto_generated.private_key
  sensitive   = true # Prevents display in console output
}

output "auto_fingerprint" {
  description = "Key fingerprint for verification"
  value       = zillaforge_keypair.auto_generated.fingerprint
}

# Example 2: User-provided public key (bring your own key)
resource "zillaforge_keypair" "user_provided" {
  name        = "team-shared-key"
  description = "Shared team key for production servers"
  public_key  = file("~/.ssh/id_ed25519.pub") # Read from local file
}

# Note: private_key will be null for user-provided keys
output "user_keypair_id" {
  description = "Keypair ID for reference in other resources"
  value       = zillaforge_keypair.user_provided.id
}

# Example 3: Minimal configuration (system-generated, no description)
resource "zillaforge_keypair" "minimal" {
  name = "minimal-example"
}

# Example 4: Update description (only updatable field)
resource "zillaforge_keypair" "updatable" {
  name        = "example-key"
  description = "Initial description" # Can be changed without replacement
}

# Changing name or public_key forces replacement:
# terraform plan will show:
#   # zillaforge_keypair.updatable must be replaced
#   -/+ resource "zillaforge_keypair" "updatable" {
#         name = "example-key" -> "new-name"  # forces replacement
#       }
