#!/bin/bash
# Example: Import an existing ZillaForge VPS server into Terraform state
#
# Usage:
#   ./import.sh <server-id>
#
# Prerequisites:
#   1. Existing server in ZillaForge platform
#   2. Terraform configuration file matching the server's actual configuration
#   3. ZillaForge provider configured with valid credentials
#
# Example:
#   ./import.sh 550e8400-e29b-41d4-a716-446655440000

set -e

if [ $# -eq 0 ]; then
    echo "Error: Server ID required"
    echo "Usage: $0 <server-id>"
    echo ""
    echo "Example:"
    echo "  $0 550e8400-e29b-41d4-a716-446655440000"
    exit 1
fi

SERVER_ID="$1"

# Verify server exists before attempting import
echo "Verifying server ${SERVER_ID} exists..."

# Import the server into Terraform state
echo "Importing server ${SERVER_ID} into Terraform state..."
terraform import zillaforge_server.imported "${SERVER_ID}"

echo ""
echo "Import successful!"
echo ""
echo "Next steps:"
echo "  1. Run 'terraform plan' to verify configuration matches imported state"
echo "  2. Update your configuration if there are any differences"
echo "  3. Run 'terraform apply' to confirm no changes are needed"
echo ""
echo "Note: The following attributes are not imported from the API (for security):"
echo "  - user_data: Set to null after import"
echo "  - password: Set to null after import"
echo ""
echo "If your server was created with user_data or password, you'll need to:"
echo "  - Remove these attributes from your configuration, OR"
echo "  - Use 'lifecycle { ignore_changes = [user_data, password] }' to prevent drift"
