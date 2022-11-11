// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ready

import (
	"context"
	"sort"

	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// CertificatesAreReady Checks the list of named objects to see if there are matching
// Cert-Manager Certificate objects, and checks if those are in a Ready state.
//
// ctx				A valid ComponentContext for the operation
// certificates		A list of NamespacedNames; this should be names of expected Certificate objects
//
// Returns true and an empty list of names if all certs are ready, false and a list of certificate names that are
// NOT in the ready state
func CertificatesAreReady(client clipkg.Client, log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, certificates []types.NamespacedName) (ready bool, certsNotReady []types.NamespacedName) {
	if len(certificates) == 0 {
		return true, []types.NamespacedName{}
	}

	if vz != nil && vz.Spec.Components.CertManager != nil && vz.Spec.Components.CertManager.Enabled != nil {
		if !*vz.Spec.Components.CertManager.Enabled {
			log.Oncef("Cert-Manager disabled, skipping certificates check")
			return true, []types.NamespacedName{}
		}
	}

	log.Oncef("Checking certificates status for %v", certificates)
	for _, name := range certificates {
		ready, err := IsCertficateIsReady(log, client, name)
		if err != nil {
			log.Errorf("Error getting certificate %s: %s", name, err)
		}
		if !ready {
			certsNotReady = append(certsNotReady, name)
		}
	}
	return len(certsNotReady) == 0, certsNotReady
}

// IsCertficateIsReady Checks if a Cert-Manager Certificate object with the specified NamespacedName
// can be found in the cluster, and if it is in a Ready state.
//
// Returns
// - true/nil if a matching Certificate object is found and Ready
// - false/nil if a matching Certifiate object is found and not Ready
// - false/error if an unexpected error has occurred
func IsCertficateIsReady(log vzlog.VerrazzanoLogger, client clipkg.Client, name types.NamespacedName) (bool, error) {
	cert := &certapiv1.Certificate{}
	if err := client.Get(context.TODO(), name, cert); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	certConditions := cert.Status.Conditions
	if len(certConditions) > 0 {
		if len(certConditions) > 1 {
			// Typically, I've only seen one condition in the Certificate object, but if there's
			// more than one sort the copy so the most recent is first
			sort.Slice(certConditions, func(i, j int) bool {
				return certConditions[i].LastTransitionTime.After(
					certConditions[j].LastTransitionTime.Time)
			})
		}
		mostRecent := certConditions[0]
		if mostRecent.Status == cmmeta.ConditionTrue && mostRecent.Type == certapiv1.CertificateConditionReady {
			return true, nil
		}
		log.Infof("Certificate %s not ready, reason: %s, message: %s", name, mostRecent.Reason, mostRecent.Message)
	}
	return false, nil
}
