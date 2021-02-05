# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

node_pools = {
  "np1" = {
    shape = "VM.Standard.E3.Flex",
    ocpus = 8,
    node_pool_size = 2,
    boot_volume_size = 100
  }
}
