# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Create the bom-validator image
FROM ghcr.io/oracle/oraclelinux:7-slim

# These are not used by the analysis, but they will be used by the k8s-dump-cluster.sh when
# the option is added to use that.
# Install specific versions of device-mapper, device-mapper-libs and dracut as a temporary workaround
# by default yum tries to install a version that is non-existent on the OEL 7 yum mirrors
# Use olcne13, which is required for kubectl 1.20.6-2
RUN yum update -y \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne13/x86_64/ \
    && yum install -y kubectl-1.20.6-2.el7 \
    && yum clean all \
    && rm -rf /var/cache/yum

RUN groupadd -r verrazzano \
    && useradd --no-log-init -r -m -d /verrazzano -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

# Copy the analysis binary
COPY ./bom-validator /usr/local/bin/bom-validator

USER 1000

ENTRYPOINT ["/usr/local/bin/bom-validator"]
