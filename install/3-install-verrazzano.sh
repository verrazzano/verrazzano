#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
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
  elif [ ${CLUSTER_TYPE} == "OLCNE" ]; then
    # Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    if [ ${INGRESS_IP} == "null" ]; then
      INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json  | jq -r '.spec.externalIPs[0]')
    fi
  fi
}

# Check if the nginx ingress ports are accessible
function check_ingress_ports() {
  exitvalue=0
  if [ ${CLUSTER_TYPE} == "OKE" ]; then
    # Get the ports from the ingress
    PORTS=$(kubectl get services -n ingress-nginx ingress-controller-nginx-ingress-controller -o=custom-columns=PORT:.spec.ports[*].name --no-headers)
    IFS=',' read -r -a port_array <<< "$PORTS"

    index=0
    for element in "${port_array[@]}"
    do
      # For each get the port, nodePort and targetPort
      RESP=$(kubectl get services -n ingress-nginx ingress-controller-nginx-ingress-controller -o=custom-columns=PORT:.spec.ports[$index].port,NODEPORT:.spec.ports[$index].nodePort,TARGETPORT:.spec.ports[$index].targetPort --no-headers)
      ((index++))

      IFS=' ' read -r -a vals <<< "$RESP"
      PORT="${vals[0]}"
      NODEPORT="${vals[1]}"
      TARGETPORT="${vals[2]}"

      # Attempt to access the port on the $INGRESS_IP
      if [ $TARGETPORT == "https" ]; then
        ARGS=(-k https://$INGRESS_IP:$PORT)
        call_curl 0 response http_code ARGS
      else
        ARGS=(http://$INGRESS_IP:$PORT)
        call_curl 0 response http_code ARGS
      fi

      # Check the result of the curl call
      if [ $? -eq 0 ]; then
        echo
        echo "Port $PORT is accessible on ingress($INGRESS_IP).  Note that '404 page not found' is an expected response."
      else
        echo
        echo "ERROR: Port $PORT is NOT accessible on ingress($INGRESS_IP)!  Check that security lists include an ingress rule for the node port $NODEPORT."
        exitvalue=1
      fi
    done
  fi
  return $exitvalue
}

VERRAZZANO_NS=verrazzano-system
VERRAZZANO_VERSION=v0.0.0-0b1ef5aef39f954cb13c13ed0757759db1ec691d
set_INGRESS_IP
check_ingress_ports
if [ $? -ne 0 ]; then
  echo "ERROR: Failed ingress port check."
  exit 1
fi

set -eu

function create_admission_controller_cert()
{
  echo # for newline before additional output from below commands
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

function dump_rancher_ingress {
  echo
  echo "########  rancher ingress details ##########"
  kubectl get ingress rancher -n cattle-system -o yaml
  echo "########  end rancher ingress details ##########"
}

function install_verrazzano()
{
  logDt "Uninstalling any existing verrazzano components"
  helm uninstall verrazzano --namespace ${VERRAZZANO_NS} /dev/null 2>&1 || true
  kubectl delete vmc local /dev/null 2>&1 || true
  kubectl delete secret verrazzano-managed-cluster-local /dev/null 2>&1 || true
  logDt "Completed uninstall"

  RancherAdminPassword=`kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode`

  if [ -z "$RancherAdminPassword" ] ; then
    consoleerr "ERROR: Failed to retrieve rancher-admin-secret - did you run the scripts to install Istio and system components?"
    return 1
  fi

  # Wait until rancher TLS cert is ready
  logDt "Waiting for Rancher TLS cert to reach ready state"
  kubectl wait --for=condition=ready cert tls-rancher-ingress -n cattle-system

  # Make sure rancher ingress has an IP
  wait_for_ingress_ip rancher cattle-system

  get_rancher_access_token "${RANCHER_HOSTNAME}" "${RancherAdminPassword}" || exit 1

  export TOKEN_ARRAY=(${RANCHER_ACCESS_TOKEN//:/ })

  logDt "Installing verrazzano from Helm chart"
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

  logDt "Verifying that needed secrets are created"
  retries=0
  until [ "$retries" -ge 60 ]
  do
      kubectl get secret -n ${VERRAZZANO_NS} verrazzano | grep verrazzano && break
      retries=$(($retries+1))
      sleep 5
  done
  if ! kubectl get secret --namespace ${VERRAZZANO_NS} verrazzano ; then
      consoleerr "ERROR: failed creating verrazzano secret"
      exit 1
  fi
  logDt "Verrazzano install completed"
}

function usage {
    consoleerr
    consoleerr "usage: $0 [-n name] [-d dns_type] [-s dns_suffix]"
    consoleerr "  -n name        Environment Name. Optional.  Defaults to default."
    consoleerr "  -d dns_type    DNS type [xip.io|manual|oci]. Optional.  Defaults to xip.io."
    consoleerr "  -s dns_suffix  DNS suffix (e.g v8o.example.com). Not valid for dns_type xip.io. Required for dns-type oci or manual"
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
        *) usage;;
    esac
done
# check for valid DNS type
if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "oci" ] && [ $DNS_TYPE != "manual" ]; then
  consoleerr
  consoleerr "Unknown DNS type ${DNS_TYPE}"
  usage
fi

set_INGRESS_IP

# check expected dns suffix for given dns type
if [ -z "$DNS_SUFFIX" ]; then
  if [ $DNS_TYPE == "oci" ] || [ $DNS_TYPE == "manual" ]; then
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
