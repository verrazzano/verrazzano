#!/bin/bash
#
# Copyright (C) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z $1 ] || [ ! -d $1 ]; then
    echo "Please provide a valid directory to check for URLs"
    exit 1
fi

#creating a temporary arrangement for linter
URL_LINTER_TEMPDIR=""
function init_url_linter() {
    local _template=url_linter_temp_XXX
    if [ -n "${WORKSPACE}" ] ; then
        _template="${WORKSPACE}/${_template}"
    fi
    export URL_LINTER_TEMPDIR=$(mktemp -d ${_template})
    if [ -z $URL_LINTER_TEMPDIR ] || [ ! -d $URL_LINTER_TEMPDIR ]; then
        echo "Failed to initialize temporary directory"
        exit 1
    fi
}

#cleanup temporary files once we are done
function cleanup_url_linter() {
    if [ -z $URL_LINTER_TEMPDIR ] || [ ! -d $URL_LINTER_TEMPDIR ]; then
        return
    fi
    rm -rf $URL_LINTER_TEMPDIR
}

init_url_linter

HELPER_SCRIPT_PATH="$SCRIPT_DIR/url_response.sh"

#color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
ORANGE='\033[0;33m'
NC='\033[0m'

#fetching url from repo
grep --exclude-dir '.git' --exclude-dir e2e --exclude-dir thirdparty --exclude-dir vendor --exclude-dir test* -I -Eorh "(http|https)://[a-zA-Z0-9./?=_%#:-]*" $1 | grep -v 'License' | grep -v 'REDACTED' | grep -v 'localhost' | grep -v 'Binary file' | grep -v '\%' | grep -v '127' | sort -u > $URL_LINTER_TEMPDIR/urls.out
sed -i -e 's/\.$//g' $URL_LINTER_TEMPDIR/urls.out

#calling the helper script in parallel to check for http response
cat $URL_LINTER_TEMPDIR/urls.out | xargs -P 6 -L1 "$HELPER_SCRIPT_PATH" "$URL_LINTER_TEMPDIR"

echo "--------------------------------------------"
echo -e "${ORANGE}Locations for dead urls:${NC}"
if [ ! -f $URL_LINTER_TEMPDIR/urls_404.out ]; then
    echo -e "${GREEN}No dead urls found${NC}"
else
    #cat $URL_LINTER_TEMPDIR/urls_404.out
    while read url_404; do
        echo -e "${RED}$url_404${NC} locations:"
        grep --exclude-dir *url_linter_temp* --exclude-dir vendor -I -r $url_404 $1
    done < $URL_LINTER_TEMPDIR/urls_404.out
fi

echo "--------------------------------------------"
echo -e "${ORANGE}Locations for urls that have permanently moved:${NC}"
if [ ! -f $URL_LINTER_TEMPDIR/urls_301.out ]; then
    echo -e "${GREEN}No urls found that have permanently moved${NC}"
else
    while read url_301; do
        echo -e "${RED}$url_301${NC} locations:"
        grep --exclude-dir *url_linter_temp* --exclude-dir vendor -I -r $url_301 $1
    done < $URL_LINTER_TEMPDIR/urls_301.out
fi

cleanup_url_linter
