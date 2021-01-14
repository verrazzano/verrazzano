# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

# Need to use specific WORKDIR to match verrazzano-application-operator's source packages
WORKDIR /root/go/src/github.com/verrazzano/oam-application-operator
COPY . .

COPY out/linux_amd64/verrazzano-application-operator /usr/local/bin/verrazzano-application-operator

RUN chmod 500 /usr/local/bin/verrazzano-application-operator

# Create the verrazzano-application-operator image
FROM ghcr.io/oracle/oraclelinux:7-slim

RUN yum update -y \
    && yum-config-manager --enable ol7_optional_latest \
    && yum-config-manager --enable ol7_addons \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum install -y openssl jq \
    && yum install -y oracle-olcne-release-el7 \
    && yum-config-manager --enable ol7_olcne11 \
    && yum install -y kubectl-1.17.9-1.0.5.el7.x86_64 \
    && yum install -y helm-3.1.1-1.0.2.el7.x86_64 \
    && yum clean all \
    && rm -rf /var/cache/yum


# Copy the operator binary
WORKDIR /

RUN groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-application-operator /usr/local/bin/verrazzano-application-operator

# Copy source tree to image
COPY --from=build_base /root/go/src/github.com/verrazzano/oam-application-operator/THIRD_PARTY_LICENSES.txt /licenses

USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-application-operator"]
