# Libredesk Multi-Tenancy & Hosting Option A Plan

This document provides a comprehensive plan for:
1. **Multi-tenant architecture** — Shared schema with `tenant_id`, following best practices
2. **Hosting Option A readiness** — Fly.io, Neon, Upstash, Cloudflare R2, Resend

---

## Email Clarification

| Email Type | Purpose | Provider | Scope |
|------------|---------|----------|-------|
| **System notifications** | Password reset, ticket assigned, SLA alerts, mentions, etc. | Config or env vars only | **Global** — single systemwide SMTP, configured via `config.toml` or `LIBREDESK_*` env vars (no UI) |
| **Ticket/conversation messages** | Customer-facing support emails (incoming/outgoing) | Tenant inboxes (Gmail, Outlook, SMTP) | **Per tenant** — each tenant's own inbox config |

System notifications (assignment alerts, password reset, SLA warnings, mentions) come from one global systemwide email account, configured via `config.toml` or environment variables — **not** via the Admin UI. Only customer-facing communications in conversations/tickets go via each tenant's inbox (Gmail, Outlook, or SMTP).

---

# Part I: Multi-Tenancy Implementation Plan

## 1. Tenant Model & Resolution

### 1.1 Tenant Table

```sql
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'trial')),
    settings JSONB DEFAULT '{}'::jsonb
);
CREATE UNIQUE INDEX index_tenants_on_slug ON tenants(slug);
```

### 1.2 Tenant Resolution Strategy

**Path prefixes only.** Tenant is resolved from the URL path:

| Path format | Example | Slug |
|-------------|---------|------|
| `/t/{slug}/...` | `/t/acme/inboxes/assigned` | `acme` |
| `/t/{slug}` (root) | `/t/acme` | `acme` |

All agent and portal routes are prefixed with `/t/{slug}`. The resolver extracts the slug, looks up the tenant, and sets it in context. No subdomains or headers.

### 1.3 Resolution Flow

```
Request → Extract slug from path (/t/{slug}/...) → Validate tenant exists & active
       → Set tenant_id in request context → Strip prefix, route to handler
```

### 1.4 Session & Tenant Binding

- Store `tenant_id` in session; validate path slug matches session tenant for logged-in agents (prevent cross-tenant access)
- Portal (contact) sessions: tenant derived from contact → contact_channels → inbox → tenant
- API key: bind `tenant_id` to API key at creation; path slug must match

---

## 2. Schema Changes (Shared Schema + tenant_id)

### 2.1 Table Classification

| Category | Tables | tenant_id | Notes |
|----------|--------|-----------|-------|
| **Root tenant tables** | tenants | — | New |
| **Tenant-owned** | sla_policies, business_hours, inboxes, teams, roles, users, conversation_statuses, conversation_priorities, organizations, automation_rules, tags, templates, oidc, ai_providers, ai_prompts, custom_attribute_definitions, webhooks | Add `tenant_id NOT NULL REFERENCES tenants(id)` | Core tenant data |
| **Tenant via FK** | contact_channels, organization_members, organization_domains, conversations, conversation_messages, conversation_drafts, macros, conversation_participants, conversation_mentions, conversation_last_seen, conversation_tags, csat_responses, views, applied_slas, sla_events, scheduled_sla_notifications, contact_notes, activity_logs, user_notifications | Inherit via parent | No direct tenant_id needed if always joined through tenant-owned parent |
| **Special** | settings, media | Tenant-scoped | See below |

### 2.2 Detailed Schema Changes

#### Tables with direct `tenant_id`

```sql
ALTER TABLE sla_policies ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE business_hours ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE inboxes ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE teams ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE roles ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE users ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE conversation_statuses ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE conversation_priorities ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE organizations ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE automation_rules ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE tags ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE templates ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE oidc ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE ai_providers ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE ai_prompts ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE custom_attribute_definitions ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE webhooks ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE activity_logs ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;

-- Backfill: create default tenant, assign existing rows (single-tenant migration)
-- Then add NOT NULL
```

#### Settings

Current: `(key, value)` with keys like `app.root_url`.

**Option A (recommended):** Add `tenant_id` to settings, composite key `(tenant_id, key)`.

**Note:** System notification email (`notification.email.*`) is configured via `config.toml` or env vars only — not stored in the settings table. No UI for it.

```sql
ALTER TABLE settings ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE DEFAULT NULL;
-- NULL tenant_id = global/platform settings (e.g. feature flags)
-- Non-NULL = tenant-specific (app.root_url, logo, etc. per tenant)
CREATE UNIQUE INDEX index_settings_on_tenant_id_and_key ON settings(COALESCE(tenant_id::text, 'global'), key);
```

#### Media

Media is linked via `model_type`/`model_id` (e.g. conversation_messages). Isolation is via conversation → inbox → tenant. For extra safety and easier cleanup:

```sql
ALTER TABLE media ADD COLUMN tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE;
```

#### Unique Constraints to Update

| Table | Current | New |
|-------|---------|-----|
| users | `(email, type)` WHERE deleted_at IS NULL | `(tenant_id, email, type)` WHERE deleted_at IS NULL |
| teams | `(name)` UNIQUE | `(tenant_id, name)` UNIQUE |
| tags | `(name)` UNIQUE | `(tenant_id, name)` UNIQUE |
| roles | `(name)` UNIQUE | `(tenant_id, name)` UNIQUE |
| conversation_statuses | `(name)` UNIQUE | `(tenant_id, name)` UNIQUE |
| conversation_priorities | `(name)` UNIQUE | `(tenant_id, name)` UNIQUE |
| custom_attribute_definitions | `(key, applies_to)` UNIQUE | `(tenant_id, key, applies_to)` UNIQUE |
| ai_providers | `(name)` UNIQUE | `(tenant_id, name)` UNIQUE |
| reference_number | Global sequence | Per-tenant: `tenant_id` prefix or per-tenant sequence |

### 2.3 Reference Number Strategy

Current: `generate_reference_number(prefix)` uses a global sequence.

**Options:**
- **A)** Add tenant prefix: `TENANT-123-100` (tenant_id or slug)
- **B)** Per-tenant sequence: `CREATE SEQUENCE conversation_ref_seq_tenant_1 START 100;`
- **C)** Keep global sequence with tenant_id in conversations (simplest)

**Recommended:** Option A — `reference_number` format `{tenant_slug}-{seq}` e.g. `acme-100`, `acme-101`.

### 2.4 System User Per Tenant

Current: Single global "System" user (email=`System`).

**Change:** One System user per tenant.

```sql
CREATE UNIQUE INDEX index_users_on_tenant_email_type
  ON users(tenant_id, email, type) WHERE deleted_at IS NULL;
```

- `GetSystemUser()` becomes `GetSystemUser(tenantID int)`
- Install: Create default tenant + System user for that tenant
- Auto-assigner, drafts, SLA, etc. use tenant's System user

---

## 3. Affected Internal Packages & Queries

### 3.1 Package-by-Package Changes

| Package | Changes |
|---------|---------|
| **auth** | Session stores tenant_id; OIDC redirect URIs per tenant |
| **authz** | Enforce within tenant (roles/permissions scoped) |
| **user** | All queries +tenant_id; `GetAgent(tenantID, id, email)`, `VerifyPassword(tenantID, email, password)` |
| **inbox** | List/filter by tenant_id |
| **team** | All ops scoped by tenant_id |
| **organization** | Scoped via tenant |
| **conversation** | All queries join through tenant-owned entities; add tenant_id checks |
| **role** | Scoped by tenant |
| **tag** | Scoped by tenant |
| **macro** | Via user/team → tenant |
| **view** | Via user/team → tenant |
| **automation** | Scoped by tenant_id |
| **sla** | Via team/conversation → tenant |
| **setting** | `Get(tenantID, key)`, `GetByPrefix(tenantID, prefix)` |
| **oidc** | Scoped by tenant_id |
| **ai** | Provider config per tenant |
| **custom_attribute** | Scoped by tenant |
| **webhook** | Scoped by tenant |
| **activity_log** | Add tenant_id, filter by tenant |
| **notification** | Dispatcher uses global system email (notification.email); no tenant scoping |
| **report** | Filter all by tenant |
| **search** | Add tenant_id to conversation/user search |
| **media** | Optional tenant_id for isolation; ensure access checks consider tenant |

### 3.2 Request Context

```go
type TenantContext struct {
    ID   int
    Slug string
    Name string
}

r.RequestCtx.SetUserValue("tenant", TenantContext{ID: tenant.ID, Slug: tenant.Slug, Name: tenant.Name})
```

### 3.3 Manager Signatures

All managers that touch tenant data need `tenantID` (from context or explicit param):

```go
func (u *Manager) GetAgent(tenantID int, id int64, email string) (models.User, error)
func (u *Manager) VerifyPassword(tenantID int, email string, password []byte) (models.User, error)

tenant := r.RequestCtx.UserValue("tenant").(TenantContext)
user, err := app.user.GetAgent(tenant.ID, 0, email)
```

---

## 4. Tenant Resolution Middleware

All agent and portal routes use path prefix `/t/{slug}`. Middleware parses the slug and resolves the tenant:

```go
func tenantResolver(next fastglue.FastRequestHandler) fastglue.FastRequestHandler {
    return func(r *fastglue.Request) error {
        slug := r.RequestCtx.UserValue("slug").(string)  // from route /t/{slug}/...
        tenant := lookupTenantBySlug(slug)
        if tenant == nil {
            return r.SendErrorEnvelope(404, "Tenant not found", nil, envelope.NotFoundError)
        }
        r.RequestCtx.SetUserValue("tenant", tenant)
        return next(r)
    }
}
```

**Routing:** Register routes under `/t/{slug}`, e.g. `/t/{slug}/api/v1/auth/login`, `/t/{slug}/inboxes/assigned`. Exclude: health (`/health`), static assets, OIDC callback (tenant encoded in state param).

---

## 5. Install & Migration Strategy

### 5.1 New Install (Multi-Tenant)

1. Run schema migrations (create tenants, add tenant_id, etc.)
2. Create default tenant (e.g. slug `default`)
3. Create System user for default tenant
4. Seed conversation_statuses, conversation_priorities, roles for default tenant
5. Seed default settings for default tenant

### 5.2 Migrating Existing Single-Tenant

1. Add `tenants` table
2. Add `tenant_id` columns (nullable)
3. Create tenant "Migration default"
4. Update all rows: `UPDATE users SET tenant_id = 1 WHERE tenant_id IS NULL`
5. Backfill settings with tenant_id
6. Add NOT NULL, indexes, update unique constraints
7. Update application code

### 5.3 Migration File Structure

```
internal/migrations/
  v1.4.0.go  -- Add tenants table, tenant_id columns, backfill, constraints
```

---

## 6. Storage (R2) Tenant Isolation

Use `bucket_path` for tenant isolation:

```toml
[upload.s3]
bucket_path = "tenant-{tenant_id}/"
```

S3/R2 store must receive tenant context when generating paths. Pass tenant into each media call from request context.

---

# Part II: Hosting Option A Readiness

**See [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)** for step-by-step account setup, Terraform Cloud variables, GitHub secrets, and workflow configuration. Dev/staging first on a secured subdomain; production deployment when ready.

## 7. Stack Overview

**All infrastructure is hosted in the EU** for compliance and latency:

| Region | Service | Location |
|--------|---------|----------|
| EU | Neon | aws-eu-central-1 (Frankfurt) |
| EU | Upstash | eu-central-1 |
| EU | Cloudflare R2 | weur (Western Europe) |
| EU | Fly.io | ams (Amsterdam) or cdg (Paris) |
| EU | Resend | EU data centers (verify in Resend dashboard) |

| Component | Service | Purpose | Terraform |
|-----------|---------|---------|-----------|
| Compute | **Fly.io** | Run Libredesk app | DAlperin/fly-io |
| PostgreSQL | **Neon** | Primary database | kislerdm/neon |
| Redis | **Upstash** | Sessions, caching | upstash/upstash |
| Object Storage | **Cloudflare R2** | Media/uploads | cloudflare/cloudflare |
| System Email | **Resend** | Password reset, notifications | **No provider** — manual setup |

---

## 7.1 Infrastructure as Code (Terraform)

**Terraform Cloud** is used with **separate workspaces** for each environment:

| Workspace | Purpose | Subdomain / URL |
|-----------|---------|-----------------|
| `libredesk-dev` | Dev/staging | `staging.libredesk.io` (or secured subdomain) |
| `libredesk-prod` | Production | `app.libredesk.io` (or production domain) |

### Terraform-managed resources

- **Neon:** Project, database, branch (via `neon` provider)
- **Upstash:** Redis database (via `upstash` provider)
- **Cloudflare:** R2 bucket, API tokens for R2 (via `cloudflare` provider)
- **Fly.io:** App and machines via DAlperin/fly-io Terraform provider; deploy/update via `flyctl` in GitHub Actions or Terraform

### Resend (manual)

Resend has no Terraform provider. Create account, verify domain, create API key manually. Document in **DEPLOYMENT_GUIDE.md**; store API key in Terraform Cloud workspace vars or GitHub secrets.

### Variable strategy

- All sensitive values and environment-specific config go in **Terraform Cloud workspace variables**
- Repository contains only **example** values in `terraform/terraform.tfvars.example` (or `variables.example.tfvars`)
- Set real values in Terraform Cloud: Workspace → Variables → Add variable

### Directory structure

```
terraform/
  main.tf           # Providers, backend (Terraform Cloud)
  variables.tf      # Variable definitions
  outputs.tf        # Outputs (DB URL, Redis URL, R2 bucket)
  neon.tf           # Neon project, database
  upstash.tf        # Upstash Redis
  cloudflare.tf     # Cloudflare R2 bucket
  fly.tf            # Fly.io app (DAlperin/fly-io provider)
  terraform.tfvars.example  # Example values only (set real values in TFC)
```

---

## 8. Fly.io Configuration

### 8.1 fly.toml Example

```toml
app = "libredesk"
primary_region = "ams"  # Amsterdam, EU. Use "cdg" for Paris

[build]
  dockerfile = "Dockerfile"

[env]
  LIBREDESK_APP_ENV = "prod"
  LIBREDESK_APP_LOG_LEVEL = "info"
  LIBREDESK_APP_SERVER_ADDRESS = "0.0.0.0:9000"

[http_service]
  internal_port = 9000
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 1
  processes = ["app"]

[[vm]]
  memory = "1gb"
  cpu_kind = "shared"
  cpus = 1

[checks]
  [checks.health]
    port = 9000
    type = "http"
    path = "/health"
    interval = "10s"
```

### 8.2 Required Secrets

```bash
fly secrets set \
  LIBREDESK_APP_ENCRYPTION_KEY="$(openssl rand -hex 16)" \
  LIBREDESK_DB_HOST="<neon-host>" \
  LIBREDESK_DB_PORT="5432" \
  LIBREDESK_DB_USER="<neon-user>" \
  LIBREDESK_DB_PASSWORD="<neon-password>" \
  LIBREDESK_DB_DATABASE="<neon-dbname>" \
  LIBREDESK_REDIS_URL="<upstash-redis-url>" \
  LIBREDESK_UPLOAD__S3__URL="<r2-endpoint>" \
  LIBREDESK_UPLOAD__S3__ACCESS_KEY="<r2-access-key>" \
  LIBREDESK_UPLOAD__S3__SECRET_KEY="<r2-secret-key>" \
  LIBREDESK_UPLOAD__S3__BUCKET="<r2-bucket>" \
  LIBREDESK_APP_ROOT_URL="https://libredesk.yourdomain.com"
```

For Resend (system email), configure via `config.toml` or env vars (no UI). Example env vars:

```bash
LIBREDESK_notification__email__host="smtp.resend.com"
LIBREDESK_notification__email__port="465"
LIBREDESK_notification__email__username="resend"
LIBREDESK_notification__email__password="<resend-api-key>"
LIBREDESK_notification__email__email_address="notifications@yourdomain.com"
```

---

## 9. Neon PostgreSQL

- Create project at [neon.tech](https://neon.tech)
- Use connection string for host, user, password, database
- Enable `pg_trgm` (Neon supports it)
- Use pooled connection string if available
- SSL: `sslmode=require`

```toml
[db]
host = "<neon-endpoint>"
port = 5432
user = "<user>"
password = "<password>"
database = "<dbname>"
ssl_mode = "require"
max_open = 20
max_idle = 10
max_lifetime = "300s"
```

---

## 10. Upstash Redis

- Create database at [upstash.com](https://upstash.com)
- Use Redis URL: `redis://default:<password>@<host>:<port>`

```toml
[redis]
url = "redis://default:<password>@<endpoint>.upstash.io:6379"
```

For TLS: `rediss://` instead of `redis://`.

---

## 11. Cloudflare R2

- Create R2 bucket in Cloudflare dashboard
- Create API token with R2 read/write
- S3-compatible endpoint: `https://<account_id>.r2.cloudflarestorage.com`

```toml
[upload]
provider = "s3"

[upload.s3]
url = "https://<account_id>.r2.cloudflarestorage.com"
access_key = "<r2-access-key-id>"
secret_key = "<r2-secret-access-key>"
region = "auto"
bucket = "libredesk-media"
bucket_path = ""
expiry = "30m"
```

For multi-tenant: `bucket_path = "tenants/"` and prepend tenant slug/id in code when storing.

---

## 12. Resend (System Email)

- Sign up at [resend.com](https://resend.com)
- Verify domain
- Create API key
- SMTP: `smtp.resend.com`, port 465 (TLS), username `resend`, password = API key

**Config/env only** — no Admin UI. Add to `config.toml` or set via `LIBREDESK_*` env vars:

```toml
# System notification email (config or env only — not stored in DB, no UI)
[notification.email]
host = "smtp.resend.com"
port = 465
username = "resend"
password = "<api-key>"
email_address = "notifications@yourdomain.com"
tls_type = "tls"
auth_protocol = "plain"
enabled = true
```

---

## 13. Environment Variable Mapping

Koanf loads `LIBREDESK_` prefixed env vars. Nested keys use `__`:

| Config Key | Env Var |
|------------|---------|
| app.encryption_key | LIBREDESK_APP_ENCRYPTION_KEY |
| app.root_url | LIBREDESK_APP_ROOT_URL |
| app.env | LIBREDESK_APP_ENV |
| db.host | LIBREDESK_DB_HOST |
| db.port | LIBREDESK_DB_PORT |
| db.user | LIBREDESK_DB_USER |
| db.password | LIBREDESK_DB_PASSWORD |
| db.database | LIBREDESK_DB_DATABASE |
| db.ssl_mode | LIBREDESK_DB_SSL_MODE |
| redis.url | LIBREDESK_REDIS_URL |
| upload.s3.url | LIBREDESK_UPLOAD__S3__URL |
| upload.s3.access_key | LIBREDESK_UPLOAD__S3__ACCESS_KEY |
| upload.s3.secret_key | LIBREDESK_UPLOAD__S3__SECRET_KEY |
| upload.s3.bucket | LIBREDESK_UPLOAD__S3__BUCKET |
| notification.email.host | LIBREDESK_notification__email__host |
| notification.email.port | LIBREDESK_notification__email__port |
| notification.email.username | LIBREDESK_notification__email__username |
| notification.email.password | LIBREDESK_notification__email__password |
| notification.email.email_address | LIBREDESK_notification__email__email_address |

---

## 14. Path Prefix Routing (Multi-Tenant)

All tenant routes use path prefixes: `/t/{slug}/...`

1. **Route structure:** `/t/acme/api/v1/...`, `/t/acme/inboxes/assigned`, etc.
2. **Frontend:** Base path is `/t/{slug}`; all API calls and navigation are relative to that.
3. **Single domain:** No wildcard DNS or subdomains; one domain (e.g. `app.libredesk.io`) serves all tenants.
4. **Login/landing:** Unauthenticated users hit `/` or `/t/{slug}`; redirect to login at `/t/{slug}/` or similar.

---

# Part III: Implementation Phases

## Phase 1: Hosting Readiness (No Multi-Tenancy)

1. Add Terraform config (`terraform/`) for Neon, Upstash, Cloudflare R2, Fly.io
2. Create Terraform Cloud workspaces: `libredesk-dev`, `libredesk-prod`
3. Add `[notification.email]` to config.sample.toml (config/env only, no UI)
4. Add Fly.io config (fly.toml)
5. Set up Resend manually (no Terraform provider)
6. Write **DEPLOYMENT_GUIDE.md** — accounts, Terraform Cloud vars, GitHub secrets, workflow setup
7. Deploy dev/staging to secured subdomain; validate health, login, inbox, messages
8. **Deliverable:** Single-tenant Libredesk on Option A; IaC in place; deployment guide complete

**Estimated effort:** 2–3 days

---

## Phase 2: Schema & Tenant Model

1. Create `tenants` table
2. Add nullable `tenant_id` to all tenant-owned tables
3. Create migration with backfill for existing data (default tenant)
4. Add NOT NULL, indexes, update unique constraints
5. Add `tenant_id` to settings, media
6. Update reference number to include tenant
7. System user per tenant
8. **Deliverable:** Schema supports multi-tenant; existing install remains single-tenant

**Estimated effort:** 3–5 days

---

## Phase 3: Tenant Resolution & Context

1. Implement tenant resolver middleware (path prefix `/t/{slug}`)
2. Add TenantContext to request
3. Create `tenant` table CRUD (admin/super-admin only for now)
4. **Deliverable:** All requests have tenant context where applicable

**Estimated effort:** 2–3 days

---

## Phase 4: Application Layer Scoping

1. Update all managers to accept/use tenant_id
2. Update all queries (queries.sql across packages)
3. Update handlers to pass tenant from context
4. Update auth: session stores tenant_id, login resolves tenant first
5. Update portal: resolve tenant from contact's inbox
6. Update OIDC: per-tenant client configs
7. notification.email: config/env only — add `[notification]` section to config.sample.toml; load from koanf at init; remove Admin UI for system email settings
8. **Deliverable:** Full tenant isolation in application layer

**Estimated effort:** 5–8 days

---

## Phase 5: Install, Migration & Onboarding

1. Update install flow: create default tenant + System user
2. Tenant onboarding flow (create tenant, seed defaults)
3. Document migration from single-tenant to multi-tenant
4. **Deliverable:** Clean install and migration paths

**Estimated effort:** 2–3 days

---

## Phase 6: Testing, Hardening & CI/CD Refactor

1. Unit tests for tenant-scoped queries
2. Integration tests: create tenant, create agent, create conversation
3. Cross-tenant isolation tests (ensure tenant A cannot access tenant B data)
4. Load testing on Fly.io
5. Refactor GitHub Actions workflows for new deployment strategy (Fly.io, Option A stack)
   - `go.yml` — build/test; deploy to Fly.io instead of/in addition to current targets
   - `frontend-ci.yml` — ensure frontend CI works with new setup
   - `release.yml` — update release workflow for Fly.io deployment
   - `crowdin.yml` — translations (likely minimal changes)
6. Complete **DEPLOYMENT_GUIDE.md** — repo secrets, workflow triggers, dev/staging vs prod deployment steps
7. **Deliverable:** Test suite, confidence in isolation, CI/CD aligned with hosting Option A, deployment guide for dev and prod

**Estimated effort:** 3–5 days

---

# Appendix A: Tables Reference (Complete)

| Table | tenant_id | Inherits From |
|-------|-----------|---------------|
| tenants | — | — |
| sla_policies | ✓ | — |
| business_hours | ✓ | — |
| inboxes | ✓ | — |
| teams | ✓ | — |
| roles | ✓ | — |
| users | ✓ | — |
| user_roles | — | users |
| conversation_statuses | ✓ | — |
| conversation_priorities | ✓ | — |
| contact_channels | — | inboxes |
| organizations | ✓ | — |
| organization_members | — | organizations |
| organization_domains | — | organizations |
| conversations | — | inboxes |
| conversation_messages | — | conversations |
| automation_rules | ✓ | — |
| conversation_drafts | — | conversations |
| macros | — | users/teams |
| conversation_participants | — | conversations |
| conversation_mentions | — | conversations |
| conversation_last_seen | — | conversations |
| media | ✓ (optional) | — |
| oidc | ✓ | — |
| settings | ✓ | — |
| tags | ✓ | — |
| team_members | — | teams |
| templates | ✓ | — |
| conversation_tags | — | conversations |
| csat_responses | — | conversations |
| views | — | users/teams |
| applied_slas | — | conversations |
| sla_events | — | applied_slas |
| scheduled_sla_notifications | — | applied_slas |
| ai_providers | ✓ | — |
| ai_prompts | ✓ | — |
| custom_attribute_definitions | ✓ | — |
| contact_notes | — | users (contact) |
| activity_logs | ✓ | — |
| webhooks | ✓ | — |
| user_notifications | — | users |

---

# Appendix B: File Checklist for Multi-Tenancy

| File/Dir | Changes |
|----------|---------|
| cmd/middlewares.go | Tenant resolver, set tenant in context |
| cmd/handlers.go | Ensure tenant required on protected routes |
| cmd/login.go | Resolve tenant before VerifyPassword |
| cmd/auth.go | OIDC: tenant in state, redirect URI per tenant |
| cmd/portal.go | Resolve tenant from contact |
| cmd/*.go (all handlers) | Pass tenant from context to managers |
| internal/user/* | Add tenantID to all queries |
| internal/inbox/* | Add tenantID |
| internal/team/* | Add tenantID |
| internal/conversation/* | Tenant scope via inbox/user |
| internal/setting/* | Get/Update by tenant_id |
| internal/role/* | Add tenantID |
| internal/tag/* | Add tenantID |
| internal/organization/* | Add tenantID |
| internal/oidc/* | Add tenantID |
| internal/automation/* | Add tenantID |
| internal/sla/* | Tenant scope |
| internal/webhook/* | Add tenantID |
| internal/activity_log/* | Add tenantID |
| internal/ai/* | Provider per tenant |
| internal/media/* | tenant in path (optional) |
| internal/search/* | Add tenant filters |
| internal/report/* | Add tenant filters |
| schema.sql | Add tenants, tenant_id columns |
| internal/migrations/ | New migration v1.4.0 |
| config.sample.toml | Add [notification.email] section; document Option A vars |
| cmd/settings.go | Remove Admin UI for notification email (system email is config/env only) |
| .github/workflows/*.yml | Refactor for Fly.io / Option A deployment strategy |
| terraform/* | IaC for Neon, Upstash, Cloudflare R2, Fly.io |
| DEPLOYMENT_GUIDE.md | Accounts, Terraform Cloud vars, GitHub secrets, workflow setup, dev vs prod |

---

*Document version: 1.0*
