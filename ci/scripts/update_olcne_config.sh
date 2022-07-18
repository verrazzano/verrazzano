ssh -o StrictHostKeyChecking=no -i "$TF_VAR_ssh_private_key_path" opc@$API_SERVER_IP "
    # Enable external IP
    olcnectl module update \
    --environment-name $TF_VAR_environment_name \
    --name $TF_VAR_kubernetes_name \
    --restrict-service-externalip=false \
    --force

    olcnectl module validate \
    --environment-name $TF_VAR_environment_name \
    --name $TF_VAR_kubernetes_name

    # Install helm module
    olcnectl module create \
    --environment-name $TF_VAR_environment_name \
    --module helm \
    --name $TF_VAR_helm_name \
    --helm-kubernetes-module $TF_VAR_kubernetes_name 

    olcnectl module validate \
    --environment-name $TF_VAR_environment_name \
    --name $TF_VAR_helm_name

    olcnectl module install \
    --environment-name $TF_VAR_environment_name \
    --name $TF_VAR_helm_name 

    # Install oci-ccm module
    # OCI private key file should exist on the master node
    olcnectl module create \
    --environment-name $TF_VAR_environment_name \
    --module oci-ccm \
    --name $TF_VAR_oci_ccm_name \
    --oci-ccm-helm-module $TF_VAR_helm_name \
    --oci-region $TF_VAR_region \
    --oci-tenancy $TF_VAR_tenancy_id \
    --oci-compartment $TF_VAR_compartment_id \
    --oci-user $TF_VAR_user_id \
    --oci-fingerprint $TF_VAR_fingerprint \
    --oci-private-key /home/opc/oci_api_key.pem \
    --oci-vcn $VCN_OCID \
    --oci-lb-subnet1 $BASTION_SUBNET_OCID \
    --oci-lb-security-mode None

    olcnectl module validate \
    --environment-name $TF_VAR_environment_name \
    --name $TF_VAR_oci_ccm_name

    olcnectl module install \
    --environment-name $TF_VAR_environment_name \
    --name $TF_VAR_oci_ccm_name
"