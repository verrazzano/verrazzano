# Helm Charts

The Helm charts used by Verrazzano.

```
helm fetch istio.io/istio --untar=true --version=1.4.6
helm fetch istio.io/istio-init --untar=true --version=1.4.6

helm repo add stable https://charts.helm.sh/stable
helm fetch stable/nginx-ingress --untar=true --version=1.27.0

helm repo add jetstack https://charts.jetstack.io
helm fetch jetstack/cert-manager --untar=true --version=v0.13.1

helm repo add rancher-stable https://releases.rancher.com/server-charts/stable
helm fetch rancher-stable/rancher --untar=true --version=v2.4.3

helm repo add stable https://charts.helm.sh/stable
helm fetch stable/mysql --untar=true --version=1.6.9

helm repo add codecentric https://codecentric.github.io/helm-charts
helm fetch codecentric/keycloak --untar=true --version=8.2.2



helm pull stable/external-dns --untar=true --version=2.20.0

```

