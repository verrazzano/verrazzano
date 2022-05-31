# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ./global-env.mk
include ./kind.mk
include ./install.mk

.PHONY: all
all: setup install test upgrade uninstall cleanup

# Perform any setup prior to executing any other parts of the test
#
# For example, these can include
# - Kubernetes cluster(s)
# - LoadBalancer resources
# - DNS zones
.PHONY: setup
setup: setup-kind
	@echo "Running test setup"

# Peforms an install of Verrazzano to the target system
.PHONY: install
install: install-verrazzano

# Executes test suite(s) against the target Verrazzano install
.PHONY: test
test:
	@echo "Running tests"

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: upgrade
upgrade:
	@echo "Running Verrazzano upgrade"

# Executes an update to the configuration of an installed Verrazzano instance
.PHONY: update
update:
	@echo "Running Verrazzano update"

# Performs an uninstall of the Verrazzano instance from the cluster
.PHONY: uninstall
uninstall:
	@echo "Running Verrazzano uninstall"

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: cleanup
cleanup: clean-kind
	@echo "Running test cleanup"

