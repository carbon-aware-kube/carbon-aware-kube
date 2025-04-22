# carbon-aware-kube
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/carbon-aware-kube)](https://artifacthub.io/packages/search?repo=carbon-aware-kube)

A project to minimize carbon impact from Kubernetes jobs. Following the philosophy of shift-left, `carbon-aware-kube` implements a set of custom resources which allow users to easily build application with carbon awareness.

Currently, this project is in alpha and only supports a single custom resource: `CarbonAwareJob` (`CarbonAwareCronJob` coming soon). Future releases will add support for more custom resources.

## Installation

### Prerequisites

1. A Kubernetes cluster
2. A [WattTime API key](https://docs.watttime.org/)

### Helm Install (development)

First, create a namespace for the operator:
```bash
kubectl create namespace carbon-aware-kube
```

Next, create a secret for the WattTime API key:
```bash
kubectl create secret generic carbon-aware-watttime --from-literal=username=${WATTIME_USERNAME} --from-literal=password=${WATTIME_PASSWORD} -n carbon-aware-kube
```

Next, install the operator:
The following command will install the operator in development mode, which will use the `dev` tag for both the operator and scheduler images:
```bash
helm upgrade --install carbon-aware-kube operator/helm/carbon-aware-kube -n carbon-aware-kube \
  --set image.tag=dev \
  --set carbon-aware-scheduler.image.tag=dev \
  --set carbon-aware-scheduler.watttime.existingSecret.enabled=true \
  --set carbon-aware-scheduler.watttime.existingSecret.name=carbon-aware-watttime
```

## Usage

To create a `CarbonAwareJob`, you can do the following:
```bash
kubectl apply -f - <<EOF
apiVersion: batch.carbon-aware-kube.dev/v1alpha1
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