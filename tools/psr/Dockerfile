# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM scratch

COPY out/linux_amd64/psr-backend  /usr/local/bin/psr-backend
WORKDIR /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/psr-backend"]
