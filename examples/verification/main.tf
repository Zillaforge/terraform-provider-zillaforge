terraform {
  required_providers {
    zillaforge = {
      source  = "hashicorp/zillaforge"
      version = "0.0.1-alpha"
    }
  }
}

provider "zillaforge" {
  # Required parameters from .env file 
  # ZILLAFORGE_API_ENDPOINT=...
  # ZILLAFORGE_API_KEY=...
  # ZILLAFORGE_PROJECT_SYS_CODE=...
}
