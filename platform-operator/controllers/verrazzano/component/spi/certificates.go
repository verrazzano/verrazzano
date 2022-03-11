// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"context"
	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
)

func CheckCertificatesReady(ctx ComponentContext, certificates []types.NamespacedName) (notReady []types.NamespacedName) {
	if len(certificates) == 0 {
		return []types.NamespacedName{}
	}

	log := ctx.Log()
	if !vzconfig.IsCertManagerEnabled(ctx.EffectiveCR()) {
		log.Oncef("Cert-Manager disabled, skipping certificates check")
		return []types.NamespacedName{}
	}

	ctx.Log().Oncef("Checking certificates status for %v", certificates)
	client := ctx.Client()
	for _, name := range certificates {
		ready, err := CertficateIsReady(client, name)
		if err != nil {
			log.Errorf("Error getting certificate %s: %s", name, err)
		}
		if !ready {
			notReady = append(notReady, name)
		}
	}
	return notReady
}

func CertficateIsReady(client clipkg.Client, name types.NamespacedName) (bool, error) {
	cert := &certapiv1.Certificate{}
	if err := client.Get(context.TODO(), name, cert); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	certConditions := cert.Status.Conditions
	if len(certConditions) > 0 {
		sort.Slice(certConditions, func(i, j int) bool {
			return certConditions[i].LastTransitionTime.After(
				certConditions[j].LastTransitionTime.Time)
		})
		mostRecent := certConditions[0]
		if mostRecent.Status == cmmeta.ConditionTrue && mostRecent.Type == certapiv1.CertificateConditionReady {
			return true, nil
		}
	}
	return false, nil
}
