# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

# Need to use specific WORKDIR to match verrazzano-cluster-operator's source packages
WORKDIR /root/go/src/github.com/verrazzano/cluster-operator
COPY . .

COPY out/linux_amd64/verrazzano-cluster-operator /usr/local/bin/verrazzano-cluster-operator

RUN chmod 500 /usr/local/bin/verrazzano-cluster-operator

# Create the verrazzano-cluster-operator image
FROM ghcr.io/oracle/oraclelinux:7-slim

RUN yum update -y \
    && yum clean all \
    && rm -rf /var/cache/yum

# Copy the operator binary
WORKDIR /

RUN groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-cluster-operator /usr/local/bin/verrazzano-cluster-operator

COPY --from=build_base /root/go/src/github.com/verrazzano/cluster-operator/THIRD_PARTY_LICENSES.txt /licenses/

USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-cluster-operator"]
