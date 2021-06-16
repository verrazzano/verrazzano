# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ARG JDK_BINARY="${JDK_BINARY:-installers/jdk-8u291-linux-x64.tar.gz}"

# Build stage for WebLogic Image Tool installation
FROM ghcr.io/oracle/oraclelinux:7-slim AS build_wit

RUN yum update -y \
    && yum install -y unzip wget \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne11/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne12/x86_64/ \
    && yum clean all \
    && rm -rf /var/cache/yum

RUN wget https://github.com/oracle/weblogic-image-tool/releases/download/release-1.9.12/imagetool.zip \
    && unzip ./imagetool.zip

# Build stage for JDK installation
FROM ghcr.io/oracle/oraclelinux:7-slim AS build_jdk

ARG JDK_BINARY

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum -y install tar gzip \
    && yum clean all \
    && rm -rf /var/cache/yum

ENV JAVA_HOME=/usr/java
COPY ${JDK_BINARY} ${JDK_BINARY}
ENV JDK_DOWNLOAD_SHA256=c5052d2e1dd9621a44658ef06be145c5cdfcd7ea956c0c9d655ccd64e79c8613

RUN set -eux \
    echo "Checking JDK hash"; \
    echo "${JDK_DOWNLOAD_SHA256} ${JDK_BINARY}" | sha256sum --check -; \
    echo "Installing JDK"; \
    mkdir -p "$JAVA_HOME"; \
    tar xzf ${JDK_BINARY} --directory "${JAVA_HOME}" --strip-components=1; \
    rm -f ${JDK_BINARY}

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
COPY --chown=verrazzano:verrazzano \
    ${JDK_BINARY} \
    installers/fmw_12.2.1.4.0_wls_lite_Disk1_1of1.zip \
    installers/weblogic-deploy.zip \
    /imagetool/installers/

USER 1000

ENTRYPOINT ["imagetool/bin/imagetool.sh"]

