variable "environment" {
  description = "Environment: staging or prod"
  type        = string
  default     = "staging"
}

variable "app_name" {
  description = "Fly.io app name"
  type        = string
}

variable "root_url" {
  description = "Base URL for the app (e.g. https://staging.libredesk.io)"
  type        = string
}

variable "neon_api_key" {
  description = "Neon API key"
  type        = string
  sensitive   = true
}

variable "neon_org_id" {
  description = "Neon Organization ID (Account Settings → Organization settings)"
  type        = string
}

variable "upstash_email" {
  description = "Upstash account email"
  type        = string
}

variable "upstash_api_key" {
  description = "Upstash API key"
  type        = string
  sensitive   = true
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "fly_api_token" {
  description = "Fly.io API token"
  type        = string
  sensitive   = true
}

variable "fly_org" {
  description = "Fly.io organization slug (e.g. personal or your-org)"
  type        = string
  default     = "personal"
}

variable "fly_primary_region" {
  description = "Fly.io primary region for machines (EU: ams=Amsterdam, cdg=Paris)"
  type        = string
  default     = "ams"
}

variable "r2_bucket_name" {
  description = "Cloudflare R2 bucket name"
  type        = string
}
