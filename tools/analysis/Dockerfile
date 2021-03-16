# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

# Need to use specific WORKDIR to match verrazzano's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano/tools/analysis
COPY . .

COPY out/linux_amd64/verrazzano-analysis /usr/local/bin/verrazzano-analysis

RUN chmod 500 /usr/local/bin/verrazzano-analysis

# Create the verrazzano-analysis image
FROM ghcr.io/oracle/oraclelinux:7-slim

# These are not used by the analysis, but they will be used by the k8s-dump-cluster.sh when
# the option is added to use that.
# Install specific versions of device-mapper, device-mapper-libs and dracut as a temporary workaround
# by default yum tries to install a version that is non-existent on the OEL 7 yum mirrors
# Use both olcne11 and olcne12 versions since helm-3.1.1 is available only on olcne11 and
# kubectl 1.18.10-3 only on olcne12
RUN yum update -y \
    && yum install -y openssl jq patch \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne11/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne12/x86_64/ \
    && yum install -y kubectl-1.18.10-3.el7 \
    && yum install -y helm-3.1.1-1.0.2.el7.x86_64 \
    && yum clean all \
    && rm -rf /var/cache/yum

RUN groupadd -r verrazzano \
    && useradd --no-log-init -r -m -d /verrazzano -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

# Copy the analysis binary
COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-analysis /usr/local/bin/verrazzano-analysis

# Copy the Verrazzano dump scripts
WORKDIR /verrazzano
#COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/tools/scripts ./tools/scripts

# Copy the config scripts
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/tools/analysis/config/scripts/run.sh ./run.sh
#COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/verrazzano/tools/analysis/config/scripts/kubeconfig-template ./kubeconfig-template

# Copy source tree to image
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano/tools/analysis/THIRD_PARTY_LICENSES.txt /licenses

USER 1000

ENTRYPOINT ["/verrazzano/run.sh"]
