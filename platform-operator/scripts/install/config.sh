#!/usr/bin/env bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
CONFIG_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
. $CONFIG_SCRIPT_DIR/logging.sh

DEFAULT_CONFIG_FILE="$CONFIG_SCRIPT_DIR/config/config_defaults.json"

INSTALL_OVERRIDES_DIR="${CONFIG_SCRIPT_DIR}/install-overrides"
EFFECTIVE_CONFIG_VALUES="${INSTALL_OVERRIDES_DIR}/effective.config.json"

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
# The configuration value is read from the following files in the given order
# - custom resource
# - install-overrides/config.${profile}.json
# - install-overrides/config.json
function get_config_value() {
  set -o pipefail
  local jq_expr="$1"
  local config_val=$(echo "$CONFIG_JSON" | jq -r "$jq_expr")
  if [ $? -ne 0 ]; then
    log "Error reading $jq_expr from config files"
    return 1
  fi
  if [ "$config_val" == "null" ]; then
    config_val=""
  fi

  # check if it is defined in the json files under install-overrides
  local config_override_value=$(get_override_config_value "$jq_expr")
  if [ "$config_override_value" != "" ]; then
    config_val=$config_override_value
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
    local email=$(get_config_value ".certificates.acme.emailAddress")
    if [ -z "$email" ]; then
        echo "For acme, the value .certificates.acme.emailAddress must be set to your email address"
        CHECK_VALUES=true
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
    local suffix=$(get_config_value ".dns.external.suffix")
    if [ -z "$suffix" ]; then
      fail "For dns type external, a suffix is expected in section .dns.external.suffix of the config file"
    fi
  elif [ "$dnsType" == "wildcard" ]; then
    #there should be an "wildcard" section containing a domain
    local domain=$(get_config_value ".dns.wildcard.domain")
    if [ -z "$domain" ]; then
      fail "For dns type wildcard, a domain is expected in section .dns.wildcard.domain of the config file"
    fi
  elif [ "$dnsType" == "oci" ]; then
    CHECK_VALUES=false
    value=$(get_config_value '.dns.oci.ociConfigSecret')
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.ociConfigSecret must be set to the OCI Configuration secret name"
        CHECK_VALUES=true
    fi

    value=$(get_config_value ".dns.oci.dnsZoneCompartmentOcid")
    if [ -z "$value" ]; then
        echo "For dns type oci, the value .dns.oci.dnsZoneCompartmentOcid must be set to the OCI Compartment OCID"
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
  else
    fail "Unknown dns type $dnsType - valid values are wildcard, oci and external"
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

# Make sure CONFIG_JSON contain valid JSON
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
    # on MAC and Windows, container IP is not accessible.  Port forwarding from 127.0.0.1 to container IP is needed.
    #ingress_ip="127.0.0.1"
    ingress_ip=$(kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o json | jq -r 'items[0].status.hostIP')
  elif [ ${ingress_type} == "LoadBalancer" ]; then
    # Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
    ingress_ip=$(kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    # In case of OLCNE, it would return null
    if [ ${ingress_ip} == "null" ]; then
      ingress_ip=$(kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o json  | jq -r '.spec.externalIPs[0]')
    fi
  fi
  echo ${ingress_ip}
}

function get_application_ingress_ip {
  local ingress_type=$(get_config_value ".ingress.type")
  if [ ${ingress_type} == "NodePort" ]; then
    # on MAC and Windows, container IP is not accessible.  Port forwarding will be needed.
    ingress_ip=$(kubectl -n istio-system get pods --selector app=istio-ingressgateway,istio=ingressgateway -o jsonpath='{.items[0].status.hostIP}')
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
  if [ $dns_type == "wildcard" ]; then
    dns_suffix="${ingress_ip}".$(get_config_value ".dns.wildcard.domain")
  elif [ $dns_type == "oci" ]; then
    dns_suffix=$(get_config_value ".dns.oci.dnsZoneName")
  elif [ $dns_type == "external" ]; then
    dns_suffix=$(get_config_value ".dns.external.suffix")
  fi
  echo ${dns_suffix}
}

function get_application_ingress_http_port {
  local ingress_type=$(get_config_value ".ingress.type")
  if [ ${ingress_type} == "NodePort" ]; then
    http_port=$(kubectl get service -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
  elif [ ${ingress_type} == "LoadBalancer" ]; then
    http_port=80
  fi
  echo ${http_port}
}

function get_application_ingress_https_port {
  local ingress_type=$(get_config_value ".ingress.type")
  if [ ${ingress_type} == "NodePort" ]; then
    https_port=$(kubectl get service -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].nodePort}')
  elif [ ${ingress_type} == "LoadBalancer" ]; then
    https_port=443
  fi
  echo ${https_port}
}

function get_nginx_helm_args_from_config {
  if [ ! -z "$(get_config_value '.ingress.verrazzano')" ] && [ ! -z "$(get_config_value '.ingress.verrazzano.nginxInstallArgs')" ]; then
    config_array_to_helm_args ".ingress.verrazzano.nginxInstallArgs[]" || return 1
  fi
}

function get_istio_helm_args_from_config {
  if [ ! -z "$(get_config_value '.ingress.application')" ] && [ ! -z "$(get_config_value '.ingress.application.istioInstallArgs')" ]; then
    config_array_to_helm_args ".ingress.application.istioInstallArgs[]" || return 1
  fi
}

function get_keycloak_helm_args_from_config {
  if [ ! -z "$(get_config_value '.keycloak')" ] && [ ! -z "$(get_config_value '.keycloak.keycloakInstallArgs')" ]; then
    config_array_to_helm_args ".keycloak.keycloakInstallArgs[]" || return 1
  fi
}

function get_mysql_helm_args_from_config {
  if [ ! -z "$(get_config_value '.keycloak.mysql')" ] && [ ! -z "$(get_config_value '.keycloak.mysql.mySqlInstallArgs')" ]; then
    config_array_to_helm_args ".keycloak.mysql.mySqlInstallArgs[]" || return 1
  fi
}

function get_verrazzano_helm_args_from_config {
  if [ ! -z "$(get_config_value '.verrazzanoInstallArgs')" ]; then
    config_array_to_helm_args ".verrazzanoInstallArgs[]" || return 1
  fi
}

function config_array_to_helm_args {
  local install_args_config_name=$1
  local extra_install_args=($(get_config_array $install_args_config_name)) || return 1
  local helm_args=""
  if [ ${#extra_install_args[@]} -ne 0 ]; then
    for arg in "${extra_install_args[@]}"; do
      param_name=$(echo "$arg" | jq -r '.name')
      param_value=$(echo "$arg" | jq -r '.value')
      param_set_string=$(echo "$arg" | jq -r '.setString')
      if [ ! -z "$param_name" ]; then
        if [ "$param_set_string" == "true" ]; then
          helm_args="$helm_args --set-string $param_name=$param_value"
        else
          helm_args="$helm_args --set $param_name=$param_value"
        fi
      fi
    done
  fi
  echo $helm_args
  return 0
}

function get_verrazzano_ports_spec() {
  local ports_spec=""
  if [ ! -z "$(get_config_value '.ingress.verrazzano')" ] && [ ! -z "$(get_config_value '.ingress.verrazzano.ports')" ]; then
    local port_mappings=($(get_config_array ".ingress.verrazzano.ports[]"))
    local port_mappings_len=${#port_mappings[@]}
    if [ $port_mappings_len -ne 0 ]; then
      printf -v joined '%s,' "${port_mappings[@]}"
      ports_spec="{\"spec\": {\"ports\": [ ${joined%,} ] }}"
    fi
  fi
  echo $ports_spec
}

function get_install_profile() {
  local profile=$(get_config_value '.profile')
  if [ -z "${profile}" ]; then
    error "The value .profile must be set in the config file"
    exit 1
  else
    echo "${profile}"
  fi
}

function get_acme_environment() {
  if [ -z "$(get_config_value '.certificates.acme.environment')" ]; then
    echo "production"
  else
    get_config_value ".certificates.acme.environment"
  fi
}

# rancher needs to be accessed by the scripts running in-cluster
# --resolve rancher.my-env.127.0.0.1.nip.io:nginx_node_port:nginx_host_ip
function get_rancher_resolve() {
  local rancher_hostname=$1
  local resolve="--resolve ${rancher_hostname}:$(get_nginx_nodeport):$(get_nginx_hostip)"
  echo ${resolve}
}

function get_rancher_in_cluster_host() {
  local rancher_hostname=$1
  local rancher_in_cluster_host=${rancher_hostname}
  if [ $(get_config_value ".ingress.type") == "NodePort" ]; then
    rancher_in_cluster_host=$(get_nginx_hostip)
  fi
  echo ${rancher_in_cluster_host}
}

function get_nginx_hostip() {
  local hostIP=$(kubectl -n ingress-nginx get pods --selector app.kubernetes.io/name=ingress-nginx,app.kubernetes.io/component=controller -o jsonpath='{.items[0].status.hostIP}')
  echo ${hostIP}
}

function get_nginx_nodeport() {
  local nodePort=$(kubectl get service -n ingress-nginx ingress-controller-ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.name=="https")].nodePort}')
  echo ${nodePort}
}

# Merge the default config json under install/install-overrides with the corresponding override for the profile
function compute_effective_override() {
  local profile=$(get_install_profile)
  local default_config="${INSTALL_OVERRIDES_DIR}/config.json"
  local profile_config_override="${INSTALL_OVERRIDES_DIR}/config.${profile}.json"
  if [ ! -f "${profile_config_override}" ]; then
    error "The file ${profile_config_override} does not exist"
    return 1
  fi
  if [ -f "$EFFECTIVE_CONFIG_VALUES" ]; then
   rm "$EFFECTIVE_CONFIG_VALUES"
  fi
  jq -rs 'reduce .[] as $item ({}; . * $item)' $default_config $profile_config_override > $EFFECTIVE_CONFIG_VALUES
}

# Read the value for a given key from effective.config.json
function get_override_config_value() {
  local config_val=""
  # Return an empty string when effective.config.json is not there - during uninstall and when validate_config_json
  # calls get_config_value.
  if [ ! -f "$EFFECTIVE_CONFIG_VALUES" ]; then
    echo $config_val
    return 0
  fi

  local jq_expr="$1"
  config_val=$(jq -r "$jq_expr" $EFFECTIVE_CONFIG_VALUES)
  if [ $? -ne 0 ]; then
    echo "Error reading $jq_expr from config files"
    return 1
  fi
  if [ "$config_val" == "null" ]; then
    config_val=""
  fi
  echo $config_val
  return 0
}

# Return true if Verrazzano Operator and Verrazzano Monitoring Operator should be enabled or not
function is_vo_vmo_enabled() {
  local profile=$(get_install_profile)
  if [ "$profile" == "managed-cluster" ]; then
   echo "true"
  elif [ "$(is_elasticsearch_console_enabled)" == "true" ] || [ "$(is_kibana_console_enabled)" == "true" ] || [ "$(is_prometheus_console_enabled)" == "true" ] || [ "$(is_grafana_console_enabled)" == "true" ]; then
    echo "true"
  else
    echo "false"
  fi
}

# Return the value for the key rancher.enabled
function is_rancher_enabled() {
  local rancher_enabled=$(get_config_value '.rancher.enabled')
  echo ${rancher_enabled}
}

# Return the value for the key keycloak.enabled
function is_keycloak_enabled() {
  local keycloak_enabled=$(get_config_value '.keycloak.enabled')
  echo ${keycloak_enabled}
}

# Return the value for the key fluentd.enabled
function is_fluentd_enabled() {
  local fluentd_enabled=$(get_config_value '.fluentd.enabled')
  echo ${fluentd_enabled}
}

function get_fluentd_extra_volume_mounts {
  if [ ! -z "$(get_config_value '.fluentd')" ] && [ ! -z "$(get_config_value '.fluentd.fluentdInstallArgs')" ]; then
    config_array_to_helm_args ".fluentd.fluentdInstallArgs[]" || return 1
  fi
}

function get_fluentd_elasticsearch_options {
  local helm_args=""
  if [ ! -z "$(get_config_value '.fluentd')" ]; then
    if [ ! -z "$(get_config_value '.fluentd.elasticsearchURL')" ]; then
      helm_args="$helm_args --set logging.elasticsearchURL=$(get_config_value '.fluentd.elasticsearchURL')"
    fi
    if [ ! -z "$(get_config_value '.fluentd.elasticsearchSecret')" ]; then
      helm_args="$helm_args --set logging.elasticsearchSecret=$(get_config_value '.fluentd.elasticsearchSecret')"
    fi
  fi
  echo $helm_args
  return 0
}

# Parse the verrazzanoInstallArgs from the config.json and return the value of the configuration value
function parse_vz_install_args() {
  local install_args=$1
  local config_value=$2
  local vz_install_args=($(get_config_array $install_args)) || return 1
  if [ ${#vz_install_args[@]} -ne 0 ]; then
    for arg in "${vz_install_args[@]}"; do
      param_name=$(echo "$arg" | jq -r '.name')
      param_value=$(echo "$arg" | jq -r '.value')
      if [ ! -z "$param_name" ] && [ ! -z "$param_value" ] && [ "$param_name" == "$config_value" ]; then
        echo "$param_value"
        return 0
      fi
    done
  fi
  return 0
}

# The function get_console_enabled outputs to stdout a configuration value, without the surrounding quotes, in the
# following order.
# - check whether "<config value>.enabled is there under verrazzanoInstallArgs in config.json, which is based on the
#   configuration specified by the custom resource
# - check whether "consolesEnabled.<config value> is set to true in install-overrides/config.${profile}.json
# - check whether "consolesEnabled.<config value> is set to true in install-overrides/config.json
# The configuration value in the
function get_console_enabled() {
  local jq_expr="$1"
  local config_value=""
  if [ ! -z "$(get_config_value '.verrazzanoInstallArgs')" ]; then
    # Read the value from the config.json
    config_value="$(parse_vz_install_args '.verrazzanoInstallArgs[]' "$jq_expr.enabled")"
    if [ -z "$config_value" ] || [ "$config_value" == "null" ] || [ "$config_value" == "" ]; then
      config_value=""
    else
      echo "${config_value}"
      return 0
    fi
  fi
  if [ "$config_value" == "" ] ; then
    # Read the value from the config overrides
    config_value=$(get_config_value ".consolesEnabled.${jq_expr}")
  fi
  echo "${config_value}"
}

# Return whether Elasticsearch console is enabled
function is_elasticsearch_console_enabled() {
  local es_enabled=$(get_console_enabled 'elasticSearch')
  echo ${es_enabled}
}

# Return whether Grafana console is enabled
function is_grafana_console_enabled() {
  local grafana_enabled=$(get_console_enabled 'grafana')
  echo ${grafana_enabled}
}

# Return whether Kibana console is enabled
function is_kibana_console_enabled() {
  local kibana_enabled=$(get_console_enabled 'kibana')
  echo ${kibana_enabled}
}

# Return whether Verrazzano console is enabled
function is_vz_console_enabled() {
  local console_enabled=$(get_console_enabled 'console')
  echo ${console_enabled}
}

# Return whether Prometheus console is enabled. For a managed-cluster profile, the console enabled flag is set to false
# in the overrides, as the admin cluster should be used to query the metrics from the managed clusters.
function is_prometheus_console_enabled() {
  local prometheus_enabled=$(get_console_enabled 'prometheus')
  echo ${prometheus_enabled}
}

# Return a flag indicating whether this is an installation leveraging OCI DNS
function is_oci_dns() {
  local dns_type=$(get_config_value '.dns.type')
  if [ "$dns_type" == "oci" ]; then
    echo "true"
  else
    echo "false"
  fi
}

# Return a flag indicating whether this is an installation leveraging external DNS (OLCNE)
function is_external_dns() {
  local dns_type=$(get_config_value '.dns.type')
  if [ "$dns_type" == "external" ]; then
    echo "true"
  else
    echo "false"
  fi
}

if [ -z "$INSTALL_CONFIG_FILE" ]; then
  INSTALL_CONFIG_FILE=$DEFAULT_CONFIG_FILE
fi
log "Reading installation config file $INSTALL_CONFIG_FILE"
if [ ! -f "$INSTALL_CONFIG_FILE" ]; then
  fail "The file $INSTALL_CONFIG_FILE does not exist"
fi
CONFIG_JSON="$(read_config $INSTALL_CONFIG_FILE)"

validate_config_json "$CONFIG_JSON" || fail "Installation config is invalid"

compute_effective_override  || fail "Failure to merge the install overrides"
