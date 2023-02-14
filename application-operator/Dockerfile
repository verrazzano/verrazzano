# Copyright (C) 2020, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:8-slim AS build_base

# Need to use specific WORKDIR to match verrazzano-application-operator's source packages
WORKDIR /root/go/src/github.com/verrazzano/application-operator
COPY . .

COPY out/linux_amd64/verrazzano-application-operator /usr/local/bin/verrazzano-application-operator

RUN chmod 500 /usr/local/bin/verrazzano-application-operator
RUN chmod 500 /root/go/src/github.com/verrazzano/application-operator/scripts/wait4webhook.sh

# Create the verrazzano-application-operator image
FROM ghcr.io/oracle/oraclelinux:8-slim AS final

RUN microdnf update -y \
    && microdnf clean all \
    && rm -rf /var/cache/yum /var/lib/rpm/*

RUN groupadd -r verrazzano  \
    && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

# Copy the operator binary
COPY --from=build_base --chown=verrazzano:verrazzano /root/go/src/github.com/verrazzano/application-operator/scripts/wait4webhook.sh /usr/local/bin/wait4webhook.sh
COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-application-operator /usr/local/bin/verrazzano-application-operator

COPY --from=build_base /root/go/src/github.com/verrazzano/application-operator/THIRD_PARTY_LICENSES.txt /licenses/

USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-application-operator"]
