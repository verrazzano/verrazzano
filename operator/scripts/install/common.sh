#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# The directory that contains this script.
SOURCE_DIR=$(cd $(dirname $BASH_SOURCE); pwd -P)
# The directory that contains the calling script.
SCRIPT_DIR=${SCRIPT_DIR:-$(cd $(dirname ${BASH_SOURCE[${#BASH_SOURCE[@]} - 1]}); pwd -P)}
# The directory where any generated artifacts should be stored.
BUILD_DIR="${SCRIPT_DIR}/build"
CHARTS_DIR=$(cd $SOURCE_DIR/../../../thirdparty/charts; pwd -P)
HELM_CONFIG_DIR=$(cd $SOURCE_DIR/../../../helm_config; pwd -P)

MANIFESTS_DIR=$(cd $SOURCE_DIR/../../../thirdparty/manifests; pwd -P)

. ${SOURCE_DIR}/logging.sh

# DEPRECATED: This function is deprecated and is replaced by the status function in logging.sh
function consoleout() {
  status "$@"
}

# DEPRECATED: This function is deprecated and is replaced by the error function in logging.sh
function consoleerr() {
  error "$@"
}

function wait_for_ingress_ip() {
  local retries=0
  local ingress_name=$1
  local namespace=$2
  local ingress_ip

  log "Waiting for ingress $ingress_name in namespace $namespace to have an IP"
  until [ "$retries" -ge 10 ]
  do
      ingress_ip=$(kubectl get ingress $ingress_name -n $namespace -o json | jq -r '.status.loadBalancer.ingress[].ip')
      if [ -n "$ingress_ip" ] ; then
          break;
      fi
      retries=$(($retries+1))
      sleep 5
  done
  if [ "$retries" -ge 10 ] ; then
    log "An error occurred - ingress $ingress_name in namespace $namespace did not have an IP address"
    return 1
  fi
}

function get_rancher_access_token {
  local rancher_hostname=$1
  local rancher_password=$2
  local rancher_admin_token=""
  local retries=0
  log "Retrieving the rancher admin token from Rancher at $rancher_hostname"

  # Use external retries instead of curl retries, since curl does not retry for all
  # the scenarios we want (e.g. connection errors)
  until [ $retries -ge 10 ]
  do
    ARGS=(-k --connect-timeout 30 $(get_rancher_resolve ${rancher_hostname}) \
    -d '{"Username":"admin", "Password":"'$rancher_password'"}' \
    -H "Content-Type: application/json" \
    -X POST https://$rancher_hostname/v3-public/localProviders/local?action=login)
    call_curl 201 response http_code ARGS
    if [ $? -eq 0 ]; then
      rancher_admin_token=$(echo $response | jq -r '.token')

      if [ ! -z "$rancher_admin_token" ] ; then
        break
      fi
    fi
    log "Retrying get rancher_admin_token"
    retries=$(($retries+1))
    sleep 30
  done

  if [ -z "$rancher_admin_token" ] ; then
      echo "rancher_admin_token is empty! Did you run the scripts to install Istio and system components?"
      return 1
  fi

  log "Retrieving the access token from Rancher at $rancher_hostname"

  local rancher_access_token=""
  # Use external retries instead of curl retries, since curl does not retry for all
  # the scenarios we want (e.g. connection errors)
  local retries=0
  until [ "$retries" -ge 10 ]
  do
    ARGS=(-k --connect-timeout 30 $(get_rancher_resolve ${rancher_hostname}) \
    -d '{"type":"token", "description":"automation"}' \
    -H "Content-Type: application/json"
    -H "Authorization: Bearer ${rancher_admin_token}" \
    -X POST https://$rancher_hostname/v3/token )
    call_curl 201 response http_code ARGS
    if [ $? -eq 0 ]; then
      rancher_access_token=$(echo $response | jq -r '.token')

      if [ ! -z "$rancher_access_token" ] ; then
        break
      fi
    fi
    log "Retrying get rancher_access_token"
    retries=$(($retries+1))
    sleep 30
  done

  if [ -z "$rancher_access_token" ] ; then
      log "rancher_access_token is empty!\n"
      echo
      echo "Dumping additional detail below"
      dump_rancher_ingress
      return 1
  fi

  RANCHER_ACCESS_TOKEN=$rancher_access_token
}

function dump_rancher_ingress {
  echo
  echo "########  rancher ingress details ##########"
  kubectl get ingress rancher -n cattle-system -o yaml
  echo "########  end rancher ingress details ##########"
}

# Check if the optional global registry secret exists
function check_registry_secret_exists() {
  local result
  kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n default > /dev/null 2>&1
  if [ $? -eq 0 ]; then
    result="TRUE"
  else
    result="FALSE"
  fi
  echo ${result}
}

# Copy global registry secret to the namespace passed in the first argument
function copy_registry_secret()
{
  DEST_NS=$1
  kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n default -o yaml \
      | sed "s|namespace: default|namespace: ${DEST_NS}|" \
      | kubectl apply -n ${DEST_NS} -f -
}

# Call curl with the given arguments and set the given variables for response body and http code.
# $1 the expected http response code; pass 0 to indicate that the http code shouldn't be checked
# $2 the variable to set with the response body
# $3 the variable to set with the http response code
# $4 array of arguments to pass to the curl call
# Exit code: success (0); error code (1) if the curl call fails or if the expected http code is not returned
function call_curl {
  local resp
  local exitcode
  local expected_code=$1
  local resp_body=$2
  local http_code=$3
  local arg_name=$4[@]
  local curl_args=("${!arg_name}")
  local regex="(.*)-- http_code:(.*)"

  # make the curl call
  resp=$(curl -s -w '-- http_code:%{http_code}\n' "${curl_args[@]}"); exitcode=$?

  # if the curl command succeeded
  if [ $exitcode -eq 0 ]; then
    # use regex to capture the response body and http code
    if [[ $resp  =~ $regex ]];  then
      local body='"${BASH_REMATCH[1]}"'
      eval $resp_body=$body
      local code=${BASH_REMATCH[2]}
      eval $http_code=$code
      # check for the expected http code response
      if [ $expected_code -gt 0 ] && [ $code -ne $expected_code ]; then
        echo "ERROR: Expected http response code" $expected_code "but got" $code"!  Response: " $resp
        return 1
      fi
      return 0
    fi
    echo "ERROR: Can't parse curl response: " $resp
  else
    echo "ERROR: curl call failed with exit code: " $exitcode
  fi
  return 1
}


VERRAZZANO_DIR=${SCRIPT_DIR}/.verrazzano

VERRAZZANO_KUBECONFIG="${VERRAZZANO_KUBECONFIG:-}"
if [ -z "${VERRAZZANO_KUBECONFIG}" ] ; then
  fail "Environment variable VERRAZZANO_KUBECONFIG must be set and point to a valid kubernetes configuration file"
fi
if [ ! -f "${VERRAZZANO_KUBECONFIG}" ] ; then
  fail "Environment variable VERRAZZANO_KUBECONFIG points to file ${VERRAZZANO_KUBECONFIG} which does not exist"
fi
export KUBECONFIG="${VERRAZZANO_KUBECONFIG}"


command -v helm >/dev/null 2>&1 || {
  fail "helm is required but cannot be found on the path. Aborting."
}
command -v kubectl >/dev/null 2>&1 || {
  fail "kubectl is required but cannot be found on the path. Aborting."
}
command -v openssl >/dev/null 2>&1 || {
  fail "openssl is required but cannot be found on the path. Aborting."
}
command -v jq >/dev/null 2>&1 || {
  fail "jq is required but cannot be found on the path. Aborting."
}
command -v curl >/dev/null 2>&1 || {
  fail "curl is required but cannot be found on the path. Aborting."
}

##################################################
####Constants for Docker images, versions, tags
##################################################
GLOBAL_HUB_REPO=ghcr.io/verrazzano
GLOBAL_IMAGE_PULL_SECRET=verrazzano-container-registry

GRAFANA_REPO=ghcr.io/verrazzano/grafana
GRAFANA_TAG=v6.4.4

PROMETHEUS_TAG=v2.13.1

MYSQL_IMAGE=ghcr.io/verrazzano/mysql
MYSQL_IMAGE_TAG=8.0.20

OAM_OPERATOR_IMAGE_REPO=ghcr.io/verrazzano/oam-kubernetes-runtime
OAM_OPERATOR_IMAGE_TAG=v0.3.0

WEBLOGIC_OPERATOR_IMAGE_REPO=oracle/weblogic-kubernetes-operator
WEBLOGIC_OPERATOR_IMAGE_TAG=3.1.0
