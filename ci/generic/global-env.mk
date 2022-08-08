# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

export GOPATH ?= /home/opc/go
export GO_REPO_PATH ?= ${GOPATH}/src/github.com/verrazzano

export VZ_ROOT := ${GO_REPO_PATH}/verrazzano
export CI_ROOT ?= ${VZ_ROOT}/ci
export CI_SCRIPTS_DIR ?= ${CI_ROOT}/scripts

export WORKSPACE ?= ${HOME}/verrazzano-workspace
export KUBECONFIG ?= ${WORKSPACE}/test_kubeconfig

export IMAGE_PULL_SECRET ?= verrazzano-container-registry
export DOCKER_REPO ?= 'ghcr.io'
export DOCKER_NAMESPACE ?= 'verrazzano'
export TEST_ROOT ?= ${VZ_ROOT}/tests/e2e
export TEST_SCRIPTS_DIR ?= ${TEST_ROOT}/config/scripts
export DUMP_ROOT_DIRECTORY ?= ${WORKSPACE}/cluster-snapshots

export OCI_OS_ARTIFACT_BUCKET=build-failure-artifacts
export OCI_OS_COMMIT_BUCKET=verrazzano-builds-by-commit
export OCI_CLI_PROFILE ?= DEFAULT
