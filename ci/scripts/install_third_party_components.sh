#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo "Installing cert-manager via helm chart"

kubectl create ns cert-manager
kubectl apply -f platform-operator/thirdparty/manifests/cert-manager

controllerTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-controller")' | jq .tag -r)
cainjectorTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-cainjector")' | jq .tag -r)
webhookTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-webhook")' | jq .tag -r)
helm upgrade cert-manager -n cert-manager platform-operator/thirdparty/charts/cert-manager \
--set image.repository=ghcr.io/verrazzano/cert-manager-controller --set image.tag=${controllerTag}  \
--set cainjector.image.repository=ghcr.io/verrazzano/cert-manager-cainjector --set cainjector.image.tag=${cainjectorTag}  \
--set webhook.image.repository=ghcr.io/verrazzano/cert-manager-webhook --set webhook.image.tag=${webhookTag} \
--set startupapicheck.enabled=false --install

echo "ensure cert-manager using ghcr.io images"
if [ ! -z "$(kubectl get po -n cert-manager -o yaml | grep quay.io)" ]
then
  kubectl get po -n cert-manager -o yaml | grep quay.io
  exit 1
fi

kubectl get pods -n cert-manager

echo "Installing ingress-nginx via helm chart"

kubectl create ns ingress-nginx

controllerTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="ingress-nginx")' | jq '.subcomponents[0].images[] | select(.image=="nginx-ingress-controller")' | jq .tag -r)
defaultBackendTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="ingress-nginx")' | jq '.subcomponents[0].images[] | select(.image=="nginx-ingress-default-backend")' | jq .tag -r)

helm upgrade ingress-controller -n ingress-nginx platform-operator/thirdparty/charts/ingress-nginx \
--set controller.image.digest="" --set controller.image.repository=ghcr.io/verrazzano/nginx-ingress-controller --set controller.image.tag=${controllerTag}  \
--set defaultBackend.image.repository=ghcr.io/verrazzano/nginx-ingress-default-backend --set defaultBackend.image.tag=${defaultBackendTag} \
--set defaultBackend.enabled=true --install

echo "ensure cert-manager using ghcr.io images"
if [ ! -z "$(kubectl get po -n cert-manager -o yaml | grep registry.k8s.io)" ]
then
  kubectl get po -n ingress-nginx -o yaml | grep registry.k8s.io
  exit 1
fi

kubectl get pods -n ingress-nginx
