#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
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
NODE_COUNT=${10:-1}
KIND_IMAGE=""
CALICO_SUFFIX=""

create_kind_cluster() {
  if [ ${K8S_VERSION} == 1.21 ]; then
    KIND_IMAGE="v1.21.12@sha256:f316b33dd88f8196379f38feb80545ef3ed44d9197dca1bfd48bcb1583210207"
  elif [ ${K8S_VERSION} == 1.22 ]; then
    KIND_IMAGE="v1.22.9@sha256:8135260b959dfe320206eb36b3aeda9cffcb262f4b44cda6b33f7bb73f453105"
  elif [ ${K8S_VERSION} == 1.23 ]; then
    KIND_IMAGE="v1.23.6@sha256:b1fa224cc6c7ff32455e0b1fd9cbfd3d3bc87ecaa8fcb06961ed1afb3db0f9ae"
  elif [ ${K8S_VERSION} == 1.24 ]; then
    KIND_IMAGE="v1.24.0@sha256:0866296e693efe1fed79d5e6c7af8df71fc73ae45e3679af05342239cdc5bc8e"
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
  # List the permissions of /dev/null.  We have seen a failure where `docker ps` gets an operation not permitted error.
  # Listing the permissions will help to analyze what is wrong, if we see the failure again.
  echo "Listing permissions for /dev/null"
  ls -l /dev/null
  echo "Using ${KIND_CONFIG_FILE}"
  for (( n=2; n<=${NODE_COUNT}; n++ ))
  do
    echo "  - role: worker" >> ${KIND_CONFIG_FILE}
    echo "    image: kindest/node:KIND_IMAGE" >> ${KIND_CONFIG_FILE}
  done
  sed -i "s/KIND_IMAGE/${KIND_IMAGE}/g" ${KIND_CONFIG_FILE}
  cat ${KIND_CONFIG_FILE}
  HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster --retain -v 1 --name ${CLUSTER_NAME} --config=${KIND_CONFIG_FILE}
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
