# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

variable "compartment_id" {}

variable "cluster_name" {
  default = "oke"
}
variable "label_prefix" {
  default = ""
}
variable "username" {
  default = ""
}
variable "tenancy_name" {
  default = "stevengreenberginc"
}

variable "operating_system_version" {
  default     = "8"
}

variable "kubernetes_version" {
  default = "v1.24.1"
}
variable "allow_worker_ssh_access" {
  default = false
}
variable "worker_mode" {
  default = "private"
}
variable "cluster_access" {
  default = "public"
}
variable "ssh_public_key_path" {
  default = ""
}
variable "ssh_private_key_path" {
  default = "none"
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
variable "node_pool_image_id" {}
variable "calico_enabled" {}
variable "calico_version" {}
variable "branch_tag" {
  default = "unknown-branch"
}
variable "pipeline_tag" {
  default = "unknown-pipeline"
}
variable "job_scenario_tag" {
  default = "unknown-scenario"
}
