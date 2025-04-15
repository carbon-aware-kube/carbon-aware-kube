# GitHub Actions Workflows for Carbon Aware Scheduler

This directory contains GitHub Actions workflows for building and publishing the Carbon Aware Scheduler Docker image and Helm chart to Google Cloud Artifact Registry.

## Workflows

1. **docker-build-push.yml**: Builds and pushes the Docker image
2. **helm-chart-publish.yml**: Packages and pushes the Helm chart

## Required GitHub Secrets

To use these workflows, you need to set up the following secrets in your GitHub repository. These values are available as outputs from the Terraform configuration in the `terraform/artifact-registry` directory:

- `GCP_PROJECT_ID`: Your Google Cloud project ID
- `GCP_ARTIFACT_REGISTRY_URL`: The URL of your Artifact Registry (from Terraform output `artifact_registry_repository_url`)
- `GCP_WORKLOAD_IDENTITY_PROVIDER`: The full identifier of your Workload Identity Provider (from Terraform output `workload_identity_provider`)
- `GCP_SERVICE_ACCOUNT_EMAIL`: The email address of your Google Cloud service account (from Terraform output `service_account_email`)

## Setting Up GCP for GitHub Actions

The required GCP resources for GitHub Actions integration are automatically created by the Terraform configuration in the `terraform/artifact-registry` directory. To set up these resources:

1. Navigate to the `terraform/artifact-registry` directory
2. Update `terraform.tfvars` with your project details, including:
   - `project_id`: Your GCP project ID
   - `project_number`: Your GCP project number
   - `github_repository`: Your GitHub repository in the format `owner/repo`
3. Run Terraform to create the resources:

```bash
terraform init
terraform apply
```

4. After Terraform has created the resources, retrieve the outputs to set up your GitHub secrets:

```bash
terraform output service_account_email
terraform output workload_identity_provider
terraform output artifact_registry_repository_url
```

## Release Process

### Creating a Release

1. Create a new GitHub release with a semantic version tag (e.g., `v1.0.0`)
2. The workflows will automatically:
   - Build and push the Docker image with the release version tag
   - Update the Helm chart version to match the release
   - Package and push the Helm chart to Artifact Registry

### Development Builds

Pushes to the main/master branch will trigger:
- Docker image builds tagged with `dev` and the short commit SHA
- Helm chart packaging and pushing with a `dev-{commit-sha}` tag

## Using the Published Artifacts

### Docker Image

```bash
# Pull the Docker image (replace with your actual registry URL from Terraform output)
docker pull REGISTRY_URL/carbon-aware-scheduler:latest
```

### Helm Chart

```bash
# Add the Helm repository (replace with your actual registry URL from Terraform output)
helm registry login REGISTRY_URL

# Install the chart
helm install ca-scheduler oci://REGISTRY_URL/charts/carbon-aware-scheduler --version 1.0.0
```
