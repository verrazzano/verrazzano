#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Common script used to automate the release process.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

# Release artifacts
declare -a releaseArtifacts=("operator.yaml"
                             "k8s-dump-cluster.sh"
                             "k8s-dump-cluster.sh.sha256"
                             "verrazzano-analysis-darwin-amd64.tar.gz"
                             "verrazzano-analysis-darwin-amd64.tar.gz.sha256"
                             "verrazzano-analysis-linux-amd64.tar.gz"
                             "verrazzano-analysis-linux-amd64.tar.gz.sha256")

# Validates whether OCI CLI is installed
function validate_oci_cli() {
     command -v oci >/dev/null 2>&1 || {
      echo "OCI cli not installed"
      return 1
    }
}

# Validates whether Github CLI is installed
function validate_github_cli() {
     command -v gh >/dev/null 2>&1 || {
      echo "Github cli not installed"
      return 1
    }
}

