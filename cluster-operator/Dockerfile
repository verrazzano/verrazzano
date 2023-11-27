# Copyright (C) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ARG BASE_IMAGE=ghcr.io/oracle/oraclelinux:8-slim
ARG FINAL_IMAGE=ghcr.io/verrazzano/ol8-static:v0.0.1-20231102152128-e7afc807

FROM $BASE_IMAGE AS build_base

# Need to use specific WORKDIR to match verrazzano-cluster-operator's source packages
WORKDIR /root/go/src/github.com/verrazzano/cluster-operator
COPY . .

COPY out/linux_amd64/verrazzano-cluster-operator /usr/local/bin/verrazzano-cluster-operator

RUN chmod 500 /usr/local/bin/verrazzano-cluster-operator \
    && groupadd -r verrazzano  \
    && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

# Create the verrazzano-cluster-operator image
FROM $FINAL_IMAGE AS final

# Copy users, groups and /home
COPY --from=build_base /etc/passwd /etc/passwd
COPY --from=build_base /etc/group /etc/group
COPY --from=build_base --chown=verrazzano:verrazzano /home/ /home/

# Copy the operator binary
COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-cluster-operator /usr/local/bin/verrazzano-cluster-operator

COPY --from=build_base /root/go/src/github.com/verrazzano/cluster-operator/THIRD_PARTY_LICENSES.txt /licenses/

USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-cluster-operator"]
