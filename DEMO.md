# carbon-aware-kube Demo

## Installation
To install the `carbon-aware-kube` operator, run the following:

```bash
helm repo add carbon-aware-kube https://carbon-aware-kube.dev/charts
helm upgrade --install carbon-aware-kube carbon-aware-kube/carbon-aware-kube -n carbon-aware-kube
```

To check the status of the operator, run:

```bash
kubectl get pods -n carbon-aware-kube
```

## Usage

1. Create a `CarbonAwareJob`

```bash
export JOB_NAME=climate-hacks-demo-$(date +%s)
export NAMESPACE=climate-hacks-demo
echo 'apiVersion: batch.carbon-aware-kube.dev/v1alpha1
kind: CarbonAwareJob
metadata:
  name: $JOB_NAME
  namespace: $NAMESPACE
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
          restartPolicy: Never' | envsubst | kubectl apply -f -
```

2. Check the status of the `CarbonAwareJob`

```bash
kubectl describe carbonawarejob $JOB_NAME -n $NAMESPACE
```

3. Delete the `CarbonAwareJob`

```bash
kubectl delete carbonawarejob $JOB_NAME -n $NAMESPACE
```