variable "project_id" {
  description = "The GCP project ID."
  type        = string
}

variable "project_number" {
  description = "The GCP project number."
  type        = string
}

variable "region" {
  description = "The GCP region where the Artifact Registry will be created."
  type        = string
  default     = "us-central1" # You can change this default
}

variable "github_repository" {
  description = "The GitHub repository in format 'owner/repo' that will be allowed to use the workload identity."
  type        = string
}

variable "repositories" {
  description = "The repositories to create in the Artifact Registry."
  type        = list(map(string))
  default     = [
    {
      repository_id = "carbon-aware-kube"
      format        = "DOCKER"
      description   = "Repository for carbon-aware-kube container images + Helm charts"
    },
  ]
}
