# Secrets live in Secret Manager; each service reads only its own (and its migration job, which
# runs as the same account). Generated values never leave Terraform state and Secret Manager.

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
  secrets = {
    lighthouse-db-owner      = { service = "lighthouse", value = neon_project.lighthouse.database_password }
    lighthouse-db-app        = { service = "lighthouse", value = random_password.lighthouse_app.result }
    lighthouse-github-secret = { service = "lighthouse", value = var.github_client_secret }
    lighthouse-edge          = { service = "lighthouse", value = random_password.edge.result }
    redacted-db-owner        = { service = "redacted", value = neon_project.redacted.database_password }
    redacted-db-app          = { service = "redacted", value = random_password.redacted_app.result }
    redacted-jwt             = { service = "redacted", value = random_password.redacted_jwt.result }
    ghostchat-mongodb-uri    = { service = "ghostchat", value = var.ghostchat_mongodb_uri }
    ghostchat-jwt            = { service = "ghostchat", value = random_password.ghostchat_jwt.result }
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
