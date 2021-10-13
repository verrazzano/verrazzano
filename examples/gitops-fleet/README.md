# GitOps with Verrazzano using Fleet

## Create GitRepo instance

Git repositories are register by creating a `GitRepo`. Apply this yaml to deploy the hello-helidon example via GitOps with fleet

```yaml
kind: GitRepo
apiVersion: fleet.cattle.io/v1alpha1
metadata:
  name: hello-helidon
  namespace: fleet-local
spec:

  repo: https://github.com/verrazzano/zerrazzano

  branch: btidd/VZ-3660

  paths:
  - examples/gitops-fleet
  - examples/hello-helidon

  targets:
  - name: local
    clusterSelector:
      matchLabels:
        name: local
```