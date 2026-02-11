# Cloudflare R2
# Docs: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

resource "cloudflare_r2_bucket" "libredesk" {
  account_id = var.cloudflare_account_id
  name       = var.r2_bucket_name
  location   = "weur" # Western Europe. Options: weur, eeur, enam, wnam, apac, oc
}

# R2 API tokens for app access: create via Cloudflare UI or cloudflare_api_token resource
# Store access_key/secret_key in Terraform Cloud / GitHub secrets
