# Libredesk Infrastructure (Terraform)

Infrastructure as code for Libredesk on Option A stack: Neon, Upstash, Cloudflare R2, Fly.io.

**All resources are configured for EU regions:**
- Neon: aws-eu-central-1 (Frankfurt)
- Upstash: eu-central-1
- Cloudflare R2: weur (Western Europe)
- Fly.io: ams (Amsterdam) or cdg (Paris)

**Resend** has no Terraform provider — set up manually (see DEPLOYMENT_GUIDE.md).

## Terraform Cloud Workspaces

| Workspace | Environment |
|-----------|-------------|
| `libredesk-dev` | Dev/staging (staging.libredesk.io) |
| `libredesk-prod` | Production (app.libredesk.io) |

## Setup

1. Create workspaces in Terraform Cloud.
2. Set variables in each workspace (see DEPLOYMENT_GUIDE.md).
3. Update `backend "remote"` in `main.tf` with your org name.
4. Copy `terraform.tfvars.example` → `terraform.tfvars` (optional; prefer workspace vars for secrets).
5. Run `terraform init` and `terraform apply`.

## Variables

Define example values in `terraform.tfvars.example` only. Set real values in Terraform Cloud workspace variables.
