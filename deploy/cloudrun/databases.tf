# One Neon project per app: the free plan's 100 compute-hours a month are per project, and each
# database sleeps after 5 idle minutes. Each project's default role owns its database and runs the
# app's migrations, which create the app's own least-privilege login role.

resource "neon_project" "lighthouse" {
  name                      = "lighthouse"
  org_id                    = var.neon_org_id
  region_id                 = var.neon_region
  pg_version                = 17
  history_retention_seconds = 21600 # the free plan's maximum: six hours of point-in-time restore
  branch {
    name          = "main"
    database_name = "lighthouse"
    role_name     = "lighthouse_owner"
  }
}

resource "neon_project" "redacted" {
  name                      = "redacted"
  org_id                    = var.neon_org_id
  region_id                 = var.neon_region
  pg_version                = 17
  history_retention_seconds = 21600
  branch {
    name          = "main"
    database_name = "rbac"
    role_name     = "rbac_owner"
  }
}

# The apps' own login roles are created by their migrations with these passwords.
resource "random_password" "lighthouse_app" {
  length  = 40
  special = false
}

resource "random_password" "redacted_app" {
  length  = 40
  special = false
}

locals {
  # Direct (not pooled) endpoints: Redacted LISTENs for access changes, which a transaction-mode
  # pooler can't carry; Lighthouse needs few connections.
  lighthouse_host = neon_project.lighthouse.database_host
  redacted_host   = neon_project.redacted.database_host
}
