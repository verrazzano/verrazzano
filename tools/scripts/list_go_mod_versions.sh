#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
TMP_DIR=$(mktemp -d)
GO_MOD_FILE=$SCRIPT_DIR/../../go.mod

function table_row() {
    local all_versions=$(go list -m -mod=mod -versions $1)
    all_versions=${all_versions#"$1"}
    local in_use=$(echo $2 | cut -d- -f1 |cut -d+ -f1 )
    local found_in_use=false
    local newer_vers=""
    local avail_vers=""
    local latest=""
    for ver in $all_versions; do
        if [[ "$ver" == "$in_use" ]]; then
                found_in_use=true
        fi
        if [[ "$ver" != *"rc"* ]] && [[ "$ver" != *"beta"* ]] && [[ "$ver" != *"alpha"* ]]; then
            avail_vers="$avail_vers $ver"
            if [[ "$ver" != *"incompatible"* ]]; then
                latest=$ver # assuming the last one is the latest
            fi
            if [[ $found_in_use == true ]] ; then
                newer_vers="$newer_vers $ver"
            fi
        fi
    done
    if [[ "$newer_vers" == "" ]]; then
        local warnNotFound="in_use_version: $2 not found!!"
        newer_vers=$avail_vers
    fi
    echo $1,$2,$latest,$newer_vers
}

function inspect_go_mod() {
    local in_require=false
    while IFS= read -r line; do
        if [[ "$line" == *"require ("* ]]; then
            in_require=true
        else
            if [[ $in_require == true ]] && [[ "$line" != *")"* ]]; then
                table_row $line
            else
                in_require=false
            fi
        fi
    done < $1
}

echo "Component,In Use,Latest,Available"
inspect_go_mod $GO_MOD_FILE
