# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Set WORKSPACE where you want all temporary test files to go, etc
# - default is ${HOME}/verrazzano-workspace
#export WORKSPACE=

# Set this to true to enable script debugging
#export VZ_TEST_DEBUG=true

# Used for the Github packages repo creds for the image pull secret for private branch builds
export DOCKER_CREDS_USR=my-github-user
export DOCKER_CREDS_PSW=$(cat ~/.github_token)
export DOCKER_REPO=ghcr.io

# Set these variables to set the Oracle Container Registry (OCR) secrets, typically your Oracle SSO
#export OCR_REPO=container-registry.oracle.com
#export OCR_CREDS_USR=me@oracle.com
#export OCR_CREDS_PSW=$(cat ~/.oracle_sso)

# Override where the Kubeconfig for the cluster is stored
#export KUBECONFIG= # Default is ${WORKSPACE}/test_kubeconfig

#### Platform Operator Settings 
#
# The following env vars control how the Platform Operator manifest is obtained
#
# - OPERATOR_YAML             - Specify a local platform operator manifest to use
# - VERRAZZANO_OPERATOR_IMAGE - Set this to use a specific operator image version, a manifest will be generated using this value
# - OCI_OS_NAMESPACE          - This must be set to the build system's ObjectStore namespace for the manifest-download default case
#
# If VERRAZZANO_OPERATOR_IMAGE and OPERATOR_YAML are not specified, the default is to download the operator manifest from
# the most recent, successful Jenkins build for the current branch
#
export OCI_OS_NAMESPACE=
#export OPERATOR_YAML=/home/mcico/tmp/workspace/downloaded-operator.yaml
#export VERRAZZANO_OPERATOR_IMAGE=ghcr.io/verrazzano/verrazzano-platform-operator-jenkins:1.4.0-20220607181222-af48cc1c
if [ -z "${OCI_OS_NAMESPACE}" ] && [ -z "" ] && [ -z "" ]; then
  echo "One of OCI_OS_NAMESPACE, OPERATOR_YAML, or VERRAZZANO_OPERATOR_IMAGE must be set in the local environment"
fi

### Cluster customizations
# export KIND_NODE_COUNT=3 # Default is 1
#export KUBERNETES_CLUSTER_VERSION=1.22 # default is 1.22

### Common VZ Install customizations
#
# INSTALL_CONFIG_FILE_KIND    - the VZ install CR to use, default is ${TEST_SCRIPTS_DIR}/v1beta1/install-verrazzano-kind.yaml
# INSTALL_PROFILE             - the install profile to use (default is "dev")
# VZ_ENVIRONMENT_NAME         - environmentName to use
# ENABLE_API_ENVOY_LOGGING    - enables debug in the Istio Envoy containers
# WILDCARD_DNS_DOMAIN         - an override for a user-specified wildcard DNS domain to use
#
#export INSTALL_CONFIG_FILE_KIND=
#export INSTALL_PROFILE=prod 

# Set this to any value for Gingko dry runs
#export DRY_RUN=true

# Required Weblogic/MySQL passwords for examples tests
#WEBLOGIC_PSW= # required by WebLogic application and console ingress test
#DATABASE_PSW= # required by console ingress test

