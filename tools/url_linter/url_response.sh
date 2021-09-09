#!/bin/bash

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