# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

# Need to use specific WORKDIR to match verrazzano-application-operator's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano/tools/psr
COPY . .

COPY out/linux_amd64/psr-backend /usr/local/bin/psr-backend

RUN chmod 500 /usr/local/bin/psr-backend

# Create the psr-backend image
FROM ghcr.io/oracle/oraclelinux:7-slim

RUN yum update -y \
    && yum clean all \
    && rm -rf /var/cache/yum

# Copy the operator binary
WORKDIR /

RUN groupadd -r psr && useradd --no-log-init -r -g psr -u 1000 psr \
    && mkdir /home/psr \
    && chown -R 1000:psr /home/psr

COPY --from=build_base --chown=psr:psr /usr/local/bin/psr-backend /usr/local/bin/psr-backend

USER 1000

ENTRYPOINT ["/usr/local/bin/psr-backend"]
