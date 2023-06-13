# Helm Charts

The Helm charts used by Verrazzano.

## Istio

The `istio` and `istio-init` folders were created by running the following commands:

```shell
export ISTIO_HELM_CHART_VERSION=1.4.6
rm -rf istio
rm -rf istio-init
helm repo add istio.io https://storage.googleapis.com/istio-release/releases/${ISTIO_HELM_CHART_VERSION}/charts
helm repo update
helm fetch istio.io/istio --untar=true --version=${ISTIO_HELM_CHART_VERSION}
helm fetch istio.io/istio-init --untar=true --version=${ISTIO_HELM_CHART_VERSION}
```

## Nginx

The `nginx-ingress` folder was created by running the following commands:

```shell
export NGINX_HELM_CHART_VERSION=3.30.0
rm -rf ingress-nginx
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm fetch ingress-nginx/ingress-nginx --untar=true --version=${NGINX_HELM_CHART_VERSION}
```

## Cert-Manager

The `cert-manager` folder was created by running the following commands:

```shell
export CERT_MANAGER_CHART_VERSION=v0.13.1
rm -rf cert-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm fetch jetstack/cert-manager --untar=true --version=${CERT_MANAGER_CHART_VERSION}
```

## Cert-Manager Webhook OCI

The `cert-manager-webhook-oci` is maintained by Verrazzano at https://github.com/verrazzano/cert-manager-webhook-oci.
It is copied from https://github.com/verrazzano/cert-manager-webhook-oci/tree/main/deploy/cert-manager-webhook-oci.

Changes to this chart should be managed in https://github.com/verrazzano/cert-manager-webhook-oci and should be
copied here.

## Rancher

The `rancher` folder was created by running the following commands:

```shell
export RANCHER_CHART_VERSION=v2.7.2
rm -rf rancher
helm repo add rancher-stable https://releases.rancher.com/server-charts/stable
helm repo update
helm fetch rancher-stable/rancher --untar=true --version=${RANCHER_CHART_VERSION}
```

## MySQL

The `mysql` folder was created by running the following commands:

```shell
export MYSQL_CHART_VERSION=2.0.8
rm -rf mysql
helm repo add mysql-operator https://mysql.github.io/mysql-operator/
helm repo update
helm fetch mysql-operator/mysql-innodbcluster --untar=true --version=${MYSQL_CHART_VERSION}
```

## MySQL Operator

The `mysql` folder was created by running the following commands:

```shell
export MYSQL_OPERATOR_CHART_VERSION=2.0.8
rm -rf mysql-operator
helm repo add mysql-operator https://mysql.github.io/mysql-operator/
helm repo update
helm fetch mysql-operator/mysql-operator --untar=true --version=${MYSQL_OPERATOR_CHART_VERSION}
```

## KeyCloak

The `keycloak` folder was created by running the following commands:

```shell
export KEYCLOAK_CHART_VERSION=8.2.2
rm -rf keycloak
helm repo add codecentric https://codecentric.github.io/helm-charts
helm repo update
helm fetch codecentric/keycloak --untar=true --version=${KEYCLOAK_CHART_VERSION}
```

## External DNS

The `external-dns` folder was created by running the following commands:

```shell
export EXTERNAL_DNS_CHART_VERSION=2.20.0
rm -rf external-dns
helm repo add stable https://charts.helm.sh/stable
helm repo update
helm fetch stable/external-dns --untar=true --version=${EXTERNAL_DNS_CHART_VERSION}
```

### WLS Operator

The `wls-operator` folder was created by running the following commands:

```shell
export WEBLOGIC_OPERATOR_CHART_REPO=https://oracle.github.io/weblogic-kubernetes-operator/charts
export WEBLOGIC_OPERATOR_CHART_VERSION=4.0.6
rm -rf weblogic-operator
helm repo add weblogic-operator ${WEBLOGIC_OPERATOR_CHART_REPO}
helm repo update
helm fetch weblogic-operator/weblogic-operator --untar=true --version=${WEBLOGIC_OPERATOR_CHART_VERSION}
```

### Coherence Operator

The `coherence-operator` folder was created by running the following commands:

```shell
export COHERENCE_OPERATOR_CHART_REPO=https://oracle.github.io/coherence-operator/charts
export COHERENCE_OPERATOR_CHART_VERSION=3.1.5
rm -rf coherence-operator
helm repo add coherence ${COHERENCE_OPERATOR_CHART_REPO}
helm repo update
helm fetch coherence/coherence-operator --untar=true --version=${COHERENCE_OPERATOR_CHART_VERSION}
```

### OAM Runtime

The `oam-kubernetes-runtime` folder was created by running the following commands:

```shell
export OAM_RUNTIME_CHART_REPO=https://charts.crossplane.io/master/
export OAM_RUNTIME_CHART_VERSION=0.3.0
rm -rf oam-kubernetes-runtime
helm repo add crossplane-master ${OAM_RUNTIME_CHART_REPO}
helm repo update
helm fetch crossplane-master/oam-kubernetes-runtime --untar=true --version=${OAM_RUNTIME_CHART_VERSION}
```

### Verrazzano Application Operator

The `verrazzano-application-operator` folder was created manually.

### Velero

The `velero` folder was created by running the following commands:

```shell
export VELERO_HELM_CHART_VERSION=2.30.0
rm -rf velero
helm repo add vmware-tanzu https://vmware-tanzu.github.io/helm-charts
helm repo update
helm fetch vmware-tanzu/velero --untar=true --version=${VELERO_HELM_CHART_VERSION}
```

### Rancher Backup Operator

The `rancher-charts` folder was created by running the following commands:

```shell
helm repo add rancher-charts https://charts.rancher.io
rm -rf rancher-backup-crd
rm -rf rancher-backup
helm repo update
helm fetch rancher-charts/rancher-backup-crd --untar=true
helm fetch rancher-charts/rancher-backup --untar=true
```

### Jaeger Operator

The `jaegertracing/jaeger-operator` folder was created by running the following commands:

```shell
export JAEGER_OPERATOR_CHART_REPO=https://jaegertracing.github.io/helm-charts
export JAEGER_OPERATOR_CHART_VERSION=2.41.0
rm -rf jaegertracing/jaeger-operator
mkdir -p jaegertracing
cd jaegertracing
helm repo add jaegertracing ${JAEGER_OPERATOR_CHART_REPO}
helm repo update
helm fetch jaegertracing/jaeger-operator --untar=true --version=${JAEGER_OPERATOR_CHART_VERSION}
```

### Kiali Server

The `kiali-server` folder was created by running the following commands:

```shell
export KIALI_SERVER_CHART_REPO=https://kiali.org/helm-charts
export KIALI_SERVER_CHART_VERSION=1.57.1
helm repo add kiali ${KIALI_SERVER_CHART_REPO}
helm repo update
rm -rf kiali-server
helm fetch kiali/kiali-server --untar=true --version=${KIALI_SERVER_CHART_VERSION}
```

### Argo CD

The `argo-cd` folder was created by running the following commands:

```shell
export ARGOCD_CHART_VERSION=5.34.3
helm repo add argocd https://argoproj.github.io/argo-helm
helm repo update
rm -rf argo-cd
helm fetch argocd/argo-cd --untar=true --version=${ARGOCD_CHART_VERSION}
```

### Prometheus Node Exporter

The `prometheus-community/prometheus-node-exporter` folder was created by running the followiong commands:

```shell
export PROMETHEUS_NODE_EXPORTER_CHART_VERSION=3.1.0
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
rm -rf prometheus-community/prometheus-node-exporter 
helm fetch prometheus-community/prometheus-node-exporter --untar=true --version=${PROMETHEUS_NODE_EXPORTER_CHART_VERSION}
```

### OpenSearch Operator

The `opensearch-operator` folder was created by running the following commands:

```shell
export OPENSEARCH_OPERATOR_CHART_VERSION=2.2.0
helm repo add opensearch-operator https://opster.github.io/opensearch-k8s-operator/
helm repo update
rm -rf opensearch-operator
helm fetch opensearch-operator/opensearch-operator --untar=true --version=${OPENSEARCH_OPERATOR_CHART_VERSION}
```

### Fluent Operator

The `fluent-operator` folder was created by running the following commands:

```shell
export FLUENT_OPERATOR_CHART_VERSION=2.2.0
helm repo add fluent https://fluent.github.io/helm-charts
helm repo update
rm -rf fluent-operator
helm fetch fluent/fluent-operator --untar=true --version=${FLUENT_OPERATOR_CHART_VERSION}
```

