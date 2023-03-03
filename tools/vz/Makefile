# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
VZ_DIR:=github.com$(shell echo ${MAKEFILE_DIR} | sed 's/.*github.com//')
VERSION_DIR:=${VZ_DIR}/cmd/version

NAME:=vz

ifndef GIT_COMMIT
	GIT_COMMIT=$(shell git rev-parse HEAD)
endif

CLI_VERSION:=$(shell grep verrazzano-development-version ${MAKEFILE_DIR}/../../.verrazzano-development-version | cut -d= -f 2)
BUILD_DATE:=$(shell date +"%Y-%m-%dT%H:%M:%SZ")

ifdef RELEASE_VERSION
	CLI_VERSION=${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

# Required for building VZ CLI module stream
GO_MIN_VERSION:=1.19.1
RPM_RELEASE_VERSION:=1
RPM_SPEC_REPO:=${MAKEFILE_DIR}/out/verrazzano-cli
MAJOR:=$(shell echo "${CLI_VERSION}" | cut -d '.' -f 1)
MINOR:=$(shell echo "${CLI_VERSION}" | cut -d '.' -f 2)
MODULE_STREAM_VERSION:=${MAJOR}.${MINOR}
MODULE_VERSION:=$(shell date +%Y%m%d%H%M%S)
SED_OPTS=-i '' -e
ifeq ("Linux", "$(shell uname -s)")
	OS_RELEASE:=$(shell cat /etc/os-release | grep ^VERSION= | cut -d "=" -f 2)
	OS_VERSION:=$(shell echo ${OS_RELEASE} | cut -d "." -f 1)
	OS_UPDATE:=$(shell echo ${OS_RELEASE} | cut -d "." -f 2)
	SED_OPTS=-i'' -e
endif

DIST_DIR:=dist
ENV_NAME=vz
GO=GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go
CLI_GO_LDFLAGS=-X '${VERSION_DIR}.gitCommit=${GIT_COMMIT}' -X '${VERSION_DIR}.buildDate=${BUILD_DATE}' -X '${VERSION_DIR}.cliVersion=${CLI_VERSION}'

#
# CLI
#
.PHONY: run
run:
	$(GO) run ${GOPATH}/src/${VZ_DIR}/main.go
#
# Go build related tasks
#
.PHONY: go-build
go-build:
	GOOS=darwin GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/darwin_amd64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go
	GOOS=darwin GOARCH=arm64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/darwin_arm64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/linux_amd64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go
	GOOS=linux GOARCH=arm64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/linux_arm64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go

.PHONY: cli
cli: ## build the CLI
	$(GO) install -ldflags "${CLI_GO_LDFLAGS}" ./...

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./...

.PHONY: build-module-stream
build-module-stream:
	cp ../../go.mod .
	sed ${SED_OPTS} "s|^module github.com/verrazzano/verrazzano.*|module github.com/verrazzano/verrazzano/tools/vz|" go.mod
	echo "replace github.com/verrazzano/verrazzano => ../../" >> go.mod
	go mod tidy
	go mod vendor
	tar czf /tmp/verrazzano-cli-${CLI_VERSION}-${BUILD_DATE}.tar.gz -C .. vz
	rm go.mod go.sum
	rm -rf "${RPM_SPEC_REPO}"
	mkdir -p "${RPM_SPEC_REPO}"
	mv /tmp/verrazzano-cli-${CLI_VERSION}-${BUILD_DATE}.tar.gz ${RPM_SPEC_REPO}/verrazzano-cli-${CLI_VERSION}.tar.gz
	cp vzcli.spec ${RPM_SPEC_REPO}/verrazzano-cli.spec
	sed ${SED_OPTS} "s|^%global APP_VERSION.*|%global APP_VERSION ${CLI_VERSION}|" ${RPM_SPEC_REPO}/verrazzano-cli.spec
	sed ${SED_OPTS} "s|^%global CLI_COMMIT.*|%global CLI_COMMIT ${GIT_COMMIT}|" ${RPM_SPEC_REPO}/verrazzano-cli.spec
	sed ${SED_OPTS} "s|^%global RELEASE_VERSION.*|%global RELEASE_VERSION ${RPM_RELEASE_VERSION}|" ${RPM_SPEC_REPO}/verrazzano-cli.spec
	sed ${SED_OPTS} "s|^%global GO_VERSION.*|%global GO_VERSION ${GO_MIN_VERSION}|" ${RPM_SPEC_REPO}/verrazzano-cli.spec
	cd "${RPM_SPEC_REPO}" && \
	git init && \
	git add ${RPM_SPEC_REPO}/verrazzano-cli.spec && \
	git add ${RPM_SPEC_REPO}/verrazzano-cli-${CLI_VERSION}.tar.gz && \
	git commit -m "Verrazzano CLI RPM spec repo"
	cp vzcli-module.yaml ${MAKEFILE_DIR}/out/verrazzano-cli-stream.yaml
	sed ${SED_OPTS} "s|<MODULE_STREAM_VERSION>|${MODULE_STREAM_VERSION}|g" ${MAKEFILE_DIR}/out/verrazzano-cli-stream.yaml
	sed ${SED_OPTS} "s|<MODULE_VERSION>|${MODULE_VERSION}|g" ${MAKEFILE_DIR}/out/verrazzano-cli-stream.yaml
	sed ${SED_OPTS} "s|<RPM_SPEC_REPO>|${RPM_SPEC_REPO}|g" ${MAKEFILE_DIR}/out/verrazzano-cli-stream.yaml
	if [ "Linux" == "$(shell uname -s)" ]; then\
		sed ${SED_OPTS} "s|<OS_VERSION>|${OS_VERSION}|" ${MAKEFILE_DIR}/out/verrazzano-cli-stream.yaml;\
		sed ${SED_OPTS} "s|<OS_UPDATE>|${OS_UPDATE}|" ${MAKEFILE_DIR}/out/verrazzano-cli-stream.yaml;\
		sudo mbs-manager build_module_locally --stream ${MODULE_STREAM_VERSION} --offline --file ${MAKEFILE_DIR}/out/verrazzano-cli-stream.yaml;\
	fi
