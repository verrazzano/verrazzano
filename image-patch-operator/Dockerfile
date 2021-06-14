# FIXME: just copying and pasting this copyright stuff???
# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

RUN yum update -y \
    && yum install -y unzip wget \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne11/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL7/olcne12/x86_64/ \
    && yum clean all \
    && rm -rf /var/cache/yum

# get WebLogic Image Tool and unzip
RUN wget https://github.com/oracle/weblogic-image-tool/releases/download/release-1.9.12/imagetool.zip \
    && unzip ./imagetool.zip

FROM ghcr.io/oracle/oraclelinux:7-slim

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum -y install tar gzip \
    && yum clean all \
    && rm -rf /var/cache/yum

ARG JDK_BINARY="${JDK_BINARY:-jdk-8u281-linux-x64.tar.gz}"
ENV JAVA_HOME=/usr/java
ENV PATH $JAVA_HOME/bin:$PATH
COPY ${JDK_BINARY} jdk.tar.gz
# ENV JDK_DOWNLOAD_SHA256=91310200f072045dc6cef2c8c23e7e6387b37c46e9de49623ce0fa461a24623d


RUN set -eux \
    # echo "Checking JDK hash"; \
    # echo "${JDK_DOWNLOAD_SHA256} *jdk.tar.gz" | sha256sum --check -; \
    echo "Installing JDK"; \
    mkdir -p "$JAVA_HOME"; \
    tar xzf jdk.tar.gz --directory "${JAVA_HOME}" --strip-components=1;
    # rm -f jdk.tar.gz;

# Copy the operator binary
# WORKDIR /

RUN groupadd -r verrazzano && useradd --no-log-init -r -g verrazzano -u 1000 verrazzano \
    && mkdir /home/verrazzano \
    && chown -R 1000:verrazzano /home/verrazzano

COPY --from=build_base --chown=verrazzano:verrazzano imagetool imagetool

USER 1000


ENTRYPOINT ["imagetool/bin/imagetool.sh"]

