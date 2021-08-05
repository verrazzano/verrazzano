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
CHARTS_DIR=$(cd $SOURCE_DIR/../../thirdparty/charts; pwd -P)
VZ_CHARTS_DIR=$(cd $SOURCE_DIR/../../helm_config/charts; pwd -P)
VZ_OVERRIDES_DIR=$(cd $SOURCE_DIR/../../helm_config/overrides; pwd -P)

MANIFESTS_DIR=$(cd $SOURCE_DIR/../../thirdparty/manifests; pwd -P)

. ${SOURCE_DIR}/logging.sh

BOM_FILE=${BOM_FILE:-/verrazzano/platform-operator/verrazzano-bom.json}

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
    -X POST https://$rancher_hostname:$(get_nginx_nodeport)/v3-public/localProviders/local?action=login)
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
    -X POST https://$rancher_hostname:$(get_nginx_nodeport)/v3/token )
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

# Common function to create/update a generic secret from a literal string if it doesn't already exist
# $1 the secret name
# $2 the namespace for the secret
# $3 the password secret
function update_secret_from_literal() {
  local secret_name=$1
  local ns=$2
  local password_secret=$3

  kubectl apply -f <(echo "
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: ${secret_name}
  namespace: ${ns}
data:
  password: $(echo -n "${password_secret}"|base64)
")
}

# Get the current state of a helm chart
# $1 chart name
# $2 namespace
# $3 chart location
function get_deployment_status() {
  echo -n $(helm status $1 -n $2 -o json 2>/dev/null | jq -r .info.status || true)
}

# Uninstalls a chart if it is stuck in "failed" or "unknown" state
# $1 chart name
# $2 namespace
# $3 chart location
function reset_chart(){
  local ns=$2
  local chartName=$1
  local chartLocation=${3:-""}
  # status values: unknown, deployed, uninstalled, superseded, failed, uninstalling, pending-install, pending-upgrade or pending-rollback
  local deployment_status=$(get_deployment_status ${chartName} ${ns})
  log "Deployment status for ${ns}/${chartName}: ${deployment_status}"
  if [ "${deployment_status}" != "deployed" ]; then
      log "Resetting chart state for ${ns}/${chartName} at ${chartLocation} if necessary"
      helm template ${chartName} -n ${ns} ${chartLocation} 2>/dev/null |  kubectl delete -f - 2>/dev/null || true
      helm uninstall -n ${ns} ${chartName} 2>/dev/null
      return $?
  fi
  log "Chart ${ns}/${chartName} at ${chartLocation} status: ${deployment_status}"
  return 0
}

# Returns "true" if the requested chart/ns is deployed, "false" otherwise
# $1 chart name
# $2 namespace
function is_chart_deployed(){
  local ns=$2
  local chartName=$1
  local chartLocation=${3:-""}

  # Reset the chart state in case it's in a stuck/failed state
  reset_chart ${chartName} ${ns} ${chartLocation} || return $?

  # status values: unknown, deployed, uninstalled, superseded, failed, uninstalling, pending-install, pending-upgrade or pending-rollback
  local deployment_status=$(get_deployment_status ${chartName} ${ns})
  if [ "${deployment_status}" == "deployed" ]; then
    log "Chart ${ns}/${chartName} in ${chartLocation} is already deployed"
    return 0
  fi
  return 1
}

# Get the repo for Docker imageNames at the component/chart level from the BOM
# $1 the component (e.g. "istio")
# $2 the chart name (e.g. "istiocoredns")
function get_component_repo_from_bom() {
  local component=$1
  local chartName=$2
  cat ${BOM_FILE} | jq -r -c --arg C "${component}" --arg CH "${chartName}" \
    '.components[] | select(.name == $C) | .subcomponents[] | select(.name == $CH) | .repository'
}

# Get the full-repositoryrepo for Docker imageNames based on BOM and env var settings
# $1 the component (e.g. "istio")
# $2 the chart name (e.g. "istiocoredns")
function build_component_repo_name() {
  local component=$1
  local chartName=$2
  # Get the component-level repository from the BOM
  local repository=$(get_component_repo_from_bom $component $chartName)
  if [ -n "${IMAGE_REPO}" ]; then
    # If there's a user-supplied repo in the env, prepend it to the repo component-level repo we got from the BOM
    repository=${IMAGE_REPO}/${repository}
  fi
  echo ${repository}
}

# Dump the set of image elements from the BOM for a component and chart
# $1 the component (e.g. "istio")
# $2 the chart name (e.g. "istiocoredns")
function get_images_for_chart() {
  local component=$1
  local chartName=$2
  cat ${BOM_FILE} | jq -r -c --arg C "${component}" --arg CH "${chartName}" \
    '.components[] | select(.name == $C) | .subcomponents[] | select(.name == $CH) | .images[]'
}

# This function builds "--set" helm args to override imageNames using a bill of materials. The resulting
# HELM_SET_ARGS variable can be passed to helm to override one or more imageNames in a helm chart.
# $1 the component (e.g. "istio")
# $2 the chart name (e.g. "istiocoredns")
function build_image_overrides(){
  local component=$1
  local chartName=$2
  local registry=${REGISTRY}
  local bomFile=${BOM_FILE}

  # if registry is not overridden in environment, pull it from the BOM
  if [ -z "${registry}" ]; then
    registry=$(cat ${bomFile} | jq -r '.registry')
  fi

  # Build the full component-level repository from the BOM and the env
  local repository=$(build_component_repo_name $component $chartName)

  HELM_IMAGE_ARGS=""

  local images=($(get_images_for_chart $component $chartName))

  # build --set arg for each image in the chart
  for row in "${images[@]}"; do

    local image=$(echo $row | jq -r '.image')
    local tag=$(echo $row | jq -r '.tag')

    local helmRegKey=$(echo $row | jq -r '.helmRegKey')
    local helmRepoKey=$(echo $row | jq -r '.helmRepoKey')
    local helmImageKey=$(echo $row | jq -r '.helmImageKey')
    local helmTagKey=$(echo $row | jq -r '.helmTagKey')
    local helmFullImageKey=$(echo $row | jq -r '.helmFullImageKey')

    local fullImageKey=""

    if [ "${helmRegKey}" != "null" ]; then
      HELM_IMAGE_ARGS="${HELM_IMAGE_ARGS} --set ${helmRegKey}=${registry} "
    else
      fullImageKey="${registry}/"
    fi

    if [ "${helmRepoKey}" != "null" ]; then
      HELM_IMAGE_ARGS="${HELM_IMAGE_ARGS} --set ${helmRepoKey}=${repository} "
    else
      fullImageKey="${fullImageKey}${repository}/"
    fi

    if [ "${helmImageKey}" != "null" ]; then
      HELM_IMAGE_ARGS="${HELM_IMAGE_ARGS} --set ${helmImageKey}=${image} "
    else
      fullImageKey="${fullImageKey}${image}"
    fi

    if [ "${helmTagKey}" != "null" ]; then
      HELM_IMAGE_ARGS="${HELM_IMAGE_ARGS} --set ${helmTagKey}=${tag} "
    else
      fullImageKey="${fullImageKey}:${tag}"
    fi

    if [ "${helmFullImageKey}" != "null" ]; then
      HELM_IMAGE_ARGS="${HELM_IMAGE_ARGS} --set ${helmFullImageKey}=${fullImageKey} "
    fi

    HELM_RAW_IMAGE="${fullImageKey}"
  done
}

function generate_password() {
  local -i _default_size=16
  local -i _minsize=12
  local -i _maxsize=256
  local -i _pwsize=${1:-${_default_size}}
  [[ ${_pwsize} -lt ${_minsize} ]] && _pwsize=${_minsize}
  [[ ${_pwsize} -gt ${_maxsize} ]] && _pwsize=${_maxsize}
  dd if=/dev/urandom bs=${_pwsize} count=3 2>/dev/null | base64 | tr -d '+/=' | cut -c1-${_pwsize}
}

# Returns 0 if no slow image pulls are detected, otherwise returns 1
# $1 the namespace to check
function check_for_slow_image_pulls() {
  local pulling_count=$(kubectl get events -n $1 | grep Pulling | wc -l)
  local pulled_count=$(kubectl get events -n $1 | grep 'Successfully pulled' | wc -l)
  if [[ $pulling_count -eq $pulled_count ]]; then
    echo "Slow image pulls detected for namepaces $1"
	  return 0
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
####Constants for Docker imageNames, versions, tags
##################################################
GLOBAL_IMAGE_PULL_SECRET=verrazzano-container-registry
