# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

#
#  Code quality targets
#

.PHONY: check
check: install-linter
	$(LINTER) run

# find or download golangci-lint
.PHONY: install-linter
install-linter:
ifeq (, $(shell command -v golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.38.0
	$(eval LINTER=$(GOPATH)/bin/golangci-lint)
else
	$(eval LINTER=$(shell command -v golangci-lint))
endif

.PHONY: coverage
coverage:
	${SCRIPT_DIR}/coverage.sh html