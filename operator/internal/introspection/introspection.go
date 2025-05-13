package cloudinfo

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CloudEnvironment struct {
	Provider string // e.g. "gcp", "aws", "azure", "unknown"
	Region   string
	Zone     string
}

// DetectCloudEnvironment inspects node labels to infer cloud provider and region.
// For GCP, this looks at topology.kubernetes.io/region and zone.
func DetectCloudEnvironment(ctx context.Context, k8sClient client.Client) (*CloudEnvironment, error) {
	nodeList := &corev1.NodeList{}
	if err := k8sClient.List(ctx, nodeList); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	if len(nodeList.Items) == 0 {
		return nil, fmt.Errorf("no nodes found in cluster")
	}

	// Use first node as representative
	node := nodeList.Items[0]
	labels := node.Labels

	region := labels["topology.kubernetes.io/region"]
	zone := labels["topology.kubernetes.io/zone"]

	provider := detectProvider(labels)

	return &CloudEnvironment{
		Provider: provider,
		Region:   region,
		Zone:     zone,
	}, nil
}

func detectProvider(labels map[string]string) string {
	switch {
	case labels["cloud.google.com/gke-nodepool"] != "":
		return "gcp"
	case labels["eks.amazonaws.com/nodegroup"] != "":
		return "aws"
	case labels["kubernetes.azure.com/role"] != "":
		return "azure"
	default:
		return "unknown"
	}
}
