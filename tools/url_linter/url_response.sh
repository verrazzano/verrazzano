#!/bin/bash
#
# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

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