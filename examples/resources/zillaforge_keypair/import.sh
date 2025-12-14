#!/bin/bash
# Import an existing keypair by ID
# Usage: ./import.sh <keypair-id>

KEYPAIR_ID=${1:-"550e8400-e29b-41d4-a716-446655440000"}

echo "Importing keypair with ID: $KEYPAIR_ID"
echo "Note: Imported keypairs will have private_key set to null (not available after creation)"

terraform import zillaforge_keypair.existing "$KEYPAIR_ID"

# After import, run terraform plan to see the imported state:
# terraform plan
#
# Example output:
#   zillaforge_keypair.existing:
#     id          = "550e8400-e29b-41d4-a716-446655440000"
#     name        = "imported-key"
#     description = "Existing keypair"
#     public_key  = "ssh-ed25519 AAAAC3Nza..."
#     fingerprint = "SHA256:abc123..."
#     private_key = null  # Never available for imported resources
