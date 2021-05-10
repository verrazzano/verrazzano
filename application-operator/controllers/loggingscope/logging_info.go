// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
)

const (
	configMapAPIVersion = "v1"
	configMapKind       = "ConfigMap"
)

// LoggingInfoSpec defines the desired state of LoggingInfo
type LoggingInfoSpec struct {
	// The fluentd image
	FluentdImage string

	// URL for Elasticsearch
	ElasticSearchURL string

	// Name of secret with Elasticsearch credentials
	SecretName string
}

// LoggingInfo is the Schema for the loggingscopes API
type LoggingInfo struct {
	metav1.ObjectMeta

	Spec LoggingInfoSpec
}

// Handler abstracts the FLUENTD integration for components
type Handler interface {
	Apply(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *LoggingInfo) (*ctrl.Result, error)
	Remove(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *LoggingInfo) (bool, error)
}

// NewLoggingScope creates and populates a new logging scope
func NewLoggingScope(ctx context.Context, cli client.Reader, fluentdImageOrverride string) (*LoggingInfo, error) {
	scope := LoggingInfo{}

	// if we're running in a managed cluster, use the multicluster ES URL and secret, and if we're
	// not the fields will be empty and we will set these fields to defaults below
	elasticSearchDetails := clusters.FetchManagedClusterElasticSearchDetails(ctx, cli)
	if elasticSearchDetails.URL != "" && elasticSearchDetails.SecretName != "" {
		scope.Spec.ElasticSearchURL = elasticSearchDetails.URL
		scope.Spec.SecretName = elasticSearchDetails.SecretName
	}

	if len(fluentdImageOrverride) != 0 {
		scope.Spec.FluentdImage = fluentdImageOrverride
	} else {
		scope.Spec.FluentdImage = DefaultFluentdImage
	}
	if scope.Spec.ElasticSearchURL == "" {
		scope.Spec.ElasticSearchURL = DefaultElasticSearchURL
	}
	if scope.Spec.SecretName == "" {
		scope.Spec.SecretName = DefaultSecretName
	}
	return &scope, nil
}
