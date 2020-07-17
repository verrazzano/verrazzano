#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "${SCRIPT_DIR}" ] ; then
    echo "SCRIPT_DIR not set"
    exit 1
fi

set -u

BUILD_DIR="${SCRIPT_DIR}/build"
LOGDIR="${BUILD_DIR}/logs"
export LOGFILE="${LOGFILE:-${LOGDIR}/$(basename $0).log}"

if [ ! -d "${LOGDIR}" ] ; then
    mkdir -p "${LOGDIR}"
fi

echo "Output redirected to $LOGFILE"

CONSOLE_STDOUT=5
CONSOLE_STDERR=6

# Reset standard out and standard error streams
exec 5<&1
exec 6<&2

mkdir -p "$LOGDIR"
exec 1>> "$LOGFILE" 2>&1

function consoleout()
{
    echo "$@"
    echo "$@" >&${CONSOLE_STDOUT}
}

function consoleerr()
{
    echo "$@"
    echo "$@" >&${CONSOLE_STDERR}
}

function fail()
{
    consoleerr ""
    consoleerr "$@"
    exit 1;
}

RES_COL=60
MOVE_TO_COL="echo -en \\033[${RES_COL}G"
SETCOLOR_SUCCESS="echo -en \\033[1;32m"
SETCOLOR_FAILURE="echo -en \\033[1;31m"
SETCOLOR_NORMAL="echo -en \\033[0;39m"

function echo_success()
{
  $MOVE_TO_COL
  echo -n "["
  $SETCOLOR_SUCCESS
  echo -n $"  OK  "
  $SETCOLOR_NORMAL
  echo -n "]"
  echo -ne "\r"
  return 0
}

function echo_failure()
{
  $MOVE_TO_COL
  echo -n "["
  $SETCOLOR_FAILURE
  echo -n $"FAILED"
  $SETCOLOR_NORMAL
  echo -n "]"
  echo -ne "\r"
  return 1
}

function echo_progress()
{
  local _progress=$1
  $MOVE_TO_COL
  echo -n "["
  $SETCOLOR_NORMAL
  echo -n $" $_progress "
  $SETCOLOR_NORMAL
  echo -n "]"
  echo -ne "\r"
  return 0
}

function spin()
{
  local spinner='\|/-'
  while :
  do
    for i in `seq 0 3`
    do
      echo_progress "${spinner:$i:1}"
      sleep .1
    done
  done
}

function action() {
  local STRING rc spin_pid
  local DISABLE_SPINNER=${DISABLE_SPINNER:-}

  STRING=$1
  consoleout -n "$STRING "


  if [ -z "${DISABLE_SPINNER}" ] ; then
    spin >&$CONSOLE_STDOUT &
    spin_pid=$!
    trap "kill -0 $spin_pid 2>/dev/null && kill $spin_pid 2>/dev/null " INT ERR EXIT
  fi

  shift
  "$@"
  rc=$?

  if [ -z "${DISABLE_SPINNER}" ] ; then
    kill $spin_pid 2>/dev/null
    wait $spin_pid 2>/dev/null
  fi

  if [ $rc -eq 0 ] ; then
      echo_success >&$CONSOLE_STDOUT
  else
      echo_failure >&$CONSOLE_STDOUT
  fi
  consoleout
  return $rc
}

function onerror()
{
    rv=$?
    if [ $rv -ne 0 ] ; then
        consoleerr ""
        consoleerr "An error occurred, please see ${LOGFILE} for details"
        echo ""
        echo "An error occurred, please see ${LOGFILE} for details"
    fi
    exit $rv
}

function wait_for_ingress_ip() {
  retries=0
  #args $1 = ingress name, $2 = namespace

  logDt "Waiting for ingress $1 in namespace $2 to have an IP"
  until [ "$retries" -ge 10 ]
  do
      ingress_ip=$(kubectl get ingress $1 -n $2 -o json | jq -r '.status.loadBalancer.ingress[].ip')
      if [ ! -z "$ingress_ip" ] ; then
          break;
      fi
      retries=$(($retries+1))
      sleep 5
  done
  if [ "$retries" -ge 10 ] ; then
    logDt "An error occurred - ingress $1 in namespace $2 did not have an IP address"
    exit 1
  fi
}

function logDt() {
  echo -e $(date -u "+%Y-%m-%d %H:%M%:%S %Z") $@
}

function get_rancher_access_token {
    # args  $1 = rancher hostname, $2 = rancher password
  logDt "Retrieving the rancher admin token from Rancher at $1"

  # Use external retries instead of curl retries, since curl does not retry for all
  # the scenarios we want (e.g. connection errors)
  retries=0
  until [ $retries -ge 10 ]
  do
    ARGS=(-k --connect-timeout 30 \
    -d '{"Username":"admin", "Password":"'$2'"}' \
    -H "Content-Type: application/json" \
    -X POST https://$1/v3-public/localProviders/local?action=login)
    call_curl 201 response http_code ARGS
    if [ $? -eq 0 ]; then
      RANCHER_ADMIN_TOKEN=$(echo $response | jq -r '.token')

      if [ ! -z "$RANCHER_ADMIN_TOKEN" ] ; then
        break
      fi
    fi
    logDt "Retrying get RANCHER_ADMIN_TOKEN"
    retries=$(($retries+1))
    sleep 30
  done

  if [ -z "$RANCHER_ADMIN_TOKEN" ] ; then
      echo "RANCHER_ADMIN_TOKEN is empty! Did you run the scripts to install Istio and system components?"
      return 1
  fi

  logDt "Retrieving the access token from Rancher at $1"

  # Use external retries instead of curl retries, since curl does not retry for all
  # the scenarios we want (e.g. connection errors)
  retries=0
  until [ "$retries" -ge 10 ]
  do
    ARGS=(-k --connect-timeout 30 \
    -d '{"type":"token", "description":"automation"}' \
    -H "Content-Type: application/json"
    -H "Authorization: Bearer ${RANCHER_ADMIN_TOKEN}" \
    -X POST https://$1/v3/token )
    call_curl 201 response http_code ARGS
    if [ $? -eq 0 ]; then
      RANCHER_ACCESS_TOKEN=$(echo $response | jq -r '.token')

      if [ ! -z "$RANCHER_ACCESS_TOKEN" ] ; then
        break
      fi
    fi
    logDt "Retrying get RANCHER_ACCESS_TOKEN"
    retries=$(($retries+1))
    sleep 30
  done

  if [ -z "$RANCHER_ACCESS_TOKEN" ] ; then
      logDt "RANCHER_ACCESS_TOKEN is empty!\n"
      echo
      echo "Dumping additional detail below"
      dump_rancher_ingress
      return 1
  fi
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

trap onerror ERR EXIT

KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:=verrazzano}
VERRAZZANO_DIR=${SCRIPT_DIR}/.verrazzano
KIND_KUBE_CONTEXT="kind-${KIND_CLUSTER_NAME}"
KIND_KUBECONFIG="${BUILD_DIR}/kind-kubeconfig"


CLUSTER_TYPE="${CLUSTER_TYPE:-}"
if [ "${CLUSTER_TYPE}" != "KIND" ] && [ "${CLUSTER_TYPE}" != "OKE" ] && [ "${CLUSTER_TYPE}" != "OLCNE" ]; then
    fail "CLUSTER_TYPE environment variable must be set to KIND, OKE or OLCNE"
fi

VERRAZZANO_KUBECONFIG="${VERRAZZANO_KUBECONFIG:-}"
if [ "${CLUSTER_TYPE}" == "KIND" ] && [ -z "${VERRAZZANO_KUBECONFIG}" ] ; then
    VERRAZZANO_KUBECONFIG="${KIND_KUBECONFIG}"
    mkdir -p $(dirname $VERRAZZANO_KUBECONFIG)
else
    if [ -z "${VERRAZZANO_KUBECONFIG}" ] ; then
        fail "Environment variable VERRAZZANO_KUBECONFIG must be set and point to a valid kubernetes configuration file"
    fi
    if [ ! -f "${VERRAZZANO_KUBECONFIG}" ] ; then
        fail "Environment variable VERRAZZANO_KUBECONFIG points to file ${VERRAZZANO_KUBECONFIG} which does not exist"
    fi
fi
export KUBECONFIG="${VERRAZZANO_KUBECONFIG}"


command -v helm >/dev/null 2>&1 || {
    fail "helm is required but cannot be found on the path. Aborting.";
}
command -v kubectl >/dev/null 2>&1 || {
    fail "kubectl is required but cannot be found on the path. Aborting.";
}
command -v openssl >/dev/null 2>&1 || {
    fail "openssl is required but cannot be found on the path. Aborting.";
}
command -v jq >/dev/null 2>&1 || {
    fail "jq is required but cannot be found on the path. Aborting.";
}

##################################################
####Constants for Docker images, versions, tags
##################################################
GLOBAL_HUB_REPO=container-registry.oracle.com/olcne

CERT_MANAGER_IMAGE=phx.ocir.io/stevengreenberginc/bfs/cert-manager-controller
CERT_MANAGER_TAG=0.13.1-049f555-17
CERT_MANAGER_VERSION=0.13.1
CERT_MANAGER_SOLVER_IMAGE=phx.ocir.io/stevengreenberginc/bfs/cert-manager-acmesolver
CERT_MANAGER_SOLVER_TAG=0.13.1-049f555-17

EXTERNAL_DNS_REPO=stevengreenberginc/bfs/external-dns
EXTERNAL_DNS_VERSION=2.20.0
EXTERNAL_DNS_TAG=v0.7.1-08253d3-9
EXTERNAL_DNS_REGISTRY=phx.ocir.io

GRAFANA_REPO=container-registry.oracle.com/olcne/grafana
GRAFANA_TAG=v6.4.4

KEYCLOAK_IMAGE=phx.ocir.io/stevengreenberginc/bfs/keycloak
ISTIO_CORE_DNS_PLUGIN_IMAGE=phx.ocir.io/stevengreenberginc/bfs/istio-coredns-plugin
ISTIO_CORE_DNS_PLUGIN_TAG=0.2-4fc7d69-12
ISTIO_CORE_DNS_IMAGE=container-registry.oracle.com/olcne/coredns
ISTIO_CORE_DNS_TAG=1.6.2
ISTIO_VERSION=1.4.6

KEYCLOAK_IMAGE_TAG=10.0.1-2fee5c4-3
KEYCLOAK_CHART_VERSION=8.2.2

MYSQL_IMAGE_TAG=8.0.20

NGINX_INGRESS_CONTROLLER_IMAGE=phx.ocir.io/stevengreenberginc/bfs/nginx-ingress-controller
NGINX_INGRESS_CONTROLLER_TAG=0.32-cf9d06b-18
NGINX_INGRESS_CONTROLLER_VERSION=1.27.0

NGINX_DEFAULT_BACKEND_IMAGE=phx.ocir.io/stevengreenberginc/bfs/nginx-ingress-default-backend
NGINX_DEFAULT_BACKEND_TAG=0.32-cf9d06b-18

RANCHER_IMAGE=phx.ocir.io/stevengreenberginc/bfs/rancher
RANCHER_VERSION=v2.4.3
RANCHER_TAG=v2.4.3-16152f5-15
