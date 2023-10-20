# Copyright (c) 2020, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ARG ISTIO_ISTIOCTL_IMAGE

ARG BASE_IMAGE=ghcr.io/oracle/oraclelinux:8-slim
FROM $BASE_IMAGE AS build_base

RUN microdnf update -y && \
    microdnf clean all && \
    rm -rf /var/cache/yum/*

# Need to use specific WORKDIR to match verrazzano's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano/platform-operator
COPY . .

COPY out/verrazzano-platform-operator /usr/local/bin/verrazzano-platform-operator

RUN chmod 500 /usr/local/bin/verrazzano-platform-operator \
    && chmod +x scripts/install/*.sh \
    && chmod +x scripts/*.sh

# istioctl image to copy the istioctl binary to the final image
FROM $ISTIO_ISTIOCTL_IMAGE AS istio_istioctl

# Create the verrazzano-platform-operator image
ARG BASE_IMAGE=build_base
FROM $BASE_IMAGE AS final

ARG REPOS_PATH=/root/go/src/github.com/verrazzano/verrazzano/platform-operator/repos
ARG ARCH=x86_64
ARG VERRAZZANO_PLATFORM_OPERATOR_IMAGE

COPY --from=build_base $REPOS_PATH/$ARCH/*.repo /etc/yum.repos.d/

COPY --from=istio_istioctl /usr/local/bin/istioctl /usr/local/bin/istioctl

RUN microdnf install -y --setopt=install_weak_deps=0 --setopt=tsflags=nodocs openssl jq kubectl-1.26.6-2.el8 \
    && microdnf clean all \
    && rm -rf /var/cache/yum /var/lib/rpm/* \
    && groupadd -r verrazzano \
    && useradd --no-log-init -r -m -d /verrazzano -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano \
    && curl --proto "=https" -L https://github.com/verrazzano/cluster-api/releases/download/v1.5.0/clusterctl-linux-amd64 -o /usr/local/bin/clusterctl \
    && chmod +x /usr/local/bin/clusterctl

# Copy the operator binary
COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-platform-operator /usr/local/bin/verrazzano-platform-operator

# Copy the Verrazzano install and uninstall scripts
WORKDIR /verrazzano
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/thirdparty ./platform-operator/thirdparty
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/manifests ./platform-operator/manifests
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/scripts/wait4webhook.sh ./platform-operator/scripts/wait4webhook.sh
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/scripts/hooks ./platform-operator/scripts/hooks
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/scripts/install ./platform-operator/scripts/install
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/config/scripts/run.sh .
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/config/scripts/kubeconfig-template ./config/kubeconfig-template
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/helm_config ./platform-operator/helm_config
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/out/generated-verrazzano-bom.json ./platform-operator/verrazzano-bom.json
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/capi/bootstrap-ocne ./capi/bootstrap-ocne
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/capi/control-plane-ocne ./capi/control-plane-ocne
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/capi/infrastructure-oci ./capi/infrastructure-oci
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/capi/cluster-api ./capi/cluster-api
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/capi/addon-verrazzano ./capi/addon-verrazzano
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/out/generated-catalog.yaml ./platform-operator/manifests/catalog/catalog.yaml

COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/platform-operator/THIRD_PARTY_LICENSES.txt /licenses/

# set the default VPO image in values.yaml for the VPO helm chart
RUN  sed -i -e "s|image:|image: $VERRAZZANO_PLATFORM_OPERATOR_IMAGE|g" ./platform-operator/helm_config/charts/verrazzano-platform-operator/values.yaml

USER 1000

ENTRYPOINT ["/verrazzano/run.sh"]
