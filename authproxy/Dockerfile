# Copyright (C) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ARG BASE_IMAGE=ghcr.io/oracle/oraclelinux:8-slim
FROM $BASE_IMAGE AS build_base

# Need to use specific WORKDIR to match verrazzano-authproxy's source packages
WORKDIR /root/go/src/github.com/verrazzano/authproxy
COPY . .

COPY out/linux_amd64/verrazzano-authproxy /usr/local/bin/verrazzano-authproxy

RUN chmod 500 /usr/local/bin/verrazzano-authproxy

# Create the verrazzano-authproxy image
FROM $BASE_IMAGE AS final

RUN rm -rf /var/cache/yum /var/lib/rpm/* \
    && groupadd -r verrazzano  \
    && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

# Copy the authproxy binary
COPY --from=build_base --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-authproxy /usr/local/bin/verrazzano-authproxy

COPY --from=build_base /root/go/src/github.com/verrazzano/authproxy/THIRD_PARTY_LICENSES.txt /licenses/

USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-authproxy"]
