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
export NGINX_HELM_CHART_VERSION=1.27.0
rm -rf nginx-ingress
helm repo add stable https://charts.helm.sh/stable
helm repo update
helm fetch stable/nginx-ingress --untar=true --version=${NGINX_HELM_CHART_VERSION}
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
export RANCHER_CHART_VERSION=v2.4.3
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

