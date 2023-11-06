# Copyright (c) 2020, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ARG ISTIO_ISTIOCTL_IMAGE

ARG BASE_IMAGE=ghcr.io/oracle/oraclelinux:8-slim
ARG FINAL_IMAGE=ghcr.io/verrazzano/ol8-base:v0.0.1-20231102152128-e7afc807

FROM $BASE_IMAGE AS build_base

ARG VERRAZZANO_PLATFORM_OPERATOR_IMAGE

# Need to use specific WORKDIR to match verrazzano's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano/platform-operator
COPY . .

COPY out/linux_amd64/verrazzano-platform-operator /usr/local/bin/verrazzano-platform-operator

RUN chmod 500 /usr/local/bin/verrazzano-platform-operator \
    && chmod +x scripts/install/*.sh \
    && chmod +x scripts/*.sh

# copy olcne repos needed to install kubectl
RUN cp /root/go/src/github.com/verrazzano/verrazzano/platform-operator/repos/*.repo /etc/yum.repos.d/

RUN microdnf install -y --setopt=install_weak_deps=0 --setopt=tsflags=nodocs jq kubectl-1.26.6-1.el8 \
    && microdnf clean all \
    && rm -rf /var/cache/yum /var/lib/rpm/* \
    && groupadd -r verrazzano \
    && useradd --no-log-init -r -m -d /verrazzano -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano \
    && curl --proto "=https" -L https://github.com/verrazzano/cluster-api/releases/download/v1.5.3/clusterctl-linux-amd64 -o /usr/local/bin/clusterctl \
    && chmod +x /usr/local/bin/clusterctl

# set the default VPO image in values.yaml for the VPO helm chart
RUN  sed -i -e "s|image:|image: $VERRAZZANO_PLATFORM_OPERATOR_IMAGE|g" /root/go/src/github.com/verrazzano/verrazzano/platform-operator/helm_config/charts/verrazzano-platform-operator/values.yaml

# create a verrazzano directory with the correct ownership and permissions so we can copy it to the final image
RUN mkdir -p /tmp/stage/verrazzano && \
    chmod 700 /tmp/stage/verrazzano

# istioctl image to copy the istioctl binary to the final image
FROM $ISTIO_ISTIOCTL_IMAGE AS istio_istioctl

# Create the verrazzano-platform-operator image
FROM $FINAL_IMAGE AS final

COPY --from=istio_istioctl /usr/local/bin/istioctl /usr/local/bin/istioctl

# copy installed tools and dependent libraries
COPY --from=build_base /usr/bin/jq /usr/bin/jq
COPY --from=build_base /usr/lib64/libjq* /usr/lib64/
COPY --from=build_base /usr/lib64/libonig* /usr/lib64/

COPY --from=build_base /usr/bin/awk /usr/bin/awk
COPY --from=build_base /usr/lib64/libsigsegv* /usr/lib64/
COPY --from=build_base /usr/lib64/libreadline* /usr/lib64/
COPY --from=build_base /usr/lib64/libmpfr* /usr/lib64/

COPY --from=build_base /usr/bin/curl /usr/bin/curl
COPY --from=build_base /usr/lib64/libcurl* /usr/lib64/
COPY --from=build_base /usr/lib64/libnghttp* /usr/lib64/
COPY --from=build_base /usr/lib64/libidn* /usr/lib64/
COPY --from=build_base /usr/lib64/libssh* /usr/lib64/
COPY --from=build_base /usr/lib64/libpsl* /usr/lib64/
COPY --from=build_base /usr/lib64/libgssapi_krb5* /usr/lib64/
COPY --from=build_base /usr/lib64/libkrb5* /usr/lib64/
COPY --from=build_base /usr/lib64/libk5crypto* /usr/lib64/
COPY --from=build_base /usr/lib64/libcom_err* /usr/lib64/
COPY --from=build_base /usr/lib64/libldap* /usr/lib64/
COPY --from=build_base /usr/lib64/liblber* /usr/lib64/
COPY --from=build_base /usr/lib64/libbrotlidec* /usr/lib64/
COPY --from=build_base /usr/lib64/libunistring* /usr/lib64/
COPY --from=build_base /usr/lib64/libkeyutils* /usr/lib64/
COPY --from=build_base /usr/lib64/libbrotlicommon* /usr/lib64/
COPY --from=build_base /usr/lib64/libsasl2* /usr/lib64/
COPY --from=build_base /usr/lib64/libcrypt* /usr/lib64/

COPY --from=build_base /usr/bin/kubectl /usr/bin/kubectl
COPY --from=build_base /usr/local/bin/clusterctl /usr/local/bin/clusterctl

# copy Verrazzano home directory
COPY --from=build_base /home/ /home/

# copy users/groups added in build_base stage
COPY --from=build_base /etc/passwd /etc/passwd
COPY --from=build_base /etc/group /etc/group
COPY --from=build_base /bin/bash /bin/bash

# copy the operator binary
COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-platform-operator /usr/local/bin/verrazzano-platform-operator

# copy licenses
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/platform-operator/THIRD_PARTY_LICENSES.txt /licenses/

# copy the Verrazzano install scripts
USER 1000

WORKDIR /verrazzano
COPY --from=build_base --chown=verrazzano:verrazzano /tmp/stage/ /
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/thirdparty ./platform-operator/thirdparty
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/manifests ./platform-operator/manifests
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/platform-operator/scripts/ ./platform-operator/scripts/
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

ENTRYPOINT ["/verrazzano/run.sh"]
