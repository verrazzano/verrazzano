# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ARG JDK_INSTALLER_BINARY="${JDK_INSTALLER_BINARY:-jdk-8u281-linux-x64.tar.gz}"

# Build stage for required software installation
FROM ghcr.io/oracle/oraclelinux:8-slim AS build_base

ARG JDK_INSTALLER_BINARY

RUN microdnf update -y \
    && microdnf install -y unzip wget tar gzip \
    && microdnf clean all

# Fetch and unzip WebLogic Image Tool
RUN wget https://github.com/oracle/weblogic-image-tool/releases/download/release-1.9.12/imagetool.zip \
    && unzip ./imagetool.zip

# Setup for JDK installation
ENV JAVA_HOME=/usr/java
COPY ./installers/${JDK_INSTALLER_BINARY} ./installers/${JDK_INSTALLER_BINARY}
ENV JDK_DOWNLOAD_SHA256=85e8c7da7248c7450fb105567a78841d0973597850776c24a527feb02ef3e586

# Install JDK
RUN set -eux \
    echo "Checking JDK hash"; \
    echo "${JDK_DOWNLOAD_SHA256} installers/${JDK_INSTALLER_BINARY}" | sha256sum --check -; \
    echo "Installing JDK"; \
    mkdir -p "$JAVA_HOME"; \
    tar xzf installers/${JDK_INSTALLER_BINARY} --directory "${JAVA_HOME}" --strip-components=1; \
    rm -f installers/${JDK_INSTALLER_BINARY}

# Final image for deploying WebLogic Image Tool
FROM ghcr.io/oracle/oraclelinux:8-slim

# Install the podman for WIT and dependencies
RUN microdnf update -y \
    && microdnf install -y podman \
    && microdnf reinstall -y shadow-utils \
    && microdnf clean all

WORKDIR /home/verrazzano

RUN groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir -p /home/verrazzano/cache \
    && chown -R 1000:verrazzano /home/verrazzano \
    # For Podman in rootless mode \
    && echo verrazzano:100000:65536 >> /etc/subuid \
    && echo verrazzano:100000:65536 >> /etc/subgid

# Copy over JDK installation
ENV JAVA_HOME=/usr/java
ENV PATH $JAVA_HOME/bin:$PATH
COPY --from=build_base --chown=verrazzano:verrazzano ${JAVA_HOME} ${JAVA_HOME}

# Copy over WebLogic Image Tool installation and wrapper script
COPY --from=build_base --chown=verrazzano:verrazzano /imagetool ./imagetool
COPY --chown=verrazzano:verrazzano ./v8o-imagetool.sh .
RUN chmod +x ./v8o-imagetool.sh

COPY THIRD_PARTY_LICENSES.txt /licenses/

USER 1000

ENTRYPOINT ["./v8o-imagetool.sh"]
