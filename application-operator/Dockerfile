# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Create the verrazzano-application-operator image
FROM ghcr.io/oracle/oraclelinux:7-slim

RUN yum update -y \
    && yum clean all \
    && rm -rf /var/cache/yum \
    && groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

WORKDIR /

COPY --chown=verrazzano:verrazzano --chmod=500 out/linux_amd64/verrazzano-application-operator /usr/local/bin/verrazzano-application-operator
COPY THIRD_PARTY_LICENSES.txt /licenses/

USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-application-operator"]
