#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

JENKINS_RUNNER_CONTAINER_LABEL=${JENKINS_RUNNER_CONTAINER_LABEL:-jenkins-runner}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-verrazzano}
KUBECONFIG=${KUBECONFIG:-${HOME}/.kube/config}

set -o pipefail

set -xv

kind_container_name=${KIND_CLUSTER_NAME}-control-plane
kind_container_ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ${kind_container_name})

if [ $? -ne 0 ]; then
    echo "Kind container ${KIND_CONTAINER_NAME} not running"
    exit 1
fi

kind_kubeconfig_cluster_name="kind-${KIND_CLUSTER_NAME}"
kubectl --kubeconfig ${KUBECONFIG} config set-cluster ${kind_kubeconfig_cluster_name} --server "https://${kind_container_ip}:6443"
if [ $? -ne 0 ] ; then
    echo "Unable to set server address for cluster ${kind_kubeconfig_cluster_name} in KUBECONFIG (${KUBECONFIG})"
    exit 1
fi

for jenkins_runner_container in $(docker ps  -q -f "label=${JENKINS_RUNNER_CONTAINER_LABEL}") ; do
    if ! docker inspect ${jenkins_runner_container} | jq -e .[].NetworkSettings.Networks.kind ; then
        docker network connect kind ${jenkins_runner_container}
        if [ $? -ne 0 ] ; then
            echo "Unable to connect container ${jenkins_runner_container} to docker network 'kind'"
            exit 1
        fi
    fi
done


exit 0
