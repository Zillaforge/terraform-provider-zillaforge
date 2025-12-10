# T057: Example configuration with all 4 attributes
provider "zillaforge" {
  # API endpoint - defaults to production if not specified
  api_endpoint = "https://api.zillaforge.com"

  # API key - required for authentication
  # Can be provided via ZILLAFORGE_API_KEY environment variable
  api_key = var.zillaforge_api_key

  # Project identifier - use exactly one of:
  # - project_id (numeric or UUID)
  # - project_sys_code (alphanumeric system code)

  # Option 1: Using project_id
  # project_id = "12345"

  # Option 2: Using project_sys_code (mutually exclusive with project_id)
  project_sys_code = "my-project-code"
}

# T058: Multi-instance provider configuration with aliases
# Use provider aliases to manage resources across different projects
provider "zillaforge" {
  alias            = "project_a"
  api_key          = var.zillaforge_api_key
  project_sys_code = "project-a"
}

provider "zillaforge" {
  alias      = "project_b"
  api_key    = var.zillaforge_api_key
  project_id = "67890"
}

# Example: Using aliased providers in resource configuration
# resource "zillaforge_example" "resource_in_project_a" {
#   provider = zillaforge.project_a
#   # resource configuration
# }
#
# resource "zillaforge_example" "resource_in_project_b" {
#   provider = zillaforge.project_b
#   # resource configuration
# }

# Environment variable usage example:
# export ZILLAFORGE_API_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
# export ZILLAFORGE_PROJECT_ID="12345"
# terraform plan
