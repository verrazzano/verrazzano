#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo "Installing cert-manager via helm chart"
echo "Setting clusterResourceNamespace to $CLUSTER_RESOURCE_NAMESPACE"

if [ -z "$(kubectl get ns | grep my-cert-manager)" ]
then
  kubectl create ns my-cert-manager
fi

if [ $CLUSTER_RESOURCE_NAMESPACE != my-cert-manager ]
then
  if [ -z "$(kubectl get ns | grep $CLUSTER_RESOURCE_NAMESPACE)" ]
  then
    kubectl create ns $CLUSTER_RESOURCE_NAMESPACE
  fi
fi

controllerTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-controller")' | jq .tag -r)
cainjectorTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-cainjector")' | jq .tag -r)
webhookTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-webhook")' | jq .tag -r)
helm upgrade cert-manager -n my-cert-manager platform-operator/thirdparty/charts/cert-manager \
--set image.repository=ghcr.io/verrazzano/cert-manager-controller --set image.tag=${controllerTag}  \
--set cainjector.image.repository=ghcr.io/verrazzano/cert-manager-cainjector --set cainjector.image.tag=${cainjectorTag}  \
--set webhook.image.repository=ghcr.io/verrazzano/cert-manager-webhook --set webhook.image.tag=${webhookTag} \
--set startupapicheck.enabled=false --set clusterResourceNamespace=${CLUSTER_RESOURCE_NAMESPACE} \
--set installCRDs=true --install

echo "ensure cert-manager using ghcr.io images"
if [ ! -z "$(kubectl get po -n my-cert-manager -o yaml | grep quay.io)" ]
then
  kubectl get po -n my-cert-manager -o yaml | grep quay.io
  exit 1
fi

kubectl get pods -n my-cert-manager

echo "Installing ingress-nginx via helm chart"

if [ -z "$(kubectl get ns | grep ingress-nginx)" ]
then
  kubectl create ns ingress-nginx
fi

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
