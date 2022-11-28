# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include global-env.mk

export CLUSTER_NAME ?= kind

setup-kind: export INSTALL_CONFIG_FILE_KIND ?= ${TEST_SCRIPTS_DIR}/v1beta1/install-verrazzano-kind.yaml
setup-kind: export CREATE_CLUSTER_USE_CALICO ?= false
setup-kind: export TESTS_EXECUTED_FILE ?= ${WORKSPACE}/tests_executed_file.tmp
setup-kind: export KUBERNETES_CLUSTER_VERSION ?= 1.24
.PHONY: setup-kind
setup-kind:
	@echo "Setup KIND cluster"
	${CI_SCRIPTS_DIR}/setup_kind.sh ${CREATE_CLUSTER_USE_CALICO}
#	${CI_SCRIPTS_DIR}/prepare_jenkins_at_environment.sh ${CREATE_CLUSTER_USE_CALICO} ${WILDCARD_DNS_DOMAIN} ${USE_DB_FOR_GRAFANA}

#clean-kind: export KUBECONFIG ?= "${WORKSPACE}/test_kubeconfig"
.PHONY: clean-kind
clean-kind:
	@echo "Cleanup kind cluster ${CLUSTER_NAME}, KUBECONFIG=${KUBECONFIG}"
	${CI_SCRIPTS_DIR}/cleanup_kind_clusters.sh ${CLUSTER_NAME} ${KUBECONFIG}

.PHONY: clean-kind-all
clean-kind-all:
	@echo "Deleting all kind clusters"
	kind delete clusters --all
