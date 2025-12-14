terraform {
  required_providers {
    zillaforge = {
      source = "registry.terraform.io/Zillaforge/zillaforge"
    }
  }
}

provider "zillaforge" {
  # Required parameters from .env file 
  # ZILLAFORGE_API_ENDPOINT=...
  # ZILLAFORGE_API_KEY=...
  # ZILLAFORGE_PROJECT_SYS_CODE=...
}
