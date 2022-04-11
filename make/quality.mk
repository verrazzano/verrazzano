# Copyright (C) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

#
#  Code quality targets
#
##@ Linting and coverage

.PHONY: check
check: install-linter word-linter url-linter check-linter ## run all linters

.PHONY: check-linter
check-linter: install-linter ## run Go linters
	$(LINTER) --color never run

# find or download golangci-lint
.PHONY: install-linter
install-linter: ## install linters
ifeq (, $(shell command -v golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.38.0
	$(eval LINTER=$(GOPATH)/bin/golangci-lint)
else
	$(eval LINTER=$(shell command -v golangci-lint))
endif

# search for internal words that should not be in the repo
# check fails if res from http req is not successful (200)
# the actual command being executed in bash is "curl -sL https://bit.ly/3iIUcdL | grep -v '^\s*\(#\|$\)' | ..."
# additional "$" is to escape literal value in makefile
.PHONY: word-linter
word-linter: ## check for use of 'bad' words
	curl -sL -o /dev/null -w "%{http_code}" https://bit.ly/3iIUcdL | grep -q '200'
	! curl -sL https://bit.ly/3iIUcdL | grep -v '^\s*\(#\|$$\)' | grep -f /dev/stdin -r *

.PHONY: url-linter
url-linter: ## check for invalid URLs
	${TOOLS_DIR}/url_linter/invalid_url_linter.sh .

.PHONY: coverage
coverage:  ## test code coverage
	${SCRIPT_DIR}/coverage.sh html
