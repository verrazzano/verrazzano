#!/bin/bash

# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

if [ -z "${KUBECONFIG}" ] ; then
    echo "Environment variable KUBECONFIG must be set to run this script."
    exit 1
fi

# List additional resources
echo "kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o yaml"
kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o yaml
echo "-----------------------------------------------------"

echo "kubectl describe certificate tls-rancher-ingress -n cattle-system"
kubectl describe certificate tls-rancher-ingress -n cattle-system
echo "-----------------------------------------------------"

echo "kubectl describe certificate verrazzano-ca-certificate -n cert-manager"
kubectl describe certificate verrazzano-ca-certificate -n cert-manager
echo "-----------------------------------------------------"

echo "kubectl describe clusterissuer verrazzano-cluster-issuer"
kubectl describe clusterissuer verrazzano-cluster-issuer
echo "-----------------------------------------------------"

echo "kubectl describe secret tls-rancher-ingress -n cattle-system"
kubectl describe secret tls-rancher-ingress -n cattle-system
echo "-----------------------------------------------------"

echo "kubectl get service -n istio-system"
kubectl get service -n istio-system
echo "-----------------------------------------------------"
