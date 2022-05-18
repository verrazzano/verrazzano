# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

GO ?= GO111MODULE=on GOPRIVATE=github.com/verrazzano go

#
# CLI
#

##@ CLI

.PHONY: cli
cli: ## build the CLI
	$(GO) install


.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./cmd/...


