#!/bin/bash
# T063-T064: Import example script for zillaforge_security_group resource

# Import an existing security group by its ID
# The security group must already exist in your ZillaForge account
# You can find the security group ID from the ZillaForge console or API

# Usage:
#   1. Create a security group resource definition in your Terraform configuration:
#
#      resource "zillaforge_security_group" "imported" {
#        name        = "existing-security-group"
#        description = "Security group imported from ZillaForge"
#        
#        # Define rules that match the existing security group
#        ingress_rule {
#          protocol    = "tcp"
#          port_range  = "22"
#          source_cidr = "10.0.0.0/8"
#        }
#      }
#
#   2. Run the import command with the security group ID:

terraform import zillaforge_security_group.imported sg-12345678-1234-1234-1234-123456789abc

# After import:
# - Run `terraform plan` to verify the configuration matches the imported state
# - If there are differences, update your Terraform configuration to match
# - Run `terraform apply` to manage the security group going forward

# Common scenarios:

# Example 1: Import a web security group
# terraform import zillaforge_security_group.web sg-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa

# Example 2: Import a database security group  
# terraform import zillaforge_security_group.db sg-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb

# Example 3: Import and verify no changes needed
# terraform import zillaforge_security_group.imported <SECURITY_GROUP_ID>
# terraform plan  # Should show "No changes" if config matches

# Troubleshooting:
# - Error "Invalid Import ID Format": Ensure the ID is a valid UUID format
# - Error "Unable to read security group": Verify the security group exists and you have access
# - Plan shows changes after import: Update your Terraform config to match the actual security group state

# Notes:
# - The import ID must be the security group UUID, not the name
# - All rules (ingress and egress) will be imported
# - After import, Terraform will manage all future changes to the security group
# - The security group must exist before importing (create via UI/API if needed)
