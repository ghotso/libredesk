# Upstash Redis
# Docs: https://registry.terraform.io/providers/upstash/upstash/latest/docs

provider "upstash" {
  email   = var.upstash_email
  api_key = var.upstash_api_key
}

resource "upstash_redis_database" "libredesk" {
  database_name = "libredesk-${var.environment}"
  region        = "eu-central-1" # EU. Options: eu-central-1, eu-west-1
  tls           = true
}
