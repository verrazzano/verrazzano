# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module "oke" {
  source = "oracle-terraform-modules/oke/oci"
  version = "3.3.0"

  tenancy_id = var.tenancy_id
  user_id = var.user_id
  region = var.region
  api_fingerprint = var.api_fingerprint
  api_private_key_path =var.api_private_key_path

  cluster_name = var.cluster_name
  compartment_id = var.compartment_id
  kubernetes_version = var.kubernetes_version
  allow_worker_ssh_access = var.allow_worker_ssh_access
  worker_mode = var.worker_mode
  cluster_access = var.cluster_access
  ssh_private_key_path = var.ssh_private_key_path
  ssh_public_key_path = var.ssh_public_key_path
  node_pools =var.node_pools
  allow_node_port_access = var.allow_node_port_access
  operator_enabled = var.operator_enabled
  bastion_enabled = var.bastion_enabled
  username = var.username

  calico_enabled = var.calico_enabled
  calico_version = var.calico_version

  vcn_name = "${var.cluster_name}-vcn"
  vcn_dns_label = var.cluster_name
  label_prefix = var.label_prefix

  operator_shape = { shape="VM.Standard.E3.Flex", ocpus=1, memory=4, boot_volume_size=50 }
  operator_notification_endpoint = ""
  operator_instance_principal = false
  operator_notification_enabled = false
  operator_timezone = "UTC"

  bastion_shape = { shape="VM.Standard.E3.Flex", ocpus=1, memory=4, boot_volume_size=50 }
  bastion_timezone = "UTC"
  bastion_notification_enabled = false
  bastion_notification_endpoint = ""

  email_address = ""

  create_service_account = false
  service_account_cluster_role_binding = ""

  existing_key_id = ""

  use_signed_images = false
  image_signing_keys = []

  node_pool_image_id = var.node_pool_image_id
}

output "oke_cluster_id" {
  value = module.oke.cluster_id
}
