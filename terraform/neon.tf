# Neon PostgreSQL
# Docs: https://registry.terraform.io/providers/kislerdm/neon/latest
# Auth: set NEON_API_KEY env var, or use api_key in provider block

provider "neon" {}
# Neon Terraform reference: https://neon.tech/docs/reference/terraform

resource "neon_project" "libredesk" {
  name   = "libredesk-${var.environment}"
  region_id = "aws-eu-central-1" # Frankfurt, EU. See https://neon.tech/docs/introduction/regions
  pg_version = 16
  org_id = var.neon_org_id

  branch {
    name          = "main"
    database_name = "libredesk"
    role_name     = "libredesk"
  }

  # Free tier: max 21600 (6h); paid: default 86400
  history_retention_seconds = 21600
}

# Provider: set NEON_API_KEY env var or api_key in provider block
