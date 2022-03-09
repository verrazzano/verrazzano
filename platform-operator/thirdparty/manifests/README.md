# Manifests

The external Kubernetes manifests used by Verrazzano.

## Cert-Manager

The `cert-manager` folder content was created by running the following commands:

```
export CERT_MANAGER_RELEASE=1.2.0
curl -L -o "cert-manager.crds.yaml" \
    "https://github.com/jetstack/cert-manager/releases/download/v${CERT_MANAGER_RELEASE}/cert-manager.crds.yaml"
```

## Operator-Lifecycle-Manager

The `operator-lifecycle-manager` folder content was created by running the following commands:

```
export OPERATOR_LIFECYCLE_MANAGER_RELEASE=0.19.1
curl -L -o "operator-lifecycle-manager.crds.yaml" \
    https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/v${OPERATOR_LIFECYCLE_MANAGER_RELEASE}/deploy/upstream/quickstart/crds.yaml
```
