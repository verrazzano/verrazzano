#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

CONFIG_DIR=$SCRIPT_DIR/config
CERTS_OUT=$SCRIPT_DIR/build/admin-control-cert

function set_INGRESS_IP() {
  if [ ${CLUSTER_TYPE} == "OKE" ]; then
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  elif [ ${CLUSTER_TYPE} == "KIND" ]; then
    INGRESS_IP=$(kubectl get node ${KIND_CLUSTER_NAME}-control-plane -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
  fi
}

VERRAZZANO_NS=verrazzano-system
VERRAZZANO_VERSION=v0.0.39
RancherAdminPassword=${RancherAdminPassword:=admin}
set_INGRESS_IP

set -u

function create_admission_controller_cert()
{
  rm -rf $CERTS_OUT
  mkdir -p $CERTS_OUT

  # Prepare verrazzano_admission_controller_ca_config.txt and verrazzano_admission_controller_cert_config.txt
  sed "s/VERRAZZANO_NS/${VERRAZZANO_NS}/g" $CONFIG_DIR/verrazzano_admission_controller_ca_config.txt > $CERTS_OUT/verrazzano_admission_controller_ca_config.txt
  sed "s/VERRAZZANO_NS/${VERRAZZANO_NS}/g" $CONFIG_DIR/verrazzano_admission_controller_cert_config.txt > $CERTS_OUT/verrazzano_admission_controller_cert_config.txt

  # Create the private key for our custom CA
  openssl genrsa -out $CERTS_OUT/ca.key 2048

  # Generate a CA cert with the private key
  openssl req -new -x509 -key $CERTS_OUT/ca.key -out $CERTS_OUT/ca.crt -config $CERTS_OUT/verrazzano_admission_controller_ca_config.txt

  # Create the private key for our server
  openssl genrsa -out $CERTS_OUT/verrazzano-key.pem 2048

  # Create a CSR from the configuration file and our private key
  openssl req -new -key $CERTS_OUT/verrazzano-key.pem -subj "/CN=verrazzano-validation.${VERRAZZANO_NS}.svc" -out $CERTS_OUT/verrazzano.csr -config $CERTS_OUT/verrazzano_admission_controller_cert_config.txt

  # Create the cert signing the CSR with the CA created before
  openssl x509 -req -in $CERTS_OUT/verrazzano.csr -CA $CERTS_OUT/ca.crt -CAkey $CERTS_OUT/ca.key -CAcreateserial -out $CERTS_OUT/verrazzano-crt.pem

  if kubectl get secret verrazzano-validation -n ${VERRAZZANO_NS} ; then
    kubectl delete secret verrazzano-validation -n ${VERRAZZANO_NS}
  fi

  kubectl create secret generic verrazzano-validation -n ${VERRAZZANO_NS} \
  --from-file=cert.pem=$CERTS_OUT/verrazzano-crt.pem \
  --from-file=key.pem=$CERTS_OUT/verrazzano-key.pem \
  --from-file=ca.crt=$CERTS_OUT/ca.crt \
  --from-file=ca.key=$CERTS_OUT/ca.key

  rm -rf $CERTS_OUT
}

function install_verrazzano()
{
  helm uninstall verrazzano --namespace ${VERRAZZANO_NS}
  kubectl delete vmc local
  kubectl delete secret verrazzano-managed-cluster-local
  kubectl -n cattle-system delete secret rancher-admin-secret

  kubectl -n cattle-system create secret generic rancher-admin-secret --from-literal=password="${RancherAdminPassword}"

  export RANCHER_ADMIN_TOKEN=$(curl -k --connect-timeout 30 --retry 10 --retry-delay 30 \
  -d '{"Username":"admin", "Password":"'"${RancherAdminPassword}"'"}' \
  -H "Content-Type: application/json" \
  -X POST https://${RANCHER_HOSTNAME}/v3-public/localProviders/local?action=login | jq -r '.token')

  export RANCHER_ACCESS_TOKEN=$(curl -k --connect-timeout 30 --retry 10 --retry-delay 30 \
  -d '{"type":"token", "description":"automation"}' \
  -H "Content-Type: application/json" -H "Authorization: Bearer ${RANCHER_ADMIN_TOKEN}" \
  -X POST https://${RANCHER_HOSTNAME}/v3/token | jq -r '.token')

  export TOKEN_ARRAY=(${RANCHER_ACCESS_TOKEN//:/ })

  helm \
      upgrade --install verrazzano \
      https://objectstorage.us-phoenix-1.oraclecloud.com/n/stevengreenberginc/b/verrazzano-helm-chart/o/${VERRAZZANO_VERSION}%2Fverrazzano-${VERRAZZANO_VERSION}.tgz \
      --namespace ${VERRAZZANO_NS} \
      --set image.pullPolicy=IfNotPresent \
      --set config.envName=${NAME} \
      --set config.dnsSuffix=${DNS_SUFFIX} \
      --set config.enableMonitoringStorage=true \
      --set verrazzanoOperator.sslVerify=false \
      --set clusterOperator.rancherURL=https://${RANCHER_HOSTNAME} \
      --set clusterOperator.rancherUserName="${TOKEN_ARRAY[0]}" \
      --set clusterOperator.rancherPassword="${TOKEN_ARRAY[1]}" \
      --set clusterOperator.rancherHostname=${RANCHER_HOSTNAME} \
      --set verrazzanoAdmissionController.caBundle="$(kubectl -n ${VERRAZZANO_NS} get secret verrazzano-validation -o json | jq -r '.data."ca.crt"' | base64 --decode)"

  retries=0
  until [ "$retries" -ge 24 ]
  do
      kubectl get secret -n ${VERRAZZANO_NS} verrazzano | grep verrazzano && break
      retries=$(($retries+1))
      sleep 5
  done
  if ! kubectl get secret --namespace ${VERRAZZANO_NS} verrazzano ; then
      consoleerr "ERROR: failed creating verrazzano secret"
      exit 1
  fi
}

function usage {
    consoleerr
    consoleerr "usage: $0 [-n name] [-d dns_type] [-s dns_suffix]"
    consoleerr "  -n name        Environment Name. Optional.  Defaults to default."
    consoleerr "  -d dns_type    DNS type [xip.io|oci]. Optional.  Defaults to xip.io."
    consoleerr "  -s dns_suffix  DNS suffix (e.g v8o.example.com). Not valid for dns_type xip.io. Required for dns-type oci."
    consoleerr "  -h             Help"
    consoleerr
    exit 1
}

NAME="default"
DNS_TYPE="xip.io"
DNS_SUFFIX=""

while getopts n:d:s:h flag
do
    case "${flag}" in
        n) NAME=${OPTARG};;
        d) DNS_TYPE=${OPTARG};;
        s) DNS_SUFFIX=${OPTARG};;
        h) usage;;
    esac
done
# check for valid DNS type
if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "oci" ]; then
  consoleerr
  consoleerr "Unknown DNS type ${DNS_TYPE}"
  usage
fi
# check expected dns suffix for given dns type
if [ -z "$DNS_SUFFIX" ]; then
  if [ $DNS_TYPE = "oci" ]; then
    consoleerr
    consoleerr "-s option is required for ${DNS_TYPE}"
    usage
  else
    DNS_SUFFIX="${INGRESS_IP}".xip.io
  fi
else
  if [ $DNS_TYPE = "xip.io" ]; then
    consoleerr
    consoleerr "A dns_suffix should not be given with dns_type xip.io!"
    usage
  fi
fi

RANCHER_HOSTNAME=rancher.${NAME}.${DNS_SUFFIX}

if ! kubectl get namespace ${VERRAZZANO_NS} ; then
  action "Creating ${VERRAZZANO_NS} namespace" kubectl create namespace ${VERRAZZANO_NS} || exit 1
fi

action "Creating admission controller cert" create_admission_controller_cert || exit 1
action "Installing Verrazzano system components" install_verrazzano || exit 1
