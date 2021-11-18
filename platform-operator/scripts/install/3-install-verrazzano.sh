#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

CONFIG_DIR=$SCRIPT_DIR/config

VERRAZZANO_NS=verrazzano-system
VERRAZZANO_MC=verrazzano-mc
MONITORING_NS=monitoring

ENV_NAME=$(get_config_value ".environmentName")

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
    PORTS=$(kubectl get services -n ingress-nginx ingress-controller-ingress-nginx-controller -o=custom-columns=PORT:.spec.ports[*].name --no-headers)
    IFS=',' read -r -a port_array <<< "$PORTS"

    index=0
    for element in "${port_array[@]}"
    do
      # For each get the port, nodePort and targetPort
      RESP=$(kubectl get services -n ingress-nginx ingress-controller-ingress-nginx-controller -o=custom-columns=PORT:.spec.ports[$index].port,NODEPORT:.spec.ports[$index].nodePort,TARGETPORT:.spec.ports[$index].targetPort --no-headers)
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
        log "For more detail please see https://verrazzano.io/docs/troubleshooting/troubleshooting/."
        exitvalue=1
      fi
    done
  fi
  return $exitvalue
}

action "Checking ingress ports" check_ingress_ports || fail "ERROR: Failed ingress port check."

platform_operator_install_message "Verrazzano"
platform_operator_install_message "Coherence Kubernetes operator"
platform_operator_install_message "WebLogic Kubernetes operator"
platform_operator_install_message "OAM Kubernetes operator"
platform_operator_install_message "Verrazzano Application Kubernetes operator"
