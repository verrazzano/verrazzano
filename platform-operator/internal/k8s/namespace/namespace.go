// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package namespace

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//CreateAndLabelNamespace - Utility function to create a namespace and optionally add either the VZ managed and/or Istio injection labels
func CreateAndLabelNamespace(client client.Client, ns string, isVerrazzanoManaged bool, withIstioInjection bool) error {
	nsObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), client, nsObj,
		func() error {
			nsObj.Labels, _ = MergeMaps(nsObj.Labels, createLabelsMap(ns, isVerrazzanoManaged, withIstioInjection))
			return nil
		},
	)
	return err
}

// CreateCertManagerNamespace - Create/Update and label the cert-manager namespace
func CreateCertManagerNamespace(client client.Client, istioInjectionEnabled bool) error {
	return CreateAndLabelNamespace(client, globalconst.CertManagerNamespace, true, istioInjectionEnabled)
}

// CreateIngressNginxNamespace - Create/Update and label the ingres-nginx namespace
func CreateIngressNginxNamespace(client client.Client, istioInjectionEnabled bool) error {
	return CreateAndLabelNamespace(client, globalconst.IngressNamespace, true, istioInjectionEnabled)
}

//CreateIstioNamespace - Create/Update and label the Istio namespace
func CreateIstioNamespace(client client.Client) error {
	return CreateAndLabelNamespace(client, globalconst.IstioSystemNamespace, true, false)
}

// CreateKeycloakNamespace - Create/Update and label the Keycloak namespace
func CreateKeycloakNamespace(client client.Client, istioInjectionEnabled bool) error {
	return CreateAndLabelNamespace(client, globalconst.KeycloakNamespace, true, istioInjectionEnabled)
}

// CreateMysqlOperator - Create/Update and label the MySQL operator namespace
func CreateMysqlOperator(client client.Client, istioInjectionEnabled bool) error {
	return CreateAndLabelNamespace(client, globalconst.MySQLOperatorNamespace, true, istioInjectionEnabled)
}

// CreateRancherNamespace - Create/Update and label the Rancher system namespace
func CreateRancherNamespace(client client.Client) error {
	return CreateAndLabelNamespace(client, globalconst.RancherSystemNamespace, true, false)
}

// CreateVerrazzanoMonitoringNamespace - Create/Update and label the Verrazzano monitoring namespace
func CreateVerrazzanoMonitoringNamespace(client client.Client, istioInjectionEnabled bool) error {
	return CreateAndLabelNamespace(client, constants.VerrazzanoMonitoringNamespace, true, istioInjectionEnabled)
}

// CreateVerrazzanoSystemNamespace - Create/Update and label the Verrazzano system namespace
func CreateVerrazzanoSystemNamespace(client client.Client, istioInjectionEnabled bool) error {
	return CreateAndLabelNamespace(client, globalconst.VerrazzanoSystemNamespace, true, istioInjectionEnabled)
}

//CreateVerrazzanoMultiClusterNamespace - Create/Update and label the Verrazzano multi-cluster namespace
func CreateVerrazzanoMultiClusterNamespace(client client.Client) error {
	return CreateAndLabelNamespace(client, globalconst.VerrazzanoMultiClusterNamespace, false, false)
}

// MergeMaps Merge one map into another, creating new one if necessary; returns the updated map and true if it was modified
func MergeMaps(to map[string]string, from map[string]string) (map[string]string, bool) {
	mergedMap := to
	if mergedMap == nil {
		mergedMap = make(map[string]string)
	}
	var updated bool
	for k, v := range from {
		mergedMap[k] = v
	}
	return mergedMap, updated
}

//createLabelsMap - Create a map with the the Verrazzano-managed and/or Istio injection labels
func createLabelsMap(ns string, isVerrazzanoManaged bool, withIstioInjection bool) map[string]string {
	annotations := map[string]string{}
	if isVerrazzanoManaged {
		annotations[globalconst.LabelVerrazzanoNamespace] = ns
	}
	if withIstioInjection {
		annotations[globalconst.LabelIstioInjection] = "enabled"
	}
	return annotations
}
