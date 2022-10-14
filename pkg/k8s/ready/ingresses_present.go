// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ready

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// IngressesPresent Check that the named ingresses are present in the cluster
func IngressesPresent(log vzlog.VerrazzanoLogger, client clipkg.Client, ingressNames []types.NamespacedName, prefix string) bool {
	for _, ingName := range ingressNames {
		ing := v1.Ingress{}
		if err := client.Get(context.TODO(), ingName, &ing); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for ingress %v to exist", prefix, ingressNames)
				// Ingress not found
				return false
			}
			log.Errorf("Failed getting ingress %v: %v", ingressNames, err)
			return false
		}
	}
	log.Oncef("%s has all the required ingresses %v", prefix, ingressNames)
	return true
}
