# Cloud Run can't pull from GHCR, so each repository's release workflow also pushes its image here.
# Only the newest few versions are kept: the free tier covers 0.5 GB of storage.
resource "google_artifact_registry_repository" "images" {
  repository_id = "portfolio"
  location      = var.region
  format        = "DOCKER"
  description   = "Images deployed to Cloud Run (also published, signed, on GHCR)."

  cleanup_policy_dry_run = false
  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 2 # about 290 MB for the three images: inside the 0.5 GB free tier
    }
  }
  cleanup_policies {
    id     = "delete-older"
    action = "DELETE"
    condition {
      older_than = "86400s"
    }
  }
  depends_on = [google_project_service.api]
}

resource "google_artifact_registry_repository_iam_member" "deployer_push" {
  repository = google_artifact_registry_repository.images.name
  location   = google_artifact_registry_repository.images.location
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.deployer.email}"
}
