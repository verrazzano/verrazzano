#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z "${KUBECONFIG:-}" ] ; then
  echo "Environment variable KUBECONFIG must be set an point to a valid kube config file"
  exit 1
fi

function usage {
    echo
    echo "usage: $0 -n cluster_name -o output_directory"
    echo "  -n cluster_name            The name of the managed cluster"
    echo "  -o output_directory        The full path to the directory in which the yaml will be generated"
    echo "  -h                         Help"
    echo
    exit 1
}

CLUSTER_NAME=default_managed_cluster
OUTPUT_DIR="./"

while getopts n:o:h flag
do
    case "${flag}" in
        n) CLUSTER_NAME=${OPTARG};;
        o) OUTPUT_DIR=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

OUTPUT_FILE=$OUTPUT_DIR/managed-cluster.yaml
CA_CERT=$(kubectl -n verrazzano-system get secret system-tls -o json | jq -r '.data."ca.crt"' | base64 -d)
AUTH_PASSWORD=$(kubectl get secret verrazzano -n verrazzano-system -o jsonpath='{.data.password}' | base64 -d)
HOST=$(kubectl get ing vmi-system-prometheus -n verrazzano-system -o jsonpath='{.spec.tls[0].hosts[0]}')

#create the yaml file
echo "prometheus:" > $OUTPUT_FILE
echo "  auth-password: $AUTH_PASSWORD" >> $OUTPUT_FILE
echo "  host: $HOST" >> $OUTPUT_FILE
echo "  ca-crt: |" >> $OUTPUT_FILE
echo -e "$CA_CERT" | sed 's/^/    /' >> $OUTPUT_FILE

exit 0

