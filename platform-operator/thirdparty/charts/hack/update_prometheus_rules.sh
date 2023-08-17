#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

# Update the Prometheus Rule resources to include "verrazzano_cluster" in "on" and "by" clauses, otherwise the
# cluster label is dropped from alerts and it is impossible to determine which cluster fired the alert. This
# script should be run whenever we upgrade any upstream charts that include Prometheus Rules (currently the
# kube-prometheus-stack and thanos charts).
sed -i -E 's/ by ?\(([^\)]*)\)/ by (\1, verrazzano_cluster)/gI' `grep -Rl PrometheusRule $SCRIPT_DIR/../* | grep templates`
sed -i -E 's/ on ?\(([^\)]*)\)/ on (\1, verrazzano_cluster)/gI' `grep -Rl PrometheusRule $SCRIPT_DIR/../* | grep templates`
