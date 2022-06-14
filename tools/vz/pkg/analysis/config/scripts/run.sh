#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# TODO: add in an option to first dump a cluster and then analyze it

# The default mode is to analyze the cluster dumps which are mapped into.
/usr/local/bin/verrazzano-analysis $*
