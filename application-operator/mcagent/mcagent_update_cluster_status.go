// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"

	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

func (s *Syncer) UpdateVerrazzanoManagedClusterStatus(name string) error {
	ingress := extv1beta1.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-console-ingress", Namespace: constants.VerrazzanoSystemNamespace}, &ingress)
	if err != nil {
		return fmt.Errorf("Unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, "verrazzano-console-ingress", err)
	}

	fetched := clustersv1alpha1.VerrazzanoManagedCluster{}
	err = s.AdminClient.Get(s.Context, types.NamespacedName{Name: name, Namespace: "verrazzano-mc"}, &fetched)
	if err != nil {
		return fmt.Errorf("Unable to fetch vmc %s/%s, %v", "verrazzano-mc", name, err)
	}
	fetched.Status.APIUrl = fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	return s.AdminClient.Status().Update(s.Context, &fetched)
}
