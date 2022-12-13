# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include env.mk

export IMAGE_PULL_SECRET ?= verrazzano-container-registry
export DOCKER_REPO ?= 'ghcr.io'
export DOCKER_NAMESPACE ?= 'verrazzano'

export OCI_OS_ARTIFACT_BUCKET=build-failure-artifacts
export OCI_OS_COMMIT_BUCKET=verrazzano-builds-by-commit
export OCI_CLI_PROFILE ?= DEFAULT

install-verrazzano: export INSTALL_CONFIG_FILE_KIND ?= ${TEST_SCRIPTS_DIR}/v1beta1/install-verrazzano-kind.yaml
install-verrazzano: export POST_INSTALL_DUMP ?= false
.PHONY: install-verrazzano
install-verrazzano:
	@echo "Running KIND install"
	${CI_SCRIPTS_DIR}/install_verrazzano.sh
