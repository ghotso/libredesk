output "neon_connection_uri" {
  description = "Neon PostgreSQL connection URI"
  value       = neon_project.libredesk.connection_uri
  sensitive   = true
}

output "upstash_redis_url" {
  description = "Upstash Redis endpoint URL"
  value       = upstash_redis_database.libredesk.endpoint
  sensitive   = true
}

output "r2_bucket_name" {
  description = "R2 bucket name for libredesk media"
  value       = cloudflare_r2_bucket.libredesk.name
}

output "fly_app_name" {
  description = "Fly.io app name"
  value       = fly_app.libredesk.name
}
