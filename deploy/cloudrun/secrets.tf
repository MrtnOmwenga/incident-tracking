# Credentials live in Secret Manager; each service reads only its own (and its migration job, which
# runs as the same account). The free tier covers 6 secret versions, so exactly six are kept here:
# the database passwords, the OAuth client secret and the MongoDB connection string. The demos'
# token-signing keys and the edge secret are plain environment variables instead (services.tf):
# only principals who can deploy to a service can read its configuration, and those can already
# run code that reads its secrets, so Secret Manager would add cost, not protection. Every value
# is also in Terraform state, which is encrypted at rest in a private bucket.

resource "random_password" "redacted_jwt" {
  length  = 64
  special = false
}

resource "random_password" "ghostchat_jwt" {
  length  = 64
  special = false
}

resource "random_password" "edge" {
  length  = 48
  special = false
}

locals {
  # GhostChat connects to the database named in the URI; with none, Mongo's default ("test") is
  # one its user may not write to. Add /ghostchat when the URI names no database.
  ghostchat_mongodb_uri = can(regex("^mongodb(\\+srv)?://[^/]+/[^?/]+", var.ghostchat_mongodb_uri)) ? var.ghostchat_mongodb_uri : replace(var.ghostchat_mongodb_uri, "/^(mongodb(\\+srv)?://[^/?]+)/?/", "$${1}/ghostchat")

  secrets = {
    lighthouse-db-owner      = { service = "lighthouse", value = neon_project.lighthouse.database_password }
    lighthouse-db-app        = { service = "lighthouse", value = random_password.lighthouse_app.result }
    lighthouse-github-secret = { service = "lighthouse", value = var.github_client_secret }
    redacted-db-owner        = { service = "redacted", value = neon_project.redacted.database_password }
    redacted-db-app          = { service = "redacted", value = random_password.redacted_app.result }
    ghostchat-mongodb-uri    = { service = "ghostchat", value = local.ghostchat_mongodb_uri }
  }
}

resource "google_secret_manager_secret" "s" {
  for_each  = local.secrets
  secret_id = each.key
  replication {
    auto {}
  }
  depends_on = [google_project_service.api]
}

resource "google_secret_manager_secret_version" "s" {
  for_each    = local.secrets
  secret      = google_secret_manager_secret.s[each.key].id
  secret_data = each.value.value
}

resource "google_secret_manager_secret_iam_member" "reader" {
  for_each  = local.secrets
  secret_id = google_secret_manager_secret.s[each.key].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.run[each.value.service].email}"
}
