// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"go.uber.org/zap"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

func FixupPrometheusDeployment(log *zap.SugaredLogger, client clipkg.Client) error {
	ctx := context.TODO()
	// If Prometheus isn't deployed don't do anything.
	promKey := clipkg.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus-0"}
	promObj := k8sapps.Deployment{}
	err := client.Get(ctx, promKey, &promObj)
	if errors.IsNotFound(err) {
		log.Debugf("No Prometheus deployment found. Skip update.")
		return nil
	}
	if err != nil {
		log.Errorf("Failed to fetch Prometheus deployment: %s", err)
		return err
	}

	// If Keycloak isn't deployed configure Prometheus to avoid the Istio sidecar for metrics scraping.
	kcKey := clipkg.ObjectKey{Namespace: "keycloak", Name: "keycloak"}
	kcObj := k8sapps.StatefulSet{}
	err = client.Get(ctx, kcKey, &kcObj)
	if errors.IsNotFound(err) {
		// Set the Istio annotation on Prometheus to exclude all IP addresses.
		promObj.Spec.Template.Annotations["traffic.sidecar.istio.io/excludeOutboundIPRanges"] = "0.0.0.0/0"
		err = client.Update(ctx, &promObj)
		if err != nil {
			log.Errorf("Failed to update Istio annotations of Prometheus deployment: %s", err)
			return err
		}
		return nil
	}
	if err != nil {
		log.Errorf("Failed to fetch Keycloak statefulset: %s", err)
		return err
	}

	// Set the Istio annotation on Prometheus to exclude Keycloak HTTP Service IP address.
	// The includeOutboundIPRanges implies all others are excluded.
	svcKey := clipkg.ObjectKey{Namespace: "keycloak", Name: "keycloak-http"}
	svcObj := k8score.Service{}
	err = client.Get(ctx, svcKey, &svcObj)
	if errors.IsNotFound(err) {
		log.Errorf("Failed to find HTTP Service for Keycloak: %s", err)
		return err
	}
	promObj.Spec.Template.Annotations["traffic.sidecar.istio.io/includeOutboundIPRanges"] = fmt.Sprintf("%s/32", svcObj.Spec.ClusterIP)
	err = client.Update(ctx, &promObj)
	if err != nil {
		log.Errorf("Failed to update Istio annotations of Prometheus deployment: %s", err)
		return err
	}
	return nil
}
