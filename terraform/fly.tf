# Fly.io — EU region (ams=Amsterdam, cdg=Paris)
# Docs: https://registry.terraform.io/providers/DAlperin/fly-io/latest

provider "fly" {
  api_token = var.fly_api_token
}

resource "fly_app" "libredesk" {
  name = var.app_name
  org  = var.fly_org
}

# When adding fly_machine: set region = var.fly_primary_region (e.g. "ams" or "cdg")
# Add fly_machine, fly_volume, etc. as needed for full app deployment
# Or use flyctl in GitHub Actions: fly deploy --region ams (or cdg)
