// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"strings"

	promoperapi "github.com/prometheus-wls/prometheus-wls/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const keycloakHTTPService = "keycloak-http"

// UpdatePrometheusAnnotations updates annotations on the Prometheus CR to include the outbound IP for Keycloak
func UpdatePrometheusAnnotations(ctx spi.ComponentContext, prometheusNamespace string, promOperComponentName string) error {
	// Get a list of Prometheus in the verrazzano-monitoring namespace
	promList := promoperapi.PrometheusList{}
	err := ctx.Client().List(context.TODO(), &promList, &client.ListOptions{
		Namespace:     prometheusNamespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{constants.VerrazzanoComponentLabelKey: promOperComponentName}),
	})
	if err != nil {
		if strings.Contains(err.Error(), "no matches for kind") || strings.Contains(err.Error(), "no kind is registered") {
			ctx.Log().Info("Prometheus CRD not installed, skip updating annotations for Keycloak on the Prometheus instance")
			return nil
		}
		return ctx.Log().ErrorfNewErr("Failed to list Prometheus in the %s namespace: %v", prometheusNamespace, err)
	}

	// Get the Keycloak service to retrieve the cluster IP for the Prometheus annotation
	svc := corev1.Service{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Name: keycloakHTTPService, Namespace: constants.KeycloakNamespace}, &svc)
	if err != nil {
		if errors.IsNotFound(err) {
			ctx.Log().Info("keycloak-http service not found, skip updating annotations for Keycloak on the Prometheus instance")
			return nil
		}
		return ctx.Log().ErrorfNewErr("Failed to get keycloak-http service: %v", err)
	}

	// If the ClusterIP is not empty, update the Prometheus annotation
	// The includeOutboundIPRanges implies all others are excluded.
	// This is done by adding the traffic.sidecar.istio.io/includeOutboundIPRanges=<Keycloak IP>/32 annotation.
	if svc.Spec.ClusterIP != "" {
		for _, prom := range promList.Items {
			_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), prom, func() error {
				if prom.Spec.PodMetadata == nil {
					prom.Spec.PodMetadata = &promoperapi.EmbeddedObjectMetadata{}
				}
				if prom.Spec.PodMetadata.Annotations == nil {
					prom.Spec.PodMetadata.Annotations = make(map[string]string)
				}
				delete(prom.Spec.PodMetadata.Annotations, "traffic.sidecar.istio.io/excludeOutboundIPRanges")
				prom.Spec.PodMetadata.Annotations["traffic.sidecar.istio.io/includeOutboundIPRanges"] = fmt.Sprintf("%s/32", svc.Spec.ClusterIP)
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
