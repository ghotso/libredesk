terraform {
  required_version = ">= 1.0"

  required_providers {
    neon = {
      source  = "kislerdm/neon"
      version = "~> 0.2"
    }
    upstash = {
      source  = "upstash/upstash"
      version = "~> 2.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
    }
    fly = {
      source  = "DAlperin/fly-io"
      version = "~> 0.3"
    }
  }

  backend "remote" {
    organization = "your-terraform-cloud-org"

    workspaces {
      name = "libredesk-dev"
    }
  }
}

# Use different workspace for prod: run `terraform workspace select libredesk-prod`
# or use separate backend config per environment
