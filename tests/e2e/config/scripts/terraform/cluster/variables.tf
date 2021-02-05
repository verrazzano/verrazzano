variable "compartment_id" {}
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

variable "cluster_name" {
  default = "oke"
}
variable "label_prefix" {
  default = ""
}
variable "username" {
  default = ""
}
variable "tenancy_name" {}
variable "kubernetes_version" {
  default = "v1.17.13"
}
variable "allow_worker_ssh_access" {
  default = false
}
variable "worker_mode" {
  default = "private"
}
variable "ssh_public_key_path" {
  default = ""
}
variable "ssh_private_key_path" {
  default = ""
}
variable "node_pools" {
  default = {"np1" = {shape="VM.Standard2.4",node_pool_size=4,boot_volume_size=50}}
}
variable "allow_node_port_access" {
  default = false
}
variable "bastion_enabled" {
  default = false
}
variable "operator_enabled" {
  default = false
}
