# Manifests

The external Kubernetes manifests used by Verrazzano.

## Cert-Manager

The `cert-manager` folder content was created by running the following commands:

```
export CERT_MANAGER_RELEASE=1.2.0
curl -L -o "cert-manager.crds.yaml" \
    "https://github.com/jetstack/cert-manager/releases/download/v${CERT_MANAGER_RELEASE}/cert-manager.crds.yaml"
```

## Prometheus Operator

The `prometheus-operator` folder contains template Prometheus ServiceMonitor and PodMonitor resources that are applied during install and upgrade. The monitors
will cause Prometheus to collect metrics from Verrazzano system components.
