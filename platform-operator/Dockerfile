# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

ARG VERRAZZANO_APPLICATION_OPERATOR_IMAGE

# Need to use specific WORKDIR to match verrazzano's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano/platform-operator
COPY . .

COPY out/linux_amd64/verrazzano-platform-operator /usr/local/bin/verrazzano-platform-operator

RUN chmod 500 /usr/local/bin/verrazzano-platform-operator \
    && chmod +x scripts/install/*.sh \
    && chmod +x scripts/uninstall/*.sh \
    && chmod +x scripts/uninstall/uninstall-steps/*.sh

# Create the verrazzano-platform-operator image
FROM ghcr.io/oracle/oraclelinux:7-slim

ARG VERRAZZANO_APPLICATION_OPERATOR_IMAGE

# Install specific versions of device-mapper, device-mapper-libs and dracut as a temporary workaround
# by default yum tries to install a version that is non-existent on the OEL 7 yum mirrors
# Use both olcne11 and olcne12 versions since helm-3.1.1 is available only on olcne11 and
# kubectl 1.18.10-3 only on olcne12
RUN yum-config-manager --disable ol7_ociyum_config \
    && yum update -y \
    && yum-config-manager --disable ol7_ociyum_config \
    && yum install -y openssl jq patch \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne11/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne12/x86_64/ \
    && yum-config-manager --disable ol7_ociyum_config \
    && yum install -y kubectl-1.18.10-3.el7 \
    && yum install -y helm-3.1.1-1.0.2.el7.x86_64 \
    && yum clean all \
    && rm -rf /var/cache/yum

RUN groupadd -r verrazzano \
    && useradd --no-log-init -r -m -d /verrazzano -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

# Copy the operator binary
COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-platform-operator /usr/local/bin/verrazzano-platform-operator

# Copy the Verrazzano install and uninstall scripts
WORKDIR /verrazzano
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/thirdparty ./platform-operator/thirdparty
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/scripts/install ./platform-operator/scripts/install
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/scripts/uninstall ./platform-operator/scripts/uninstall
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/config/scripts/run.sh .
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/config/scripts/kubeconfig-template ./config/kubeconfig-template
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/helm_config ./platform-operator/helm_config

# Update the Verrazzano application operator image name in the chart override file
RUN sed -i"" -e "s|image:|image: ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}|g" /verrazzano/platform-operator/helm_config/overrides/verrazzano-application-operator-values.yaml

# Copy source tree to image
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/platform-operator/THIRD_PARTY_LICENSES.txt /licenses

USER 1000

ENTRYPOINT ["/verrazzano/run.sh"]
