output "urls" {
  value = { for k, host in local.hosts : k => "https://${host}" }
}

output "origins" {
  description = "The Cloud Run URLs behind the edge (Lighthouse refuses direct visits)."
  value       = { for k in local.services : k => google_cloud_run_v2_service.app[k].uri }
}

# What each repository's release workflow needs (as GitHub Actions variables, not secrets: none of
# these is a credential).
output "github_actions" {
  value = {
    GCP_WORKLOAD_IDENTITY_PROVIDER = google_iam_workload_identity_pool_provider.github.name
    GCP_SERVICE_ACCOUNT            = google_service_account.deployer.email
    GCP_REGION                     = var.region
    GCP_REGISTRY                   = "${var.region}-docker.pkg.dev/${var.gcp_project}/${google_artifact_registry_repository.images.repository_id}"
  }
}
