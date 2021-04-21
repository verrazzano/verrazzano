#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo "Install Calico..."

curl -sS https://docs.projectcalico.org/v3.18/manifests/calico-policy-only.yaml -o /tmp/calico.yaml

# uncomment the pod CIDR setting and replace the default pod CIDR with the OKE pod CIDR default
sed -i -e "s?# - name: CALICO_IPV4POOL_CIDR?- name: CALICO_IPV4POOL_CIDR?" /tmp/calico.yaml
sed -i -e "s?#   value: \"192.168.0.0/16\"?  value: \"10.244.0.0/16\"?" /tmp/calico.yaml
grep -B 1 -A 2 POOL_CIDR /tmp/calico.yaml

kubectl apply -f /tmp/calico.yaml
