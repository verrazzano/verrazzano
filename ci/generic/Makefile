# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ./global-env.mk
include ./kind.mk
include ./install.mk
include ./acceptance-tests.mk

.PHONY: all
all: setup install test upgrade uninstall cleanup

# Perform any setup prior to executing any other parts of the test
#
# For example, these can include
# - Kubernetes cluster(s)
# - LoadBalancer resources
# - DNS zones
.PHONY: setup
setup:
	@echo "Running test setup"
	make setup-kind

# Peforms an install of Verrazzano to the target system
.PHONY: install
install:
	@echo "Installing Verrazzano"
	make install-verrazzano

# Executes test suite(s) against the target Verrazzano install
test: export TEST_SUITES ?= verify-install/...
.PHONY: test
test:
	@echo "Running tests ${TEST_SUITES}"
	make run-test

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: upgrade
upgrade:
	@echo "Upgrade not implemented"

# Executes an update to the configuration of an installed Verrazzano instance
.PHONY: update
update:
	@echo "Update not implemented"

# Performs an uninstall of the Verrazzano instance from the cluster
.PHONY: uninstall
uninstall:
	@echo "Uninstall not implemented"

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: cleanup
cleanup: pipeline-artifacts clean-kind
	@echo "Running test cleanup"

