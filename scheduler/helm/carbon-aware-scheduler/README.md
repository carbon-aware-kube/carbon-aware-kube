# Carbon Aware Scheduler Helm Chart

This Helm chart deploys the Carbon Aware Scheduler service to a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+
- WattTime API credentials

## Installing the Chart

To install the chart with the release name `carbon-scheduler`:

```bash
# Update values.yaml with your WattTime API credentials or use --set flags
helm install ca-scheduler ./charts/carbon-aware-scheduler
```

## Using Existing Secrets

If you prefer to manage your WattTime API credentials separately, you can create a Kubernetes secret and reference it in the Helm chart:

```bash
# Create a secret with your WattTime credentials
kubectl create secret generic watttime-credentials \
  --from-literal=username=your-username \
  --from-literal=password=your-password

# Install the chart referencing the existing secret
helm install carbon-scheduler ./charts/carbon-aware-scheduler \
  --set watttime.existingSecret.enabled=true \
  --set watttime.existingSecret.name=watttime-credentials
```

## Configuration

The following table lists the configurable parameters of the Carbon Aware Scheduler chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `carbon-aware-scheduler` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Kubernetes service port | `8080` |
| `resources.limits.cpu` | CPU resource limits | `500m` |
| `resources.limits.memory` | Memory resource limits | `512Mi` |
| `resources.requests.cpu` | CPU resource requests | `100m` |
| `resources.requests.memory` | Memory resource requests | `128Mi` |
| `watttime.existingSecret.enabled` | Use existing secret for WattTime credentials | `false` |
| `watttime.existingSecret.name` | Name of existing secret | `""` |
| `watttime.credentials.username` | WattTime API username | `""` |
| `watttime.credentials.password` | WattTime API password | `""` |
| `podAnnotations` | Pod annotations | `{}` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity | `{}` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example:

```bash
helm install ca-scheduler ./charts/carbon-aware-scheduler -f values.yaml
```
