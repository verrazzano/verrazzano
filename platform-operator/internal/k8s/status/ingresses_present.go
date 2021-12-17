// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package status

import (
	"context"

	"go.uber.org/zap"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// IngressesPresent Check that the named ingresses are present in the cluster
func IngressesPresent(log *zap.SugaredLogger, client clipkg.Client, ingressNames []types.NamespacedName) bool {
	for _, ingName := range ingressNames {
		ing := v1.Ingress{}
		if err := client.Get(context.TODO(), ingName, &ing); err != nil {
			if errors.IsNotFound(err) {
				log.Debugf("%v ingress not found", ingName)
				// Ingress not found
				return false
			}
			log.Errorf("Unexpected error checking for ingress %v: %v", ingName, err)
			return false
		}
	}
	return true
}
