#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
CLUSTER_COUNT=${1:-1}
KUBECONFIG_DIR=$2
INSTALL_CALICO=${3:-true}
REQUIRED_VNC_COUNT=0
REQUIRED_LB_COUNT=0
CLUSTER_NAME_PREFIX="oke"
MODULE_NAME_PREFIX="oke"

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

if [ -z "$TF_VAR_compartment_id" ] ; then
    echo "TF_VAR_compartment_id env var must be set!"
    exit 1
fi

echo "Check OCI CLI is working..."
# If OCI CLI is not configured correctly, the following command will have a non-zero return code
# which will cause the job to fail at this point
oci ce cluster list --compartment-id=${TF_VAR_compartment_id} --region=${TF_VAR_region} > /dev/null

if [ -f /tmp/main.tf ]; then
  rm /tmp/main.tf
fi

# Create main.tf files for each cluster, and evaluate the VNC and LB counts
for i in $(seq 1 $CLUSTER_COUNT)
do
  let REQUIRED_VNC_COUNT=$REQUIRED_VNC_COUNT+1
  let REQUIRED_LB_COUNT=$REQUIRED_LB_COUNT+2
  cp ${SCRIPT_DIR}/terraform/cluster/multi_cluster_main_tf_template /tmp/main.tf.$i
  sed -i "s/MODULE_NAME/$MODULE_NAME_PREFIX-$i/g" /tmp/main.tf.$i
  sed -i "s/CLUSTER_NAME/$CLUSTER_NAME_PREFIX-$i/g" /tmp/main.tf.$i
done

echo "Minimum required VNC count: ${REQUIRED_VNC_COUNT}"
echo "Minimum required Load Balancer count: ${REQUIRED_LB_COUNT}"

rm -rf ${KUBECONFIG_DIR}/*

# check available resources
check_for_resources VCN vcn vcn-count $REQUIRED_VNC_COUNT
check_for_resources LB load-balancer lb-flexible-count $REQUIRED_LB_COUNT

cd ${SCRIPT_DIR}/terraform/cluster

# Set whether Calico is to be installed or not by the OCI OKE TF provider
export TF_VAR_calico_enabled="${INSTALL_CALICO}"
export TF_VAR_calico_version="$(grep 'calico-version=' ${SCRIPT_DIR}/../../../../.third-party-test-versions | sed 's/calico-version=//g')"

for i in $(seq 1 $CLUSTER_COUNT)
do
  echo 'Create OKE cluster...'
  # Copy the temporary file as main.tf
  cat /tmp/main.tf.$i
  cp /tmp/main.tf.$i ${SCRIPT_DIR}/terraform/cluster/main.tf

  ./create-multi-cluster.sh $i $CLUSTER_NAME_PREFIX
  status_code=$?

  if [ ${status_code:-1} -eq 0 ]; then
    echo "Create kube config for cluster ${TF_VAR_label_prefix}-$CLUSTER_NAME_PREFIX-$i ..."
    CLUSTER_OCID=$(oci ce cluster list --compartment-id "${TF_VAR_compartment_id}" --name "${TF_VAR_label_prefix}-${CLUSTER_NAME_PREFIX}-$i" --lifecycle-state "ACTIVE" | jq -r '.data[0]."id"')
    echo "OCID of the cluster ${TF_VAR_label_prefix}-$CLUSTER_NAME_PREFIX-$i : ${CLUSTER_OCID}."
    mkdir -p "${KUBECONFIG_DIR}/$i"
    oci ce cluster create-kubeconfig --cluster-id ${CLUSTER_OCID} --file "${KUBECONFIG_DIR}/$i/kube_config" --region "${TF_VAR_region}" --token-version 2.0.0
    export KUBECONFIG="${KUBECONFIG_DIR}/$i/kube_config"
    # Adding a Service Account Authentication Token to kubeconfig
    # https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengaddingserviceaccttoken.htm
    ${SCRIPT_DIR}/update_oke_kubeconfig.sh
  else
    echo "OKE Cluster creation request failed!"
    exit 1
  fi
done

# Wait for all of the OKE clusters to be ready
for i in $(seq 1 $CLUSTER_COUNT)
do
  export KUBECONFIG="${KUBECONFIG_DIR}/$i/kube_config"
  echo "Waiting for nodes to be added to cluster..."
  timeout 15m bash -c 'until kubectl get nodes | grep NAME; do sleep 10; done'
  echo "Waiting for nodes to transition to 'READY'..."
  kubectl wait --for=condition=ready nodes --timeout=5m --all
done
