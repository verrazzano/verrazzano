#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# $1 kind cluster name (OPTIONAL)
#      If specified this is used to determine whether it is a well-known cluster scenario
#      such as the pre-created Kind clusters which exist for Jenkins jobs (apo-integ, vpo-integ, at-tests).
#
#      The pre-created clusters are handled specially as we need to use the pre-baked host name rather
#      than an IP address in the kubeconfig. The pre-baked host name is included in the certSANs, the IP
#      address can change from when the Kind cluster was originally baked in (which would cause cert validation
#      to fail, so we don't rely on IP address for pre-created clusters)
#
#      If not specified it will default to verrazzano, and if specified but not a pre-created cluster it will
#      rely on the IP address
#
# $2 kubeconfig location (REQUIRED for pre-created clusters, not used for others)

KIND_CLUSTER_NAME=$1

JENKINS_RUNNER_CONTAINER_LABEL=${JENKINS_RUNNER_CONTAINER_LABEL:-jenkins-runner}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-verrazzano}
KUBECONFIG=${KUBECONFIG:-${HOME}/.kube/config}

set -o pipefail

set -xv

kind_container_name=${KIND_CLUSTER_NAME}-control-plane
if [ $? -ne 0 ]; then
    echo "Kind container ${KIND_CONTAINER_NAME} not running"
    exit 1
fi

# Pre-created clusters use a pre-baked host name
case $KIND_CLUSTER_NAME in
    "apo-integ")
        BAKED_HOST_NAME="APOINTEGHOST"
        ;;
    "vpo-integ")
        BAKED_HOST_NAME="VPOINTEGHOST"
        ;;
    "at-tests")
        BAKED_HOST_NAME="ATTESTSHOST"
        ;;
    *)
        BAKED_HOST_NAME=""
        ;;
esac

kind_container_ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ${kind_container_name})

if [ -z $BAKED_HOST_NAME ]; then
    echo "Using ${KIND_CLUSTER_NAME} cluster"

    # If it is not a pre-created cluster, the caller already created the cluster and has the full kubeconfig for it, and using
    # the IP for the server address since it was created in place (ie: certSANs for the IP will match)
    kind_kubeconfig_cluster_name="kind-${KIND_CLUSTER_NAME}"
    kubectl --kubeconfig ${KUBECONFIG} config set-cluster ${kind_kubeconfig_cluster_name} --server "https://${kind_container_ip}:6443"
    if [ $? -ne 0 ] ; then
        echo "Unable to set server address for cluster ${kind_kubeconfig_cluster_name} in KUBECONFIG (${KUBECONFIG})"
        exit 1
    fi
else
    echo "Using ${KIND_CLUSTER_NAME} cluster, pre-created on the VM with hostname: ${BAKED_HOST_NAME}"

    # For pre-created clusters there we need to get the kubeconfig from kind for the cluster
    if [ -z $2 ]; then
        echo "Pre-created cluster requires the target kubeconfig location be specified"
    fi
    kind get kubeconfig --name=${KIND_CLUSTER_NAME} > $2

    # Add an entry in /etc/hosts for the IP to pre-baked hostname, and update the server in the kubeconfig
    # to use the pre-baked hostname
    cp /etc/hosts temp_hosts
    echo $kind_container_ip | sed "s/$/  ${BAKED_HOST_NAME}/g" | cat $1 >> temp_hosts
    sudo cp temp_hosts /etc/hosts
    sed -i -e "s|127.0.0.1.*|${BAKED_HOST_NAME}:6443|g" $2
esac

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
