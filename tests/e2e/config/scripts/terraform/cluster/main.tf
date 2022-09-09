# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module "oke" {
  source = "oracle-terraform-modules/oke/oci"
  version = "4.2.7"

  tenancy_id = var.tenancy_id
  user_id = var.user_id
  home_region = "us-phoenix-1"
  region = var.region
  api_fingerprint = var.api_fingerprint
  api_private_key_path =var.api_private_key_path

  cluster_name = var.cluster_name
  compartment_id = var.compartment_id
  kubernetes_version = var.kubernetes_version
  allow_worker_ssh_access = var.allow_worker_ssh_access
  worker_type = var.worker_mode
  control_plane_type = var.cluster_access
  ssh_private_key_path = var.ssh_private_key_path
  ssh_public_key_path = var.ssh_public_key_path
  node_pools =var.node_pools
  allow_node_port_access = var.allow_node_port_access
  username = var.username

  enable_calico = var.calico_enabled
  calico_version = var.calico_version

  vcn_name = "${var.cluster_name}-vcn"
  vcn_dns_label = var.cluster_name
  label_prefix = var.label_prefix

  create_operator = var.operator_enabled
  operator_shape = { shape="VM.Standard.E3.Flex", ocpus=1, memory=4, boot_volume_size=50 }
  operator_notification_endpoint = ""
  enable_operator_instance_principal = false
  enable_operator_notification = false
  operator_timezone = "UTC"

  create_bastion_host = var.bastion_enabled
  bastion_shape = { shape="VM.Standard.E3.Flex", ocpus=1, memory=4, boot_volume_size=50 }
  bastion_timezone = "UTC"
  enable_bastion_notification = false
  bastion_notification_endpoint = ""

  email_address = ""

  create_service_account = false
  service_account_cluster_role_binding = ""

  use_signed_images = false
  image_signing_keys = []

  node_pool_image_id = var.node_pool_image_id

  freeform_tags = {
    vcn = {
        "verrazzano-infra/Branch" = var.branch_tag
        "verrazzano-infra/Pipeline" = var.pipeline_tag
        "verrazzano-infra/JobScenario" = var.job_scenario_tag
    }
    bastion = {
        "verrazzano-infra/Branch" = var.branch_tag
        "verrazzano-infra/Pipeline" = var.pipeline_tag
        "verrazzano-infra/JobScenario" = var.job_scenario_tag
    }
    operator = {
        "verrazzano-infra/Branch" = var.branch_tag
        "verrazzano-infra/Pipeline" = var.pipeline_tag
        "verrazzano-infra/JobScenario" = var.job_scenario_tag
    }
    oke = {
      cluster = {
        "verrazzano-infra/Branch" = var.branch_tag
        "verrazzano-infra/Pipeline" = var.pipeline_tag
        "verrazzano-infra/JobScenario" = var.job_scenario_tag
      }
      service_lb = {
        "verrazzano-infra/Branch" = var.branch_tag
        "verrazzano-infra/Pipeline" = var.pipeline_tag
        "verrazzano-infra/JobScenario" = var.job_scenario_tag
      }
      persistent_volume = {
        "verrazzano-infra/Branch" = var.branch_tag
        "verrazzano-infra/Pipeline" = var.pipeline_tag
        "verrazzano-infra/JobScenario" = var.job_scenario_tag
      }
      node_pool = {
        "verrazzano-infra/Branch" = var.branch_tag
        "verrazzano-infra/Pipeline" = var.pipeline_tag
        "verrazzano-infra/JobScenario" = var.job_scenario_tag
      }
    }
  }

  providers = {
    oci.home = oci.home
  }
}

output "oke_cluster_id" {
  value = module.oke.cluster_id
}
