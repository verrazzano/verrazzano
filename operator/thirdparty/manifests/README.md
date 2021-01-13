# Manifests

The external Kubernetes manifests used by Verrazzano.

## Cert-Manager

The `cert-manager` folder content was created by running the following commands:

```
export CERT_MANAGER_RELEASE=0.13
curl -L -o "00-crds.yaml" \
    "https://raw.githubusercontent.com/jetstack/cert-manager/release-${CERT_MANAGER_RELEASE}/deploy/manifests/00-crds.yaml"
```


