#!/bin/bash
#
# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z $1 ] || [ ! -d $1 ]; then
    echo "No valid directory provided to store suspect urls"
    exit 1
fi

if [[  $1 != *url_linter_temp* ]]; then
    echo "The directory provided is not the same as temporary linter directory"
    echo "Looks like the script was called independent of the linter. This is a helper script for the linter."
    exit 1
fi

if [ -z $2 ]; then
    echo "URL has not been provided"
    exit 1
fi

URL=${2?}

#skipping urls that are supposed to be ignored
grep $URL $(dirname "$0")/ignore_urls.txt > /dev/null
if [ $? -eq 1 ]; then
    response=$(curl -Li --max-redirs 5 --write-out '%{http_code}' --silent --output /dev/null $URL)
    if [ "$response" -eq 404 ]; then
        echo $URL >> $1/urls_404.out
    elif [ "$response" -eq 301 ]; then
        echo $URL >> $1/urls_301.out
    fi
fi