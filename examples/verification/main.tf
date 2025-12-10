terraform {
  required_providers {
    zillaforge = {
      source = "registry.terraform.io/Zillaforge/zillaforge"
    }
  }
}

provider "zillaforge" {
  api_endpoint     = var.api_endpoint
  api_key          = var.api_key
  project_sys_code = var.project_sys_code
}

data "zillaforge_coffees" "example" {}