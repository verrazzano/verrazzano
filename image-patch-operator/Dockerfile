# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ARG JDK_BINARY="${JDK_BINARY:-jdk-8u281-linux-x64.tar.gz}"

# Build stage for WebLogic Image Tool installation
FROM ghcr.io/oracle/oraclelinux:7-slim AS build_wit

RUN yum update -y \
    && yum install -y unzip wget \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne11/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne12/x86_64/ \
    && yum clean all \
    && rm -rf /var/cache/yum

# Download WebLogic Image Tool and unzip
RUN wget https://github.com/oracle/weblogic-image-tool/releases/download/release-1.9.12/imagetool.zip \
    && unzip ./imagetool.zip

# Build stage for JDK installation
FROM ghcr.io/oracle/oraclelinux:7-slim AS build_jdk

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum -y install tar gzip \
    && yum clean all \
    && rm -rf /var/cache/yum

ARG JDK_BINARY
ENV JAVA_HOME=/usr/java
COPY ${JDK_BINARY} /

RUN set -eux \
    echo "Installing JDK"; \
    mkdir -p "$JAVA_HOME"; \
    tar xzf ${JDK_BINARY} --directory "${JAVA_HOME}" --strip-components=1;

# Final image for deploying WebLogic Image Tool
FROM ghcr.io/oracle/oraclelinux:7-slim

ARG JDK_BINARY

# copy over JDK installation
ENV JAVA_HOME=/usr/java
ENV PATH $JAVA_HOME/bin:$PATH
COPY --from=build_jdk --chown=verrazzano:verrazzano ${JAVA_HOME} ${JAVA_HOME}

# copy over WebLogic Image Tool installation
RUN groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano
COPY --from=build_wit --chown=verrazzano:verrazzano imagetool imagetool

# copy over installers
COPY --from=build_jdk --chown=verrazzano:verrazzano ${JDK_BINARY} /imagetool/installers/

USER 1000

ENTRYPOINT ["imagetool/bin/imagetool.sh"]

