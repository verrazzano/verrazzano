#!/bin/bash

#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

kubectl -n kube-system create serviceaccount kubeconfig-sa
kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:kubeconfig-sa
# In k8s 1.24 and later, secret is not created for service account. Create a service account token secret and get the
#	token from the same.
TOKENNAME=kubeconfig-sa-token
kubectl -n kube-system apply -f <<EOF -
  apiVersion: v1
  kind: Secret
  metadata:
    name: ${TOKENNAME}
    annotations:
      kubernetes.io/service-account.name: kubeconfig-sa
  type: kubernetes.io/service-account-token
EOF
TOKEN=`kubectl -n kube-system get secret $TOKENNAME -o jsonpath='{.data.token}'| base64 --decode`
kubectl config set-credentials kubeconfig-sa --token=$TOKEN
kubectl config set-context --current --user=kubeconfig-sa
