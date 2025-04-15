output "service_account_email" {
  description = "The email address of the GitHub Actions service account"
  value       = google_service_account.github_actions.email
  sensitive   = false
}

output "workload_identity_provider" {
  description = "The full identifier of the Workload Identity Provider for GitHub Actions"
  value       = google_iam_workload_identity_pool_provider.github_actions.name
  sensitive   = false
}

output "artifact_registry_repository_url" {
  description = "The URL of the Artifact Registry repository"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.main["carbon-aware-kube"].repository_id}"
  sensitive   = false
}
