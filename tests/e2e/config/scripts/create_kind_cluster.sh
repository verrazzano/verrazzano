#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
CLUSTER_NAME=$1
PLATFORM_OPERATOR_DIR=$2
KUBECONFIG=$3
K8S_VERSION=$4
KIND_IMAGE=""

create_kind_cluster() {
  if [ ${K8S_VERSION} == 1.17 ]; then
    KIND_IMAGE="v1.17.0@sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62"
  elif [ ${K8S_VERSION} == 1.18 ]; then
    KIND_IMAGE="v1.18.15@sha256:5c1b980c4d0e0e8e7eb9f36f7df525d079a96169c8a8f20d8bd108c0d0889cc4"
  elif [ ${K8S_VERSION} == 1.19 ]; then
    KIND_IMAGE="v1.19.7@sha256:a70639454e97a4b733f9d9b67e12c01f6b0297449d5b9cbbef87473458e26dca"
  elif [ ${K8S_VERSION} == 1.20 ]; then
    KIND_IMAGE="v1.20.2@sha256:8f7ea6e7642c0da54f04a7ee10431549c0257315b3a634f6ef2fecaaedb19bab"
  else
    echo "ERROR: Invalid value for Kubernetes Version ${K8S_VERSION}."
    exit 1
  fi
  
  cd ${PLATFORM_OPERATOR_DIR}/build/scripts
  ./cleanup.sh ${CLUSTER_NAME}

  cd ${SCRIPT_DIR}/
  echo "KinD Image : ${KIND_IMAGE}"
  sed -i "s/KIND_IMAGE/${KIND_IMAGE}/g" kind-config.yaml
  HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster -v 1 --name ${CLUSTER_NAME} --wait 5m --config=kind-config.yaml
  kubectl config set-context kind-${CLUSTER_NAME}
  sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.IPAddress' | sed 's/"//g'`:6443|g" ${KUBECONFIG}
}

create_kind_cluster
