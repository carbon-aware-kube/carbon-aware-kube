# carbon-aware-kube
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/carbon-aware-kube)](https://artifacthub.io/packages/search?repo=carbon-aware-kube)

A project to minimize carbon impact from Kubernetes jobs. Following the philosophy of shift-left, `carbon-aware-kube` implements a set of custom resources which allow users to easily build application with carbon awareness.

Currently, this project is in alpha and only supports a single custom resource: `CarbonAwareJob` (`CarbonAwareCronJob` coming soon). Future releases will add support for more custom resources.

## Installation

### Prerequisites

1. A Kubernetes cluster

### Helm Install (development)

The following command will install the operator in development mode, which will use the `dev` tag for both the operator and scheduler images:
```bash
helm repo add carbon-aware https://carbon-aware.github.io/charts
helm upgrade --install carbon-aware-kube helm/carbon-aware-kube -n carbon-aware-kube --create-namespace
```

## Usage

To create a `CarbonAwareJob`, you can do the following:
```bash
kubectl apply -f - <<EOF
apiVersion: batch.carbonaware.dev/v1alpha1
kind: CarbonAwareJob
metadata:
  name: example
spec:
  maxDelay: "1h"
  maxDuration: "10m"
  template:
    spec:
      template:
        spec:
          containers:
            - name: example
              image: busybox
              command: ["sh", "-c", "echo Hello World"]
EOF
```

As opposed to:
```bash
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: example
spec:
  template:
    spec:
      containers:
        - name: example
          image: busybox
          command: ["sh", "-c", "echo Hello World"]
EOF
```


## Contributing

Contributions are extremely welcome! Please open an issue or submit a pull request.

## License

[Apache License 2.0](LICENSE)