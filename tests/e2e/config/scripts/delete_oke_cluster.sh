#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

set +e

# delete LB given its IP address
function delete_lb() {
  if [ ! -z "$1" ]; then
    echo "trying to delete LB with IP address $1"
    local LB_ID=$(oci lb load-balancer list --compartment-id="${TF_VAR_compartment_id}" --region="${TF_VAR_region}" \
    | jq -r --arg IP_ADDRESS "$1" '.data[] | select(."ip-addresses"[0]."ip-address" == ($IP_ADDRESS)) | .id')
    if [ ! -z "${LB_ID}" ]; then
      echo "LB to be deleted has ID ${LB_ID}"
      oci lb load-balancer delete --force --load-balancer-id ${LB_ID} --region ${TF_VAR_region}
    fi
  fi
}

# get LB IP
ISTIO_IP=$(kubectl get svc istio-ingressgateway -n istio-system -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The LB IP address for istio-ingressgateway is ${ISTIO_IP}"
NGINX_IP=$(kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The LB IP address for nginx-controller is ${NGINX_IP}"

# uninstalling some services to release resources
helm uninstall keycloak --namespace keycloak --timeout 5m0s
helm uninstall verrazzano --namespace verrazzano-system --timeout 5m0s
helm uninstall ingress-controller --namespace ingress-nginx --timeout 5m0s
kubectl delete service istio-ingressgateway -n istio-system --timeout 2m
kubectl delete deployment istio-ingressgateway -n istio-system --timeout 2m

# delete LB if they are deleted by deleting the services
delete_lb "${ISTIO_IP}"
delete_lb "${NGINX_IP}"

# clean up PVC
kubectl delete pvc --all -n keycloak --timeout 2m
# wait until OKE cleans up the deleted resources
timeout 2m bash -c 'until kubectl get pv -A 2>&1 | grep "No resources found"; do sleep 10; done'

# log what still exists, just in case
kubectl get pvc,pv,svc -A

# delete the OKE cluster
cd $SCRIPT_DIR/terraform/cluster
./delete-cluster.sh
