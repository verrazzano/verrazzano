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
    KIND_IMAGE="v1.17.11@sha256:5240a7a2c34bf241afb54ac05669f8a46661912eab05705d660971eeb12f6555"
  elif [ ${K8S_VERSION} == 1.19 ]; then
    KIND_IMAGE="v1.19.1@sha256:98cf5288864662e37115e362b23e4369c8c4a408f99cbc06e58ac30ddc721600"
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
  sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.Networks[].IPAddress' | sed 's/"//g'`:6443|g" ${KUBECONFIG}
  cat ${KUBECONFIG} | grep server
  $(docker network connect kind $(docker ps | grep "jenkins-runner" | awk '{ print $1 }'))
}

create_kind_cluster
