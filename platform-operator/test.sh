#!/bin/bash

VAR1=${VZ_DEV_IMAGE}
VAR2="iad.ocir.io/odsbuilddev/aamitra/dev/verrazzano-platform-operator-dev:local-b67c5587"

if [ "$VAR1" = "$VAR2" ]; then
    echo "Strings are equal."
else
    echo "Strings are not equal."
fi

