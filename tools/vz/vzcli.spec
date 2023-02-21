# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

%global debug_package       %{nil}

%global APP_NAME            verrazzano-cli
%global APP_VERSION         %{?cliversion}
%global CLI_COMMIT          %{?clicommit}
%global RELEASE_VERSION     %{?rpmreleaseversion}
%global GO_VERSION          %{?goversion}
%global REPO_NAME           verrazzano
%global MOD_PATH            github.com/verrazzano

Name:                       %{APP_NAME}
Version:                    %{APP_VERSION}
Release:                    %{RELEASE_VERSION}%{?dist}
Source0:                    %{APP_NAME}-%{APP_VERSION}.tar.gz
Summary:                    Verrazzano CLI
Group:                      Development/Tools
License:                    Universal Permissive License v1.0
URL:                        https://github.com/verrazzano/verrazzano
Vendor:                     Oracle America
BuildRequires:              golang >= %{GO_VERSION}

%description
The Verrazzano CLI is a command-line utility that allows Verrazzano operators to query and manage a Verrazzano environment.

%prep

%setup -q -n vz

%build
GOPATH_SRC=%{_builddir}/go/src/%{MOD_PATH}/%{REPO_NAME}/tools/vz
%__mkdir_p $GOPATH_SRC
%__mkdir_p %{_builddir}/%{name}-%{version}/output/bin
%__rm -r $GOPATH_SRC
%__ln_s $PWD $GOPATH_SRC
pushd $GOPATH_SRC
GIT_COMMIT=%{CLI_COMMIT}
BUILD_DATE=${BUILD_DATE:-$(date +"%Y-%m-%dT%H:%M:%SZ")}
VERSION=%{version}
%ifarch x86_64
GOARCH=amd64
%else
GOARCH=arm64
%endif
VZ_DIR=github.com$(echo $PWD | sed 's/.*github.com//')
VERSION_DIR=${VZ_DIR}/cmd/version
CLI_GO_LDFLAGS="-X '${VERSION_DIR}.gitCommit=${GIT_COMMIT}' -X '${VERSION_DIR}.buildDate=${BUILD_DATE}' -X '${VERSION_DIR}.cliVersion=${VERSION}'"
GOOS=linux GOARCH=${GOARCH} GO111MODULE=on GOPRIVATE=%{MOD_PATH}/* go build \
        -mod vendor \
        -ldflags "${CLI_GO_LDFLAGS}" \
        -o out/linux_${GOARCH}/vz \
        main.go
popd
%__mv ${GOPATH_SRC}/out/linux_${GOARCH}/vz %{_builddir}/%{name}-%{version}/output/bin/
%__cp ${GOPATH_SRC}/vendor/github.com/verrazzano/verrazzano/LICENSE.txt %{_builddir}/vz/LICENSE

%install
install -D -p -m 555 %{_builddir}/%{name}-%{version}/output/bin/vz %{buildroot}/usr/bin/vz

%files
%license LICENSE THIRD_PARTY_LICENSES.txt
/usr/bin/vz

%changelog
* Mon Feb 27 2023 Asha Yarangatta <asha.yarangatta@oracle.com> - 1.0.0-1
- Initial release