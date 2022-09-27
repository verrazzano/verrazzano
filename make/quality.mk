# Copyright (C) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

#
#  Code quality targets
#
##@ Linting and coverage

GOLANGCI_LINT_VERSION=1.47.3

.PHONY: check
check: lint-go word-linter url-linter ## run all linters

.PHONY: lint-go
lint-go: check-golangci-lint
	golangci-lint --color never run --max-same-issues 25 --timeout 300s

.PHONY: check-golangci-lint
check-golangci-lint: install-golangci-lint ## run Go linters
	@{ \
		set -eu ; \
		ACTUAL_GOLANGCI_LINT_VERSION=$$(golangci-lint version --format short | sed -e 's%^v%%') ; \
		if [ "$${ACTUAL_GOLANGCI_LINT_VERSION}" != "${GOLANGCI_LINT_VERSION}" ] ; then \
			echo "Bad golangci-lint version $${ACTUAL_GOLANGCI_LINT_VERSION}, please install ${GOLANGCI_LINT_VERSION}" ; \
		fi ; \
	}

# find or download golangci-lint
.PHONY: install-golangci-lint
install-golangci-lint: ## install golangci-lint
	@{ \
		set -eu ; \
		if ! command -v golangci-lint ; then \
			curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${GOLANGCI_LINT_VERSION} ; \
		fi; \
	}

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
