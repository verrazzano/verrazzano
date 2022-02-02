#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

TOKENNAME=`kubectl -n kube-system get serviceaccount/kubeconfig-sa -o jsonpath='{.secrets[0].name}'`
TOKEN=`kubectl -n kube-system get secret $TOKENNAME -o jsonpath='{.data.token}'| base64 --decode`
kubectl config set-credentials kubeconfig-sa --token=$TOKEN
kubectl config set-context --current --user=kubeconfig-sa
