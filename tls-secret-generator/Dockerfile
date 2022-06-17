# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim as builder

# Need to use specific WORKDIR to match verrazzano-tls-secret-generator's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano/tls-secret-operator
COPY . .

COPY out/linux_amd64/verrazzano-tls-secret-generator /usr/local/bin/verrazzano-tls-secret-generator

RUN chmod 500 /usr/local/bin/verrazzano-tls-secret-generator

# Create the verrazzano-tls-secret-generator image
FROM ghcr.io/oracle/oraclelinux:7-slim

RUN yum update -y \
    && yum clean all \
    && rm -rf /var/cache/yum

WORKDIR /

RUN groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

COPY --from=builder --chown=verrazzano:verrazzano /usr/local/bin/verrazzano-tls-secret-generator /usr/local/bin/verrazzano-tls-secret-generator

COPY --from=builder /root/go/src/github.com/verrazzano/verrazzano/tls-secert-generator/THIRD_PARTY_LICENSES.txt /licenses/

USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-tls-secret-generator"]
