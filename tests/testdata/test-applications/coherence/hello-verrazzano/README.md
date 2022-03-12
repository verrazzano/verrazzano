## Coherence Application

Sample Coherence application, for testing the Coherence workload.

## Requires

[Maven](https://maven.apache.org/download.cgi)

## Create a Docker image
The Dockerfile provided in this example uses an Oracle Linux image as the base image, which doesn't include the Java Development Kit (JDK).
The Dockerfile expects `openjdk-<version>_linux-x64_bin.tar.gz` in the project root directory, which is available on the [OpenJDK General-Availability Releases](https://jdk.java.net/archive/) page.
Please check the exact version of the JDK from the Dockerfile and install accordingly.

    $ cd <project root directory>
    $ mvn package
    $ docker build -t coherence-application .

The application provides a few endpoints:  
* http://localhost:8080/   // An index page  
* http://localhost:8080/hello   // A page displaying one or more messages
* http://localhost:8080/actuator  // Spring Boot actuator endpoint  
* http://localhost:8080/actuator/prometheus   // Prometheus endpoint  

A new message can be posted on /postMessage using curl as  
curl -X POST -H "Content-type: application/json" -d "Good morning, Verrazzano." "http://localhost:8080/postMessage"

Copyright (c) 2022, Oracle and/or its affiliates.