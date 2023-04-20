#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#


RELEASE=${1}

GIT_RELEASE_COMMIT_ID=$(git rev-parse release-"$RELEASE")
echo "$RELEASE:$GIT_RELEASE_COMMIT_ID" >> ${workspace}/releaseAndCommits.txt
                            