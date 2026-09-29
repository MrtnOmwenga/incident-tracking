# Lighthouse's clock. On Cloud Run an idle instance gets no CPU, so Cloud Scheduler calls
# POST /internal/tick, which runs the checks that are due. The call carries a Google-signed
# identity token for this service account; Lighthouse checks it (internal/oidc).
resource "google_cloud_scheduler_job" "tick" {
  name             = "lighthouse-tick"
  region           = var.region
  description      = "Runs Lighthouse's due checks"
  schedule         = var.tick_schedule
  time_zone        = "Etc/UTC"
  attempt_deadline = "180s"

  # No retries (the default): the next tick is soon enough.

  http_target {
    http_method = "POST"
    uri         = "${google_cloud_run_v2_service.app["lighthouse"].uri}/internal/tick"
    oidc_token {
      service_account_email = google_service_account.scheduler.email
      audience              = "${local.public_url}/internal/tick" # Lighthouse's TICK_AUDIENCE default
    }
  }
  depends_on = [google_project_service.api]
}
