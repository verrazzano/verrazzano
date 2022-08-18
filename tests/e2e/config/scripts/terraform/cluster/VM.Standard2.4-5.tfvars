# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

node_pools = {
  "np1" = {
    shape = "VM.Standard2.4",
    node_pool_size = 5,
    boot_volume_size = 50
  }
}
