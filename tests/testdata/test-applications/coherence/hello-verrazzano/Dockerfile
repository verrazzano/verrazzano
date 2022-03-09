# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

RUN yum update -y \
    && yum install -y tar unzip gzip \
    && yum clean all; rm -rf /var/cache/yum

ENV JAVA_HOME=/usr/java
ENV PATH $JAVA_HOME/bin:$PATH
ARG JDK_BINARY="${JDK_BINARY:-openjdk-11+28_linux-x64_bin.tar.gz}"
COPY ${JDK_BINARY} jdk.tar.gz
ENV JDK_DOWNLOAD_SHA256=3784cfc4670f0d4c5482604c7c513beb1a92b005f569df9bf100e8bef6610f2e

RUN set -eux \
    echo "Checking JDK hash"; \
    echo "${JDK_DOWNLOAD_SHA256} *jdk.tar.gz" | sha256sum --check -; \
    echo "Installing JDK"; \
    mkdir -p "$JAVA_HOME"; \
    tar xzf jdk.tar.gz --directory "${JAVA_HOME}" --strip-components=1; \
    rm -f jdk.tar.gz;

ARG JAR_FILE=target/coherence-spring-boot*.jar
COPY ${JAR_FILE} coherence-spring-boot.jar
ENTRYPOINT ["java","-jar","/coherence-spring-boot.jar"]
