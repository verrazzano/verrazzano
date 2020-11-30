# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM container-registry.oracle.com/os/oraclelinux:7-slim@sha256:fcc6f54bb01fc83319990bf5fa1b79f1dec93cbb87db3c5a8884a5a44148e7bb AS build_base

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum install -y oracle-golang-release-el7 \
    && yum-config-manager --add-repo http://yum.oracle.com/repo/OracleLinux/OL7/developer/golang113/x86_64 \
    && yum install -y git gcc make golang-1.13.3-1.el7 \
    && yum clean all \
    && go version

# Compile to /usr/bin
ENV GOBIN=/usr/bin

# Set go path
ENV GOPATH=/go

ARG BUILDVERSION
ARG BUILDDATE

# Need to use specific WORKDIR to match verrazzano's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano
COPY . .

ENV CGO_ENABLED 0
RUN go version
RUN go env

RUN GO111MODULE=on go build \
    -mod=vendor \
    -ldflags '-extldflags "-static"' \
    -ldflags "-X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}" \
    -o /usr/bin/verrazzano-platform-operator .

# Create the verrazzano-platform-operator image
FROM container-registry.oracle.com/os/oraclelinux:7-slim@sha256:fcc6f54bb01fc83319990bf5fa1b79f1dec93cbb87db3c5a8884a5a44148e7bb

RUN yum update -y \
    && yum-config-manager --enable ol7_optional_latest \
    && yum-config-manager --enable ol7_addons \
    && yum install -y oracle-golang-release-el7 \
    && yum-config-manager --enable ol7_developer_golang113 \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum install -y openssl jq patch \
    && yum install -y oracle-olcne-release-el7 \
    && yum-config-manager --enable ol7_olcne11 \
    && yum install -y kubectl-1.17.9-1.0.5.el7.x86_64 \
    && yum install -y helm-3.1.1-1.0.2.el7.x86_64 \
    && yum clean all \
    && rm -rf /var/cache/yum

# Copy the operator binary
COPY --from=build_base /usr/bin/verrazzano-platform-operator /usr/local/bin/verrazzano-platform-operator

# Copy the Verrazzano install and uninstall scripts
WORKDIR /verrazzano
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/install ./install
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/uninstall ./uninstall
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/config/scripts/run.sh .
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/config/scripts/kubeconfig-template ./config/kubeconfig-template

RUN groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && chown -R 1000:verrazzano /verrazzano \
    && chmod +x /verrazzano/install/1-install-istio.sh \
    && chmod +x /verrazzano/install/2-install-system-components.sh \
    && chmod +x /verrazzano/install/3-install-verrazzano.sh \
    && chmod +x /verrazzano/install/4-install-keycloak.sh \
    && chmod +x /verrazzano/uninstall/uninstall-verrazzano.sh \
    && chmod +x /verrazzano/uninstall/uninstall-steps/0-uninstall-applications.sh \
    && chmod +x /verrazzano/uninstall/uninstall-steps/1-uninstall-istio.sh \
    && chmod +x /verrazzano/uninstall/uninstall-steps/2-uninstall-system-components.sh \
    && chmod +x /verrazzano/uninstall/uninstall-steps/3-uninstall-verrazzano.sh \
    && chmod +x /verrazzano/uninstall/uninstall-steps/4-uninstall-keycloak.sh \
    && chmod +x /verrazzano/run.sh \
    && mkdir /home/verrazzano \
    && mkdir -p go/src/github.com/verrazzano/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano \
    && chown 1000:verrazzano /usr/local/bin/verrazzano-platform-operator \
    && chmod 500 /usr/local/bin/verrazzano-platform-operator

# Copy source tree to image
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano go/src/github.com/verrazzano/verrazzano

USER 1000

ENTRYPOINT ["/verrazzano/run.sh"]
