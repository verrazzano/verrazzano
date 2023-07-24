// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// source file: application-operator/controllers/metricstrait/workloads.go
package controllers

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func NewTraitDefaultsForWLSDomainWorkload(workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	// Port precedence: trait, workload annotation, default
	port := 7001
	path := "/wls-exporter/metrics"
	secret, err := fetchWLSDomainCredentialsSecretName(workload)
	scraper := "verrazzano-system/vmi-system-prometheus-0"
	if err != nil {
		return nil, err
	}
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:   &path,
		Secret: secret,
		Scraper: &scraper,
	}, nil
}
func fetchWLSDomainCredentialsSecretName(workload *unstructured.Unstructured) (*string, error) {
	secretName, found, err := unstructured.NestedString(workload.Object, "spec", "webLogicCredentialsSecret", "name")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &secretName, nil
}
func NewTraitDefaultsForCOHWorkload(workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	path := "/metrics"
	port := 9612
	var secret *string
	scraper := "verrazzano-system/vmi-system-prometheus-0"
	enabled, p, s, err := fetchCoherenceMetricsSpec(workload)
	if err != nil {
		return nil, err
	}
	if enabled == nil || *enabled {
		if p != nil {
			port = *p
		}
		if s != nil {
			secret = s
		}
	}
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:   &path,
		Secret: secret,
		Scraper: &scraper,
	}, nil
}
func fetchCoherenceMetricsSpec(workload *unstructured.Unstructured) (*bool, *int, *string, error) {
	// determine if metrics is enabled
	enabled, found, err := unstructured.NestedBool(workload.Object, "spec", "coherence", "metrics", "enabled")
	if err != nil {
		return nil, nil, nil, err
	}
	var e *bool
	if found {
		e = &enabled
	}

	// get the metrics port
	port, found, err := unstructured.NestedInt64(workload.Object, "spec", "coherence", "metrics", "port")
	if err != nil {
		return nil, nil, nil, err
	}
	var p *int
	if found {
		p2 := int(port)
		p = &p2
	}

	// get the secret if ssl is enabled
	enabled, found, err = unstructured.NestedBool(workload.Object, "spec", "coherence", "metrics", "ssl", "enabled")
	if err != nil {
		return nil, nil, nil, err
	}
	var s *string
	if found && enabled {
		secret, found, err := unstructured.NestedString(workload.Object, "spec", "coherence", "metrics", "ssl", "secrets")
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			s = &secret
		}
	}
	return e, p, s, nil
}
func  NewTraitDefaultsForGenericWorkload() (*vzapi.MetricsTraitSpec, error) {
	port := 8080
	path := "/metrics"
	scraper := "verrazzano-system/vmi-system-prometheus-0"
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:    &path,
		Secret:  nil,
		Scraper: &scraper,
	} , nil
}
