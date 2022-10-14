# Helm Charts

The Helm charts used by Verrazzano.

## Istio

The `istio` and `istio-init` folders were created by running the following commands:

```
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

```
export NGINX_HELM_CHART_VERSION=3.30.0
rm -rf ingress-nginx
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm fetch ingress-nginx/ingress-nginx --untar=true --version=${NGINX_HELM_CHART_VERSION}
```

## Cert-Manager

The `cert-manager` folder was created by running the following commands:

```
export CERT_MANAGER_CHART_VERSION=v0.13.1
rm -rf cert-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm fetch jetstack/cert-manager --untar=true --version=${CERT_MANAGER_CHART_VERSION}
```

## Rancher

The `rancher` folder was created by running the following commands:

```
export RANCHER_CHART_VERSION=v2.5.7
rm -rf rancher
helm repo add rancher-stable https://releases.rancher.com/server-charts/stable
helm repo update
helm fetch rancher-stable/rancher --untar=true --version=${RANCHER_CHART_VERSION}
```

## Mysql

The `mysql` folder was created by running the following commands:

```
export MYSQL_CHART_VERSION=1.6.9
rm -rf mysql
helm repo add stable https://charts.helm.sh/stable
helm repo update
helm fetch stable/mysql --untar=true --version=${MYSQL_CHART_VERSION}
```

## KeyCloak

The `keycloak` folder was created by running the following commands:

```
export KEYCLOAK_CHART_VERSION=8.2.2
rm -rf keycloak
helm repo add codecentric https://codecentric.github.io/helm-charts
helm repo update
helm fetch codecentric/keycloak --untar=true --version=${KEYCLOAK_CHART_VERSION}
```

## External DNS

The `external-dns` folder was created by running the following commands:

```
export EXTERNAL_DNS_CHART_VERSION=2.20.0
rm -rf external-dns
helm repo add stable https://charts.helm.sh/stable
helm repo update
helm fetch stable/external-dns --untar=true --version=${EXTERNAL_DNS_CHART_VERSION}
```

### WLS Operator

The `wls-operator` folder was created by running the following commands:

```
export WEBLOGIC_OPERATOR_CHART_REPO=https://oracle.github.io/weblogic-kubernetes-operator/charts
export WEBLOGIC_OPERATOR_CHART_VERSION=3.3.0
rm -rf weblogic-operator
helm repo add weblogic-operator ${WEBLOGIC_OPERATOR_CHART_REPO}
helm repo update
helm fetch weblogic-operator/weblogic-operator --untar=true --version=${WEBLOGIC_OPERATOR_CHART_VERSION}
```

### Coherence Operator

The `coherence-operator` folder was created by running the following commands:

```
export COHERENCE_OPERATOR_CHART_REPO=https://oracle.github.io/coherence-operator/charts
export COHERENCE_OPERATOR_CHART_VERSION=3.1.5
rm -rf coherence-operator
helm repo add coherence ${COHERENCE_OPERATOR_CHART_REPO}
helm repo update
helm fetch coherence/coherence-operator --untar=true --version=${COHERENCE_OPERATOR_CHART_VERSION}
```

### OAM Runtime

The `oam-kubernetes-runtime` folder was created by running the following commands:

```
export OAM_RUNTIME_CHART_REPO=https://charts.crossplane.io/master/
export OAM_RUNTIME_CHART_VERSION=0.3.0
rm -rf oam-kubernetes-runtime
helm repo add crossplane-master ${OAM_RUNTIME_CHART_REPO}
helm repo update
helm fetch crossplane-master/oam-kubernetes-runtime --untar=true --version=${OAM_RUNTIME_CHART_VERSION}
```

### Verrazzano Application Operator

The `verrazzano-application-operator` folder was created manually.


### Kiali Server

The `kiali-server` folder was created by running the following commands:

```shell
export KIALI_SERVER_CHART_REPO=https://kiali.org/helm-charts
export KIALI_SERVER_CHART_VERSION=1.42.0
helm repo add kiali ${KIALI_SERVER_CHART_REPO}
helm repo update
rm -rf kiali-server
helm fetch kiali/kiali-server --untar=true --version=${KIALI_SERVER_CHART_VERSION}
```