#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

PRIVATE_CLUSTER=${1:-false}
INSTALL_CALICO=${2:-true}

set_private_access() {
  echo "Cluster access set to private."
  export TF_VAR_cluster_access=private
  export TF_VAR_bastion_enabled=true
}

check_for_resources() {
  local resource_type=$1
  local service_name=$2
  local limit_name=$3
  local min_required=$4

  local count=$(${SCRIPT_DIR}/get_resource_availability.sh $service_name $limit_name)

  local status_code=$?
  if [ ${status_code:-1} -eq 0 ]; then
    # OCI query succeeded, proceed with value evaluation
    if [ $count -lt $min_required ]; then
      echo "ERROR: Not enough ${resource_type}s available to create the OKE cluster : ${count}"
      exit 1
    elif [ $count -lt 5 ]; then
      echo "WARNING: Critically low number of ${resource_type}s available in tenancy: ${count}. Proceeding with creating the OKE cluster ..."
    else
      echo "Sufficient number of ${resource_type}s available for creating the OKE cluster: ${count}"
    fi
  else
    echo "ERROR: Query for available number of ${resource_type}s in tenancy failed."
    exit 1
  fi
}

if [ $PRIVATE_CLUSTER == true ] ; then
    set_private_access
fi

if [ -z "$TF_VAR_compartment_id" ] ; then
    echo "TF_VAR_compartment_id env var must be set!"
    exit 1
fi

if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

echo "Check OCI CLI is working..."
# If OCI CLI is not configured correctly, the following command will have a non-zero return code
# which will cause the job to fail at this point
oci ce cluster list --compartment-id=${TF_VAR_compartment_id} --region=${TF_VAR_region} > /dev/null

# check available resources
check_for_resources VCN vcn vcn-count 1
check_for_resources LB load-balancer lb-flexible-count 2

echo 'Install OKE...'
echo 'Create cluster...'
cd ${SCRIPT_DIR}/terraform/cluster

# Set whether Calico is to be installed or not by the OCI OKE TF provider
export TF_VAR_calico_enabled="${INSTALL_CALICO}"
export TF_VAR_calico_version="$(grep 'calico-version=' ${SCRIPT_DIR}/../../../../.third-party-test-versions | sed 's/calico-version=//g')"

echo "Create cluster started at $(date)"
./create-cluster.sh
status_code=$?
echo "Create cluster completed at $(date)"
if [ ${status_code:-1} -eq 0 ]; then

    # if the cluster has been created with private endpoints then setup the ssh tunnel through the bastion host
    if [ "$TF_VAR_bastion_enabled" = true ] ; then
      echo "Setting up ssh tunnel through bastion host."
      ../../setup_ssh_tunnel.sh
      if [ $? -ne 0 ]; then
          echo "Can't setup ssh tunnel through bastion host!"
          exit 1
      fi
    fi


    echo "Updating generated KUBECONFIG $(date)"
    cat generated/kubeconfig > ${KUBECONFIG}
    echo "Generated KUBECONFIG contents:"
    cat ${KUBECONFIG}
    # Adding a Service Account Authentication Token to kubeconfig
    # https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengaddingserviceaccttoken.htm
    ${SCRIPT_DIR}/update_oke_kubeconfig.sh
    echo "KUBECONFIG contents after update:"
    cat ${KUBECONFIG}

    # Right after oke cluster is provisioned, it takes a while before any node is added to the cluster
    # The next command will wait for node to come up before continue
    echo "Waiting for nodes to be added to cluster at $(date)..."
    timeout 15m bash -c 'until kubectl get nodes | grep NAME; do sleep 10; done'
    echo "Waiting for nodes to transition to 'READY' at $(date)..."
    kubectl wait --for=condition=ready nodes --timeout=5m --all
else
    echo "OKE Cluster creation request failed!"
    exit 1
fi
