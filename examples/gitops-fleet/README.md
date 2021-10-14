# GitOps with Verrazzano using Fleet
## Hello World Helidon

The Hello World Helidon example is a [Helidon](https://helidon.io/#/)-based service that returns a "Hello World" response when invoked.

For more information and the source code of this application, see the [Verrazzano Examples](https://github.com/verrazzano/examples).

## Create hello-helidon namespace
```
kubectl create namespace hello-helidon
kubectl label namespace hello-helidon verrazzano-managed=true istio-injection=enabled
```

## Create GitRepo instance

Git repositories are register by creating a `GitRepo`. Apply this yaml to deploy the hello-helidon example via GitOps with Fleet

```yaml
kind: GitRepo
apiVersion: fleet.cattle.io/v1alpha1
metadata:
  name: hello-helidon
  namespace: fleet-local
spec:

  repo: https://github.com/verrazzano/verrazzano

  revision: fleet-blog
  
  paths:
  - examples/gitops-fleet
  - examples/hello-helidon

  targets:
  - name: local
    clusterSelector:
      matchLabels:
        name: local
```
Copyright (c) 2021, Oracle and/or its affiliates.
