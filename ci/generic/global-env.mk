# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

export GOPATH ?= /home/opc/go
export GO_REPO_PATH ?= ${GOPATH}/src/github.com/verrazzano
export CI_ROOT ?= ${VZ_ROOT}/ci
export CI_SCRIPTS_DIR ?= ${CI_ROOT}/scripts
export WORKSPACE ?= ${CURDIR}/workspace
export IMAGE_PULL_SECRET ?= verrazzano-container-registry
export DOCKER_REPO ?= 'ghcr.io'
export DOCKER_NAMESPACE ?= 'verrazzano'
