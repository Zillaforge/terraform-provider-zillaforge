variable "api_endpoint" {
  description = "API endpoint for the provider"
  type        = string
}

variable "api_key" {
  description = "API key (sensitive)"
  type        = string
  sensitive   = true
}

variable "project_sys_code" {
  description = "Project system code"
  type        = string
}
