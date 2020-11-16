#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

CONFIG_DIR=$SCRIPT_DIR/config
TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

VERRAZZANO_NS=verrazzano-system

ENV_NAME=$(get_config_value ".environmentName")
# check environment name length
validate_environment_name $ENV_NAME
if [ $? -ne 0 ]; then
  exit 1
fi

INGRESS_TYPE=$(get_config_value ".ingress.type")
INGRESS_IP=$(get_verrazzano_ingress_ip)
if [ -n "${INGRESS_IP:-}" ]; then
  log "Found ingress address ${INGRESS_IP}"
else
  fail "Failed to find ingress address."
fi

DNS_TYPE=$(get_config_value ".dns.type")
DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})

# Check if the nginx ingress ports are accessible
function check_ingress_ports() {
  exitvalue=0
  if [ ${INGRESS_TYPE} == "LoadBalancer" ] && [ $DNS_TYPE != "external" ]; then
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
        log "Port $PORT is accessible on ingress address $INGRESS_IP.  Note that '404 page not found' is an expected response."
      else
        log "ERROR: Port $PORT is NOT accessible on ingress address $INGRESS_IP!  Check that security lists include an ingress rule for the node port $NODEPORT."
        log "See install README for details(https://github.com/verrazzano/verrazzano/blob/master/install/README.md#1-oke-missing-security-list-ingress-rules)."
        exitvalue=1
      fi
    done
  fi
  return $exitvalue
}

action "Checking ingress ports" check_ingress_ports || fail "ERROR: Failed ingress port check."

set -eu

function create_admission_controller_cert()
{
  echo # for newline before additional output from below commands

  # Prepare verrazzano_admission_controller_ca_config.txt and verrazzano_admission_controller_cert_config.txt
  sed "s/VERRAZZANO_NS/${VERRAZZANO_NS}/g" $CONFIG_DIR/verrazzano_admission_controller_ca_config.txt > $TMP_DIR/verrazzano_admission_controller_ca_config.txt
  sed "s/VERRAZZANO_NS/${VERRAZZANO_NS}/g" $CONFIG_DIR/verrazzano_admission_controller_cert_config.txt > $TMP_DIR/verrazzano_admission_controller_cert_config.txt

  # Create the private key for our custom CA
  if ! openssl genrsa -out $TMP_DIR/ca.key 2048 ; then
    echo "ERROR: Failed to create private key for our CA"
    return 1
  fi

  # Generate a CA cert with the private key
  if ! openssl req -new -x509 -key $TMP_DIR/ca.key -out $TMP_DIR/ca.crt -config $TMP_DIR/verrazzano_admission_controller_ca_config.txt; then
    echo "ERROR: Failed to generate CA cert with private key"
    return 1
  fi

  # Create the private key for our server
  if ! openssl genrsa -out $TMP_DIR/verrazzano-key.pem 2048; then
    echo "ERROR: Failed to create private key for server"
    return 1
  fi

  # Create a CSR from the configuration file and our private key
  if ! openssl req -new -key $TMP_DIR/verrazzano-key.pem -subj "/CN=verrazzano-validation.${VERRAZZANO_NS}.svc" -out $TMP_DIR/verrazzano.csr -config $TMP_DIR/verrazzano_admission_controller_cert_config.txt; then
  echo "ERROR: Failed to create a certificate signing request (CSR) from the configuration file and private key"
    return 1
  fi

  # Create the cert signing the CSR with the CA created before
  if ! openssl x509 -req -in $TMP_DIR/verrazzano.csr -CA $TMP_DIR/ca.crt -CAkey $TMP_DIR/ca.key -CAcreateserial -out $TMP_DIR/verrazzano-crt.pem; then
    echo "ERROR: Failed to create certificate signing request (CSR) with CA"
    return 1
  fi

  kubectl create secret generic verrazzano-validation -n ${VERRAZZANO_NS} \
  --from-file=cert.pem=$TMP_DIR/verrazzano-crt.pem \
  --from-file=key.pem=$TMP_DIR/verrazzano-key.pem \
  --from-file=ca.crt=$TMP_DIR/ca.crt \
  --from-file=ca.key=$TMP_DIR/ca.key

  if [ $? -ne 0 ]; then
      echo "ERROR: Failed to create secret verrazzano-validation"
      return 1
  fi
}

function install_verrazzano()
{
  local RANCHER_HOSTNAME=rancher.${ENV_NAME}.${DNS_SUFFIX}

  local rancher_admin_password=`kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode`

  if [ -z "$rancher_admin_password" ] ; then
    error "ERROR: Failed to retrieve rancher-admin-secret - did you run the scripts to install Istio and system components?"
    return 1
  fi

  # Wait until rancher TLS cert is ready
  log "Waiting for Rancher TLS cert to reach ready state"
  kubectl wait --for=condition=ready cert tls-rancher-ingress -n cattle-system

  # Make sure rancher ingress has an IP
  wait_for_ingress_ip rancher cattle-system || exit 1

  get_rancher_access_token "${RANCHER_HOSTNAME}" "${rancher_admin_password}"
  if [ $? -ne 0 ] ; then
    error "ERROR: Failed to get rancher access token"
    exit 1
  fi
  local token_array=(${RANCHER_ACCESS_TOKEN//:/ })

  EXTRA_V8O_ARGUMENTS=""
  if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
    EXTRA_V8O_ARGUMENTS=" --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
  fi

  log "Installing verrazzano from Helm chart"
  helm \
      upgrade --install verrazzano \
      ${SCRIPT_DIR}/chart \
      --namespace ${VERRAZZANO_NS} \
      --set image.pullPolicy=IfNotPresent \
      --set config.envName=${ENV_NAME} \
      --set config.dnsSuffix=${DNS_SUFFIX} \
      --set config.enableMonitoringStorage=true \
      --set clusterOperator.rancherURL=https://${RANCHER_HOSTNAME} \
      --set clusterOperator.rancherUserName="${token_array[0]}" \
      --set clusterOperator.rancherPassword="${token_array[1]}" \
      --set clusterOperator.rancherHostname=${RANCHER_HOSTNAME} \
      --set verrazzanoAdmissionController.caBundle="$(kubectl -n ${VERRAZZANO_NS} get secret verrazzano-validation -o json | jq -r '.data."ca.crt"' | base64 --decode)" \
      ${EXTRA_V8O_ARGUMENTS} || return $?

  log "Verifying that needed secrets are created"
  retries=0
  until [ "$retries" -ge 60 ]
  do
      kubectl get secret -n ${VERRAZZANO_NS} verrazzano | grep verrazzano && break
      retries=$(($retries+1))
      sleep 5
  done
  if ! kubectl get secret --namespace ${VERRAZZANO_NS} verrazzano ; then
      error "ERROR: failed creating verrazzano secret"
      exit 1
  fi
  log "Verrazzano install completed"
}

# Set environment variable for checking if optional imagePullSecret was provided
REGISTRY_SECRET_EXISTS=$(check_registry_secret_exists)

if ! kubectl get namespace ${VERRAZZANO_NS} ; then
  action "Creating ${VERRAZZANO_NS} namespace" kubectl create namespace ${VERRAZZANO_NS} || exit 1
fi

if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
  if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ${VERRAZZANO_NS} > /dev/null 2>&1 ; then
    action "Copying ${GLOBAL_IMAGE_PULL_SECRET} secret to ${VERRAZZANO_NS} namespace" \
        copy_registry_secret "${VERRAZZANO_NS}"
  fi
fi

action "Creating admission controller cert" create_admission_controller_cert || exit 1
action "Installing Verrazzano system components" install_verrazzano || exit 1
