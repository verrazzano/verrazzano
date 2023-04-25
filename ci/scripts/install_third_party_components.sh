#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo "Installing cert-manager via helm chart"
kubectl create ns cert-manager
controllerTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-controller")' | jq .tag -r)
cainjectorTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-cainjector")' | jq .tag -r)
webhookTag=$(cat platform-operator/verrazzano-bom.json | jq '.components[] | select(.name=="cert-manager")' | jq '.subcomponents[0].images[] | select(.image=="cert-manager-webhook")' | jq .tag -r)
helm upgrade cert-manager -n cert-manager platform-operator/thirdparty/charts/cert-manager \
--set image.repository=ghcr.io/verrazzano/cert-manager-controller --set image.tag=${controllerTag}  \
--set cainjector.image.repository=ghcr.io/verrazzano/cert-manager-cainjector --set cainjector.image.tag=${cainjectorTag}  \
--set webhook.image.repository=ghcr.io/verrazzano/cert-manager-webhook --set webhook.image.tag=${webhookTag} \
--set startupapicheck.enabled=false \
--install

echo "ensure cert-manager using ghcr.io images"
if [ ! -z "$(kubectl get po -n cert-manager -o yaml | grep quay.io)" ]
then
  kubectl get po -n cert-manager -o yaml | grep quay.io
  exit 1
fi

kubectl get pods -n cert-manager
