#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
CLUSTER_NAME=$1
PLATFORM_OPERATOR_DIR=$2
KUBECONFIG=$3
K8S_VERSION=$4
CLEANUP_KIND_CONTAINERS=${5:-true}
CONNECT_JENKINS_RUNNER_TO_NETWORK=${6:-false}
KIND_AT_CACHE=${7:-false}
SETUP_CALICO=${8:-false}
KIND_AT_CACHE_NAME=${9:-"NONE"}
NODE_COUNT=${10:-1}
CALICO_SUFFIX=""
K8S_VERSION=${K8S_VERSION:-1.21}

if [ -z "$TEST_SCRIPTS_DIR" ]; then
  echo "TEST_SCRIPTS_DIR must be set to the E2E test script directory location"
  exit 1
fi
if [ -z "${KUBECONFIG}" ]; then
  echo "KUBECONFIG must be set"
  exit 1
fi
if [ -z "$WORKSPACE" ]; then
  echo "WORKSPACE must be set"
  exit 1
fi

create_kind_cluster() {

  clusterNames=$(kind get clusters)
  if [[ $clusterNames == *"${CLUSTER_NAME}"* ]]; then
    echo "${CLUSTER_NAME} already exists"
    return 0
  fi

  if [ ${K8S_VERSION} == 1.20 ]; then
    KIND_IMAGE="v1.20.15@sha256:6f2d011dffe182bad80b85f6c00e8ca9d86b5b8922cdf433d53575c4c5212248"
  elif [ ${K8S_VERSION} == 1.21 ]; then
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

  export KUBECONFIG=$KUBECONFIG
  echo "Kubeconfig ${KUBECONFIG}"
  touch -f $KUBECONFIG

  echo "KIND Image : ${KIND_IMAGE}"
  KIND_CONFIG_FILE_NAME=kind-config${CALICO_SUFFIX}.yaml
  SOURCE_KIND_CONFIG_FILE=${TEST_SCRIPTS_DIR}/${KIND_CONFIG_FILE_NAME}
  KIND_CONFIG_FILE=${WORKSPACE}/${KIND_CONFIG_FILE_NAME}
  if [ ${KIND_AT_CACHE} == true ]; then
    if [ ${KIND_AT_CACHE_NAME} != "NONE" ]; then
      # If a cache name was specified, replace the at_test cache name with the one specified (this is used only
      # for multi-cluster tests at the moment)
      KIND_CONFIG_FILE_NAME=kind-config-ci${CALICO_SUFFIX}_${KIND_AT_CACHE_NAME}.yaml
      KIND_CONFIG_FILE=${WORKSPACE}/${KIND_CONFIG_FILE_NAME}
      SOURCE_KIND_CONFIG_FILE=${TEST_SCRIPTS_DIR}/${KIND_CONFIG_FILE_NAME}
      sed "s;v8o_cache/at_tests;v8o_cache/${KIND_AT_CACHE_NAME};g" ${SOURCE_KIND_CONFIG_FILE} > ${KIND_CONFIG_FILE}
    else
      # If no cache name specified use at_tests cache
      SOURCE_KIND_CONFIG_FILE=${TEST_SCRIPTS_DIR}/kind-config-ci${CALICO_SUFFIX}.yaml
    fi
  fi
  cp -v ${SOURCE_KIND_CONFIG_FILE} ${KIND_CONFIG_FILE}

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
  sed -i -e "s/KIND_IMAGE/${KIND_IMAGE}/g" ${KIND_CONFIG_FILE}
  cat ${KIND_CONFIG_FILE}
  HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster --retain -v 1 --name ${CLUSTER_NAME} \
    --config=${KIND_CONFIG_FILE}
}

set -e
if [ -n "${VZ_TEST_DEBUG}" ]; then
  set -xv
fi

create_kind_cluster

kubectl config set-context kind-${CLUSTER_NAME}

if [ "${CONNECT_JENKINS_RUNNER_TO_NETWORK}" == "true" ]; then
  dockerIP=$(docker inspect ${CLUSTER_NAME}-control-plane | jq -r '.[].NetworkSettings.Networks[].IPAddress')
  sed -i -e "s|127.0.0.1.*|${dockerIP}:6443|g" ${KUBECONFIG}
  cat ${KUBECONFIG} | grep server

  jenkinsRunnerNetwork=$(docker ps | grep "jenkins-runner" | awk '{ print $1 }')
  if [ -n "${jenkinsRunnerNetwork}" ]; then
    echo "Jenkins runner network detected, connecting to ${jenkinsRunnerNetwork}"
    docker network connect kind ${jenkinsRunnerNetwork}
  else
    echo "Ignore connecting jenkins-runner to a network."
  fi
fi

echo "KIND cluster ${CLUSTER_NAME} setup complete"
