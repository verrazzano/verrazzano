# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

node_pools = {
  "np1" = {
    shape = "VM.Standard.E3.Flex",
    ocpus = 4,
    memory = 80,
    node_pool_size = 4,
    boot_volume_size = 100
  }
}
