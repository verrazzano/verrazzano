# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ./global-env.mk
include ./env.mk

install-verrazzano: export INSTALL_CONFIG_FILE_KIND ?= ${TEST_SCRIPTS_DIR}/v1beta1/install-verrazzano-kind.yaml
install-verrazzano: export POST_INSTALL_DUMP ?= false
.PHONY: install-verrazzano
install-verrazzano:
	@echo "Running KIND install"
	${CI_SCRIPTS_DIR}/install_verrazzano.sh ${WILDCARD_DNS_DOMAIN}
