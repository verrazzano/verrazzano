#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/logging.sh

DEFAULT_CONFIG_FILE="$SCRIPT_DIR/config/config_defaults.json"

# The max length of the environment name passed in by the user.
ENV_NAME_LENGTH_LIMIT=10

# Read a JSON installation config file and output the JSON to stdout
function read_config() {
  local config_file=$1
  local config_json=$(cat $config_file)
  echo "$config_json"
}

# get_config_value outputs to stdout a configuration value, without the surrounding quotes
# Note: if the value requested is an array, it will return a JSON array - use get_config_array
# if you want a bash array.
function get_config_value() {
  set -o pipefail
  local jq_expr="$1"
  local config_val=$(echo "$CONFIG_JSON" | jq -r "$jq_expr")
  if [ $? -ne 0 ] || [ -z "$config_val" ] || [ "$config_val" == "null" ]; then
    config_val=$(echo "$DEFAULT_CONFIG_JSON" | jq -r "$jq_expr")
  fi
  if [ $? -ne 0 ]; then
    log "Error reading $jq_expr from config files"
    return 1
  fi
  if [ "$config_val" == "null" ]; then
    config_val=""
    return 0
  fi
  echo $config_val
  return 0
}

# get_config_array outputs to stdout, the contents of a configuration array element. It expects
# input expression to be in the form of ".someField.someArray[]" i.e. with trailing box brackets. Caller should enclose return
# value in parentheses to get the result as an array
# (e.g.) MY_CONFIG_ARRAY=($(get_config_array ".ingress.verrazzano.nginxInstallArgs[]"))
# Array elements will each be enclosed in quotes
function get_config_array() {
  set -o pipefail
  local jq_expr="$1"
  local config_array=($(echo $CONFIG_JSON | jq -rc $jq_expr | tr "\n" " "))
  if [ $? -ne 0 ] || [ ${#config_array[@]} -eq 0 ]; then
    config_array=($(echo $DEFAULT_CONFIG_JSON | jq -rc $jq_expr | tr "\n" " "))
  fi
  if [ $? -ne 0 ]; then
    log "Error reading $jq_expr from config files"
    return 1
  fi
  if [ ${#config_array[@]} -ne 0 ]; then
    echo "${config_array[@]}"
    return 0
  fi
  return 0
}

function validate_certificates_section() {
  set -o pipefail
  local jsonToValidate=$1
  local issuerType=$(get_config_value '.certificates.issuerType') || fail "Could not get certificates issuer type from config"
  if [ "$issuerType" == "ca" ]; then
    # must have .certificates.ca.secretName
    local secretName=$(get_config_value ".certificates.ca.secretName")
    if [ -z "$secretName" ]; then
      fail "The value .certificates.ca.secretName must be set to the tls Secret containing a signing key pair"
    fi
    local clusterResourceNamespace=$(get_config_value ".certificates.ca.clusterResourceNamespace")
    if [ -z "$clusterResourceNamespace" ]; then
      fail "The value .certificates.ca.clusterResourceNamespace must be set to the namespace where the secret named by 'secretName' is"
    fi
  elif [ "$issuerType" == "acme" ]; then
    # must have .certificates.acme.provider
    local provider=$(get_config_value ".certificates.acme.provider")
    if [ -z "$provider" ]; then
      fail "The value .certificates.acme.provider must be set"
    fi
    if [ "$provider" != "letsEncrypt" ]; then
      fail "The only .certificates.acme.provider spported is letsEncrypt"
    fi
  else
    fail "Unknown certificates issuer type $issuerType - valid values are ca and acme"
  fi
}

function validate_dns_section {
  set -o pipefail
  local jsonToValidate=$1
  local dnsType=$(get_config_value '.dns.type') || fail "Could not get dns type from config"
  if [ "$dnsType" == "external" ]; then
    #there should be an "external" section containing a suffix
    echo "$dnsJson" | jq '.external.suffix' || fail "For dns type external, a suffix is expected in section .dns.external.suffix of the config file"
  elif [ "$dnsType" == "oci" ]; then
    CHECK_VALUES=false
    value=$(get_config_value '.dns.oci.region')
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.region must be set to the OCI Region"
        CHECK_VALUES=true
    fi

    value=$(get_config_value '.dns.oci.tenancyOcid')
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.tenancyOcid must be set to the OCI Tenancy OCID"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.userOcid")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.userOcid must be set to the OCI User OCID"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.dnsZoneCompartmentOcid")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.dnsZoneCompartmentOcid must be set to the OCI Compartment OCID"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.fingerprint")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.fingerprint must be set to the OCI Fingerprint"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.privateKeyFile")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.privateKeyFile must be set to the OCI Private Key File"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.emailAddress")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.emailAddress must be set to your email address"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.dnsZoneOcid")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.dnsZoneOcid must be set to the OCI DNS Zone OCID"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.dnsZoneName")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.dnsZoneName must be set to the OCI DNS Zone Name"
        CHECK_VALUES=true
    fi

    if [ $CHECK_VALUES = true ]; then
        exit 1
    fi
  elif [ "$dnsType" != "xip.io" ]; then
    fail "Unknown dns type $dnsType - valid values are xip.io, oci and external"
  fi
}

function validate_environment_name {
  set -o pipefail
  local jsonToValidate=$1
  local env_name=$(get_config_value '.environmentName') || fail "Could not get environmentName from config"
  # check environment name length
  if [ ${#env_name} -gt $ENV_NAME_LENGTH_LIMIT ]; then
    fail "The environment name "${env_name}" is too long!  The maximum length is "${ENV_NAME_LENGTH_LIMIT}"."
  fi
}

# Make sure CONFIG_JSON and DEFAULT_CONFIG_JSON contain valid JSON
function validate_config_json {
  set -o pipefail
  local jsonToValidate=$1
  echo "$jsonToValidate" | jq . > /dev/null || fail "Failed to read installation config file contents. Make sure it is valid JSON"

  validate_environment_name "$jsonToValidate"
  validate_dns_section "$jsonToValidate"
  validate_certificates_section "$jsonToValidate"
}

function get_verrazzano_ingress_ip {
  local ingress_type=$(get_config_value ".ingress.type")
  if [ ${ingress_type} == "NodePort" ]; then
    ingress_ip=$(get_config_value ".ingress.nodePort.ingressIp")
  elif [ ${ingress_type} == "LoadBalancer" ]; then
    # Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
    ingress_ip=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    # In case of OLCNE, it would return null
    if [ ${ingress_ip} == "null" ]; then
      ingress_ip=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json  | jq -r '.spec.externalIPs[0]')
    fi
  fi
  echo ${ingress_ip}
}

function get_application_ingress_ip {
  local ingress_type=$(get_config_value ".ingress.type")
  if [ ${ingress_type} == "NodePort" ]; then
    ingress_ip=$(get_config_value ".ingress.nodePort.ingressIp")
  elif [ ${ingress_type} == "LoadBalancer" ]; then
    # Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
    ingress_ip=$(kubectl get svc istio-ingressgateway -n istio-system -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    # In case of OLCNE, it would return null
    if [ ${ingress_ip} == "null" ]; then
      ingress_ip=$(kubectl get svc istio-ingressgateway -n istio-system -o json  | jq -r '.spec.externalIPs[0]')
    fi
  fi
  echo ${ingress_ip}
}

function get_dns_suffix {
  local ingress_ip=$1
  local dns_type=$(get_config_value ".dns.type")
  if [ $dns_type == "xip.io" ]; then
    dns_suffix="${ingress_ip}".xip.io
  elif [ $dns_type == "oci" ]; then
    dns_suffix=$(get_config_value ".dns.oci.dnsZoneName")
  elif [ $dns_type == "external" ]; then
    dns_suffix=$(get_config_value ".dns.external.suffix")
  fi
  echo ${dns_suffix}
}

function get_nginx_helm_args_from_config {
  config_array_to_helm_args ".ingress.verrazzano.nginxInstallArgs[]"
}

function get_istio_helm_args_from_config {
  config_array_to_helm_args ".ingress.application.istioInstallArgs[]"
}

function config_array_to_helm_args {
  local install_args_config_name=$1
  local extra_install_args=($(get_config_array $install_args_config_name)) || return 1
  local helm_args=""
  if [ ${#extra_install_args[@]} -ne 0 ]; then
    for arg in "${extra_install_args[@]}"; do
      param_name=$(echo "$arg" | jq -r '.name')
      param_value=$(echo "$arg" | jq -r '.value')
      if [ ! -z "$param_name" ] && [ ! -z "$param_value" ]; then
        helm_args="$helm_args --set $param_name=$param_value"
      fi
    done
  fi
  echo $helm_args
  return 0
}
log "Reading default installation config file $DEFAULT_CONFIG_FILE"
DEFAULT_CONFIG_JSON="$(read_config $DEFAULT_CONFIG_FILE)"

if [ -z "$INSTALL_CONFIG_FILE" ]; then
  INSTALL_CONFIG_FILE=$DEFAULT_CONFIG_FILE
  CONFIG_JSON=$DEFAULT_CONFIG_JSON
else
  log "Reading installation config file $INSTALL_CONFIG_FILE"
  CONFIG_JSON="$(read_config $INSTALL_CONFIG_FILE)"
fi

validate_config_json "$CONFIG_JSON" || fail "Installation config is invalid"
validate_config_json "$DEFAULT_CONFIG_JSON" || fail "Default installation config is invalid"
echo env name is $(get_config_value ".environmentName")
echo dns type is $(get_config_value ".dns.type")
## Test cases - TODO remove before merging
#ENV_NAME=$(get_config_value ".environmentName")
#log "got environmentName value ${ENV_NAME}"
#EXTRA_ARG0=$(get_config_value ".ingress.verrazzano.nginxInstallArgs[0]")
#log "status $? and got 0th extra argument value ${EXTRA_ARG0}"
#EXTRA_ARGS_ARR=($(get_config_array ".ingress.verrazzano.nginxInstallArgs[]"))
#echo "status $? and got array [ ${EXTRA_ARGS_ARR[@]} ] containing ${EXTRA_ARGS_ARR[0]} and ${EXTRA_ARGS_ARR[1]}"
