# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

#
#  Code quality targets
#

.PHONY: check
check: go-fmt go-vet go-ineffassign go-lint

.PHONY: go-fmt
go-fmt:
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/ | grep -v /pkg/assets/) > error.txt
	if [ -s error.txt ]; then\
		cat error.txt;\
		rm error.txt;\
		exit 1;\
	fi
	rm error.txt

.PHONY: go-vet
go-vet:
	$(GO) vet $(shell go list ./... | grep -v github.com/verrazzano/verrazzano-application-operator/pkg/assets)

.PHONY: go-lint
go-lint:
	@{ \
	set -eu ; \
	GOLINT_VERSION=$$(go list -m -f '{{.Version}}' golang.org/x/lint) ; \
	${GO} get golang.org/x/lint/golint@$${GOLINT_VERSION} ; \
	}
	golint -set_exit_status $(shell go list ./... | grep -v github.com/verrazzano/verrazzano-application-operator/pkg/assets)

.PHONY: go-ineffassign
go-ineffassign:
	@{ \
	set -eu ; \
	INEFFASSIGN_VERSION=$$(go list -m -f '{{.Version}}' github.com/gordonklaus/ineffassign) ; \
	${GO} get github.com/gordonklaus/ineffassign@$${INEFFASSIGN_VERSION} ; \
	}
	ineffassign $(shell go list ./...)

.PHONY: coverage
coverage:
	./build/coverage.sh html