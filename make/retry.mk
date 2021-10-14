# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

DEFAULT_RETRIES:=5

#
# Retry a command a parameterized amount of times.
#
define retry_cmd
    for i in `seq 1 $1`; do $2 && break; done
endef

define retry_docker_push
    $(call retry_cmd,${DEFAULT_RETRIES},docker push $1)
endef