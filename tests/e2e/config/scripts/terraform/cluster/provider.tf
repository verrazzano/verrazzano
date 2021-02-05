variable "region" {}
variable "tenancy_id" {}
variable "user_id" {}
variable "api_fingerprint" {}
variable "api_private_key_path" {}

provider "oci" {
  version              = ">= 3.0.0"

  tenancy_ocid         = var.tenancy_id
  user_ocid            = var.user_id
  fingerprint          = var.api_fingerprint
  private_key_path     = var.api_private_key_path
  region               = var.region
}

terraform {
  required_version = ">= 0.12"
}
