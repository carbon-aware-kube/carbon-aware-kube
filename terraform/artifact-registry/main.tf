resource "google_project_service" "artifactregistry" {
  project = var.project_id
  service = "artifactregistry.googleapis.com"

  # Keep the API enabled even if Terraform destroys this resource
  disable_on_destroy = false 
}

resource "google_project_service" "iam" {
  project = var.project_id
  service = "iam.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "iamcredentials" {
  project = var.project_id
  service = "iamcredentials.googleapis.com"
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "main" {
  for_each      = { for repo in var.repositories : repo.repository_id => repo }
  project       = var.project_id
  location      = var.region
  repository_id = each.value.repository_id
  description   = each.value.description
  format        = each.value.format

  labels = {
    environment = "shared"
    managed-by  = "terraform"
  }

  depends_on = [
    google_project_service.artifactregistry
  ]
}

# GitHub Actions Service Account
resource "google_service_account" "github_actions" {
  project      = var.project_id
  account_id   = "github-actions-sa"
  display_name = "GitHub Actions Service Account"
  description  = "Service account for GitHub Actions to push to Artifact Registry"
  
  depends_on = [
    google_project_service.iam
  ]
}

# Grant Artifact Registry Writer role to the service account
resource "google_project_iam_member" "github_actions_artifact_registry" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.github_actions.email}"
}

# Workload Identity Pool
resource "google_iam_workload_identity_pool" "github_actions" {
  project                   = var.project_id
  workload_identity_pool_id = "github-actions-pool"
  display_name              = "GitHub Actions Pool"
  description               = "Identity pool for GitHub Actions"
  
  depends_on = [
    google_project_service.iam
  ]
}

# Workload Identity Provider
resource "google_iam_workload_identity_pool_provider" "github_actions" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_actions.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-actions-provider"
  display_name                       = "GitHub Actions Provider"
  description                        = "OIDC identity pool provider for GitHub Actions"
  
  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.aud"        = "assertion.aud"
    "attribute.repository" = "assertion.repository"
  }
  
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  attribute_condition = "attribute.repository == '${var.github_repository}'"
}

# Allow GitHub Actions to impersonate the service account
resource "google_service_account_iam_binding" "workload_identity_binding" {
  service_account_id = google_service_account.github_actions.name
  role               = "roles/iam.workloadIdentityUser"
  
  members = [
    "principalSet://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${google_iam_workload_identity_pool.github_actions.workload_identity_pool_id}/*"
  ]
}
