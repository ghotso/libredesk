# Libredesk Deployment Guide

This guide walks through setting up Libredesk for deployment on the Option A stack (Fly.io, Neon, Upstash, Cloudflare R2, Resend), using Terraform Cloud for infrastructure and GitHub Actions for CI/CD.

**All infrastructure is hosted in the EU** (Neon Frankfurt, Upstash eu-central-1, R2 Western Europe, Fly.io Amsterdam/Paris).

**Environments:**
- **Dev/Staging** — Secured subdomain (e.g. `staging.libredesk.io`), Terraform Cloud workspace `libredesk-dev`. Deploy first; use for testing.
- **Production** — Production domain (e.g. `app.libredesk.io`), Terraform Cloud workspace `libredesk-prod`. Deploy when ready; same setup, different workspace/vars.

---

## 1. Account Setup

Create accounts for each service before running Terraform or workflows:

| Service | Sign up | Notes |
|---------|---------|-------|
| **Terraform Cloud** | [app.terraform.io](https://app.terraform.io) | Already set up |
| **Fly.io** | [fly.io](https://fly.io) | `fly auth signup` or web |
| **Neon** | [neon.tech](https://neon.tech) | PostgreSQL |
| **Upstash** | [upstash.com](https://upstash.com) | Redis |
| **Cloudflare** | [cloudflare.com](https://cloudflare.com) | R2, DNS |
| **Resend** | [resend.com](https://resend.com) | System email (no Terraform) |

---

## 2. Terraform Cloud Setup

### 2.1 Workspaces

Create two workspaces in Terraform Cloud:

| Workspace | Purpose |
|-----------|---------|
| `libredesk-dev` | Dev/staging — staging.libredesk.io |
| `libredesk-prod` | Production — app.libredesk.io |

### 2.2 Workspace Variables (set in Terraform Cloud, not in repo)

Configure in: Workspace → Variables → Add variable.

**Common (both workspaces):**

| Variable | Description | Sensitive | Example |
|----------|-------------|------------|---------|
| `neon_api_key` | Neon API key | ✓ | `neon-api-key-xxx` |
| `neon_org_id` | Neon Organization ID (Account Settings → Organization settings) | | `your-org-id` |
| `upstash_email` | Upstash account email | | `you@example.com` |
| `upstash_api_key` | Upstash API key | ✓ | `AXxxx...` |
| `cloudflare_api_token` | Cloudflare API token (R2, DNS) | ✓ | `xxx` |
| `cloudflare_account_id` | Cloudflare account ID | | `xxx` |
| `fly_api_token` | Fly.io API token | ✓ | `fly-xxx` |
| `fly_org` | Fly.io org slug (e.g. personal) | | `personal` |
| `fly_primary_region` | Fly.io region: ams (Amsterdam) or cdg (Paris) | | `ams` |

**Dev/staging only:**

| Variable | Description | Example |
|----------|-------------|---------|
| `environment` | `dev` or `staging` | `staging` |
| `app_name` | Fly app name | `libredesk-staging` |
| `root_url` | Base URL | `https://staging.libredesk.io` |
| `r2_bucket_name` | R2 bucket name | `libredesk-staging-media` |

**Prod only:**

| Variable | Description | Example |
|----------|-------------|---------|
| `environment` | `prod` | `prod` |
| `app_name` | Fly app name | `libredesk` |
| `root_url` | Base URL | `https://app.libredesk.io` |
| `r2_bucket_name` | R2 bucket name | `libredesk-media` |

### 2.3 Resend (Manual — no Terraform)

1. Sign up at [resend.com](https://resend.com)
2. Verify your domain
3. Create API key
4. Store in Terraform Cloud workspace vars or GitHub secrets:

| Variable | Description | Where |
|----------|-------------|-------|
| `resend_api_key` | Resend API key | Terraform Cloud (or GitHub secret) |
| `notification_email_address` | From address | e.g. `notifications@yourdomain.com` |

---

## 3. Terraform Apply

### 3.1 Backend

Terraform uses Terraform Cloud remote backend. Configure in `terraform/main.tf`:

```hcl
terraform {
  backend "remote" {
    organization = "your-org-name"
    workspaces {
      name = "libredesk-dev"  # or libredesk-prod
    }
  }
}
```

### 3.2 Run Terraform

```bash
cd terraform

# Initialize (connects to Terraform Cloud)
terraform init

# Plan (uses workspace variables from Terraform Cloud)
terraform plan

# Apply
terraform apply
```

**Important:** Switch workspaces before applying for prod (or use separate config/dir per env).

---

## 4. GitHub Repository Secrets

Add these secrets in GitHub: Settings → Secrets and variables → Actions.

| Secret | Description | Used by |
|--------|-------------|---------|
| `FLY_API_TOKEN` | Fly.io API token | Deploy workflow |
| `FLY_APP_NAME` | Fly app name (e.g. `libredesk-staging`) | Deploy workflow |
| `LIBREDESK_APP_ENCRYPTION_KEY` | `openssl rand -hex 16` | Deploy |
| `LIBREDESK_DB_HOST` | Neon host (from Terraform output) | Deploy |
| `LIBREDESK_DB_PASSWORD` | Neon password (from Terraform output) | Deploy |
| `LIBREDESK_DB_USER` | Neon user | Deploy |
| `LIBREDESK_DB_DATABASE` | Neon database name | Deploy |
| `LIBREDESK_REDIS_URL` | Upstash Redis URL | Deploy |
| `LIBREDESK_UPLOAD__S3__URL` | R2 endpoint | Deploy |
| `LIBREDESK_UPLOAD__S3__ACCESS_KEY` | R2 access key | Deploy |
| `LIBREDESK_UPLOAD__S3__SECRET_KEY` | R2 secret key | Deploy |
| `LIBREDESK_UPLOAD__S3__BUCKET` | R2 bucket name | Deploy |
| `LIBREDESK_APP_ROOT_URL` | e.g. `https://staging.libredesk.io` | Deploy |
| `LIBREDESK_notification__email__host` | `smtp.resend.com` | Deploy |
| `LIBREDESK_notification__email__port` | `465` | Deploy |
| `LIBREDESK_notification__email__username` | `resend` | Deploy |
| `LIBREDESK_notification__email__password` | Resend API key | Deploy |
| `LIBREDESK_notification__email__email_address` | From address | Deploy |

**Environment-specific:** Use GitHub Environments (`staging`, `production`) to scope secrets per environment.

---

## 5. Workflow Triggers

| Workflow | Trigger | Action |
|----------|---------|--------|
| Dev/staging deploy | Push to `main` or `develop` | Deploy to `libredesk-staging` |
| Prod deploy | Release created, or manual | Deploy to `libredesk` |

Configure in `.github/workflows/release.yml` (or a dedicated `deploy.yml`).

---

## 6. First-Time Deployment (Dev/Staging)

1. **Terraform**
   - Create workspace `libredesk-dev` in Terraform Cloud
   - Set all required variables
   - Run `terraform apply` from `terraform/`

2. **Resend**
   - Create account, verify domain, create API key

3. **GitHub**
   - Add all secrets (or use environment `staging`)

4. **Deploy**
   - Push to `main` (or configured branch) to trigger deploy
   - Or run workflow manually

5. **Post-deploy**
   - Run `libredesk --install --idempotent-install --yes` (if first run)
   - Set System user password: `fly ssh console -a libredesk-staging` then `./libredesk --set-system-user-password`

---

## 7. Production Deployment

When ready for production:

1. Create Terraform Cloud workspace `libredesk-prod`
2. Set prod variables (different app name, bucket, root_url)
3. Run `terraform apply` for prod
4. Add GitHub environment `production` with prod secrets
5. Deploy via release workflow or manual trigger
6. Point production domain (e.g. `app.libredesk.io`) to Fly.io
7. Run install/upgrade and set System user password

---

## 8. Useful Commands

```bash
# Fly.io
fly apps list
fly status -a libredesk-staging
fly logs -a libredesk-staging
fly ssh console -a libredesk-staging

# Terraform
cd terraform && terraform output
```

---

*See MULTI_TENANT_AND_HOSTING_PLAN.md for full architecture and implementation phases.*
