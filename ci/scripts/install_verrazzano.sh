#!/usr/bin/env bash
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Required env vars:
# INSTALL_CONFIG_FILE_KIND - source Verrazzano install configuration for KIND
# WORKSPACE - workspace for output files, temp files, etc
# TEST_SCRIPTS_DIR - Location of the E2E tests directory
# KUBECONFIG - kubeconfig path for the target cluster
# GO_REPO_PATH - Local path to the Verrazzano Github repo
#
# Indirect/optional env vars (used to process the installation config):
#
# INSTALL_PROFILE - Verrazzano profile, defaults to "dev"
# VZ_ENVIRONMENT_NAME - environmentName default
# EXTERNAL_ELASTICSEARCH - if "true" && VZ_ENVIRONMENT_NAME=="admin", sets Fluentd configuration to point to EXTERNAL_ES_SECRET and EXTERNAL_ES_URL
# SYSTEM_LOG_ID - configures Verrazzano for OCI logging using the specified OCI logging ID
# ENABLE_API_ENVOY_LOGGING - enables debug in the Istio Envoy containers
# WILDCARD_DNS_DOMAIN - an override for a user-specified wildcard DNS domain to use
# VERRAZZANO_INSTALL_LOGS_DIR - output location for the VZ install logs
#
set -o pipefail

if [ -z "$GO_REPO_PATH" ] ; then
  echo "GO_REPO_PATH must be set"
  exit 1
fi
if [ -z "$WORKSPACE" ]; then
  echo "WORKSPACE must be set"
  exit 1
fi
if [ -z "$TEST_SCRIPTS_DIR" ]; then
  echo "TEST_SCRIPTS_DIR must be set to the E2E test script directory location"
  exit 1
fi
if [ -z "${KUBECONFIG}" ]; then
  echo "KUBECONFIG must be set"
  exit 1
fi
if [ -z "$INSTALL_CONFIG_FILE_KIND" ]; then
  echo "INSTALL_CONFIG_FILE_KIND must be set to valid Verrazzano install file"
  exit 1
fi

scriptHome=$(dirname ${BASH_SOURCE[0]})

set -e
if [ -n "${VZ_TEST_DEBUG}" ]; then
  set -xv
fi

WILDCARD_DNS_DOMAIN=${WILDCARD_DNS_DOMAIN:-""}

ENABLE_API_ENVOY_LOGGING=${ENABLE_API_ENVOY_LOGGING:-"false"}

INSTALL_PROFILE=${INSTALL_PROFILE:-"dev"}
VERRAZZANO_INSTALL_LOGS_DIR=${VERRAZZANO_INSTALL_LOGS_DIR:-${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs}

# Configure the custom resource to install Verrazzano on Kind
${TEST_SCRIPTS_DIR}/process_kind_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND} ${WILDCARD_DNS_DOMAIN}

echo "Installing Verrazzano on Kind"
install_retries=0
install_failed=false
until kubectl apply -f ${INSTALL_CONFIG_FILE_KIND}; do
  install_retries=$((install_retries+1))
  sleep 6
  if [ $install_retries -ge 10 ] ; then
    echo "Installation Failed trying to apply the Verrazzano CR YAML"
    install_failed=true
  fi
done

# wait for Verrazzano install to complete
${TEST_SCRIPTS_DIR}/wait-for-verrazzano-install.sh
result=$?

if [ "${POST_INSTALL_DUMP}" == "true" ]; then
  echo "Generating post-install cluster-snapshot..."
  if [ -e ${WORKSPACE}/post-vz-install-cluster-snapshot ]; then
    echo "Removing existing post-install cluster-snapshot"
    rm -rf ${WORKSPACE}/post-vz-install-cluster-snapshot
  fi
  # TODO: Capture full cluster only when an environment variable CAPTURE_FULL_CLUSTER is set
  FULL_CLUSTER_DIR="${WORKSPACE}/post-vz-install-cluster-snapshot/full-cluster"
  mkdir -p ${FULL_CLUSTER_DIR}
  ANALYSIS_REPORT="analysis.report"
  ${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${FULL_CLUSTER_DIR} -r ${FULL_CLUSTER_DIR}/${ANALYSIS_REPORT}
  if [[ $result -ne 0 ]]; then
    echo "Post-install cluster snapshot failed"
    exit 1
  fi

  # Create a bug-report and run analysis tool on the bug-report
  # Requires environment variable KUBECONFIG or $HOME/.kube/config
  BUG_REPORT_DIR="${WORKSPACE}/post-vz-install-cluster-snapshot/bug-report"
  BUG_REPORT_FILE="${BUG_REPORT_DIR}/bug-report.tar.gz"
  mkdir -p ${BUG_REPORT_DIR}
  if [[ -x $GOPATH/bin/vz ]]; then
    $GOPATH/vz bug-report --report-file ${BUG_REPORT_FILE}
  else
    GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go bug-report --report-file ${BUG_REPORT_FILE}
  fi

  # Check if the bug-report exists
  if [ -f "${BUG_REPORT_FILE}" ]; then
    tar -xvf ${BUG_REPORT_FILE} -C ${BUG_REPORT_DIR}
    rm ${BUG_REPORT_FILE}

    # Run vz analyze on the extracted directory
    if [[ -x $GOPATH/bin/vz ]]; then
      $GOPATH/vz analyze --capture-dir ${BUG_REPORT_DIR} --report-format detailed --report-file ${BUG_REPORT_DIR}/${ANALYSIS_REPORT}
    else
      GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go analyze --capture-dir ${BUG_REPORT_DIR} --report-format detailed --report-file ${BUG_REPORT_DIR}/${ANALYSIS_REPORT}
    fi
  fi

fi

if [ "install_failed" == "true" ]; then
  exit 1
fi

if [ "${ENABLE_API_ENVOY_LOGGING}" == "true" ]; then
  vz_api_pod=$(kubectl get pod -n verrazzano-system -l app=verrazzano-authproxy --no-headers -o custom-columns=":metadata.name")
  if [ -z "$vz_api_pod" ]; then
    echo "Could not find verrazzano-authproxy pod, not enabling debug logging"
  else
    kubectl exec $vz_api_pod -c istio-proxy -n verrazzano-system -- curl -X POST http://localhost:15000/logging?level=debug
  fi
  nginx_ing_pod=$(kubectl get pod -n ingress-nginx -l app.kubernetes.io/component=controller --no-headers -o custom-columns=":metadata.name")
  if [ -z "$nginx_ing_pod" ]; then
    echo "Could not find nginx ingress controller pod, not enabling debug logging"
  else
    kubectl exec $nginx_ing_pod -c istio-proxy -n ingress-nginx -- curl -X POST http://localhost:15000/logging?level=debug
  fi
fi

exit 0
