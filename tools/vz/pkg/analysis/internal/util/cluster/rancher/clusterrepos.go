// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Minimal definition of object that only contains the fields that will be analyzed
type clusterRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []clusterRepo `json:"items"`
}
type clusterRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            cattleStatus `json:"status,omitempty"`
}
