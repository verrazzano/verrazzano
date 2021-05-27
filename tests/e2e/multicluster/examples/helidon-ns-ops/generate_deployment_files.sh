#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# create output dir
mkdir -p $NS_SOURCE_DIR

# create project file
yq eval ".metadata.name"=\"hello-helidon-ns\" $HH_SOURCE_DIR/verrazzano-project.yaml > $NS_SOURCE_DIR/verrazzano-project.yaml
yq -i eval ".spec.template.namespaces[0].metadata.name"=\"hello-helidon-ns\" $NS_SOURCE_DIR/verrazzano-project.yaml
# create component file
yq eval ".spec.template.spec.workload.metadata.namespace"=\"hello-helidon-ns\" $HH_SOURCE_DIR/mc-hello-helidon-comp.yaml > $NS_SOURCE_DIR/mc-hello-helidon-comp.yaml
yq -i eval ".metadata.namespace"=\"hello-helidon-ns\" $NS_SOURCE_DIR/mc-hello-helidon-comp.yaml
# create app file
yq eval ".metadata.namespace"=\"hello-helidon-ns\" $HH_SOURCE_DIR/mc-hello-helidon-app.yaml > $NS_SOURCE_DIR/mc-hello-helidon-app.yaml
