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
CLEANUP_KIND_CONTAINERS=${5:-true}
CONNECT_JENKINS_RUNNER_TO_NETWORK=${6:-true}
KIND_AT_CACHE=${7:-false}
SETUP_CALICO=${8:-false}
KIND_AT_CACHE_NAME=${9:-"NONE"}
KIND_IMAGE=""
CALICO_SUFFIX=""

create_kind_cluster() {
  if [ ${K8S_VERSION} == 1.17 ]; then
    KIND_IMAGE="v1.17.11@sha256:5240a7a2c34bf241afb54ac05669f8a46661912eab05705d660971eeb12f6555"
  elif [ ${K8S_VERSION} == 1.18 ]; then
    KIND_IMAGE="v1.18.8@sha256:f4bcc97a0ad6e7abaf3f643d890add7efe6ee4ab90baeb374b4f41a4c95567eb"
  elif [ ${K8S_VERSION} == 1.19 ]; then
    KIND_IMAGE="v1.19.1@sha256:98cf5288864662e37115e362b23e4369c8c4a408f99cbc06e58ac30ddc721600"
  elif [ ${K8S_VERSION} == 1.20 ]; then
    KIND_IMAGE="v1.20.2@sha256:8f7ea6e7642c0da54f04a7ee10431549c0257315b3a634f6ef2fecaaedb19bab"
  else
    echo "ERROR: Invalid value for Kubernetes Version ${K8S_VERSION}."
    exit 1
  fi

  if [ $SETUP_CALICO == true ] ; then
    CALICO_SUFFIX="-calico"
  fi

  # Set CLEANUP_KIND_CONTAINERS to true, while second cluster and onwards
  if [ ${CLEANUP_KIND_CONTAINERS} == true ]; then
    cd ${PLATFORM_OPERATOR_DIR}/build/scripts
    ./cleanup.sh ${CLUSTER_NAME}
  else
    echo "Delete the cluster and kube config in multi-cluster environment"
    kind delete cluster --name ${CLUSTER_NAME}
    if [ -f "${KUBECONFIG}" ]
    then
      echo "Deleting the kubeconfig '${KUBECONFIG}' ..."
      rm ${KUBECONFIG}
    fi
  fi

  export KUBECONFIG=$KUBECONFIG
  echo "Kubeconfig ${KUBECONFIG}"
  echo "KIND Image : ${KIND_IMAGE}"
  cd ${SCRIPT_DIR}/
  KIND_CONFIG_FILE=kind-config${CALICO_SUFFIX}.yaml
  if [ ${KIND_AT_CACHE} == true ]; then
    if [ ${KIND_AT_CACHE_NAME} != "NONE" ]; then
      # If a cache name was specified, replace the at_test cache name with the one specified (this is used only
      # for multi-cluster tests at the moment)
      sed "s;v8o_cache/at_tests;v8o_cache/${KIND_AT_CACHE_NAME};g" kind-config-ci${CALICO_SUFFIX}.yaml > kind-config-ci${CALICO_SUFFIX}_${KIND_AT_CACHE_NAME}.yaml
      KIND_CONFIG_FILE=kind-config-ci${CALICO_SUFFIX}_${KIND_AT_CACHE_NAME}.yaml
    else
      # If no cache name specified use at_tests cache
      KIND_CONFIG_FILE=kind-config-ci${CALICO_SUFFIX}.yaml
    fi
  fi
  echo "Using ${KIND_CONFIG_FILE}"
  sed -i "s/KIND_IMAGE/${KIND_IMAGE}/g" ${KIND_CONFIG_FILE}
  HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster -v 1 --name ${CLUSTER_NAME} --config=${KIND_CONFIG_FILE}
  kubectl config set-context kind-${CLUSTER_NAME}
  sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.Networks[].IPAddress' | sed 's/"//g'`:6443|g" ${KUBECONFIG}
  cat ${KUBECONFIG} | grep server
  if [ ${CONNECT_JENKINS_RUNNER_TO_NETWORK} == true ]; then
    $(docker network connect kind $(docker ps | grep "jenkins-runner" | awk '{ print $1 }'))
  else
    echo "Ignore connecting jenkins-runner to a network."
  fi
}

create_kind_cluster
