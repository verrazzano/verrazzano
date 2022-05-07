# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:8-slim as builder-env

RUN microdnf update -y && \
    microdnf clean all && \
    rm -rf /var/cache/yum/*

WORKDIR /root/go/src/github.com/verrazzano/verrazzano/verrazzano-backup
COPY out/linux_amd64/ .
COPY inject.sh .

RUN chmod +x inject.sh && \
    chmod +x verrazzano-backup

ENTRYPOINT ["./inject.sh"]
