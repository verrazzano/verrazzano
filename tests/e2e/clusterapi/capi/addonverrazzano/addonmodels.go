// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package addonverrazzano

import "time"

type Verrazzano struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			KubectlKubernetesIoLastAppliedConfiguration string `json:"kubectl.kubernetes.io/last-applied-configuration"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		ResourceVersion   string    `json:"resourceVersion"`
		UID               string    `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		Components struct {
		} `json:"components"`
		EnvironmentName string `json:"environmentName"`
		Profile         string `json:"profile"`
		Security        struct {
		} `json:"security"`
	} `json:"spec"`
	Status struct {
		Available  string `json:"available"`
		Conditions []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Message            string    `json:"message"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		Instance struct {
			ConsoleURL              string `json:"consoleUrl"`
			GrafanaURL              string `json:"grafanaUrl"`
			KeyCloakURL             string `json:"keyCloakUrl"`
			KialiURL                string `json:"kialiUrl"`
			OpenSearchDashboardsURL string `json:"openSearchDashboardsUrl"`
			OpenSearchURL           string `json:"openSearchUrl"`
			PrometheusURL           string `json:"prometheusUrl"`
			RancherURL              string `json:"rancherUrl"`
		} `json:"instance"`
		State   string `json:"state"`
		Version string `json:"version"`
	} `json:"status"`
}

type VerrazzanoFleetBinding struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		OwnerReferences   []struct {
			APIVersion         string `json:"apiVersion"`
			BlockOwnerDeletion bool   `json:"blockOwnerDeletion"`
			Controller         bool   `json:"controller"`
			Kind               string `json:"kind"`
			Name               string `json:"name"`
			UID                string `json:"uid"`
		} `json:"ownerReferences"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		ClusterRef struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			Namespace  string `json:"namespace"`
		} `json:"clusterRef"`
		Verrazzano struct {
			Spec struct {
				Components struct {
				} `json:"components"`
				EnvironmentName string `json:"environmentName"`
				Profile         string `json:"profile"`
				Security        struct {
				} `json:"security"`
				Version string `json:"version"`
			} `json:"spec"`
		} `json:"verrazzano"`
	} `json:"spec"`
	Status struct {
		Conditions []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		Revision   int    `json:"revision"`
		Status     string `json:"status"`
		Verrazzano struct {
			ComponentsAvailable string `json:"componentsAvailable"`
			State               string `json:"state"`
			Version             string `json:"version"`
		} `json:"verrazzano"`
	} `json:"status"`
}
