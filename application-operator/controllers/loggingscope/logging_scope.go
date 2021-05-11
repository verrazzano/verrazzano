// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"

	"github.com/verrazzano/verrazzano/application-operator/constants"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
)

const (
	configMapAPIVersion = "v1"
	configMapKind       = "ConfigMap"
)

// LoggingScope contains information needed for logging
type LoggingScope struct {
	// The fluentd image
	FluentdImage string

	// URL for Elasticsearch
	ElasticSearchURL string

	// Name of secret with Elasticsearch credentials
	SecretName string

	// Namespace for the logging secret
	SecretNamespace string
}

// Handler abstracts the FLUENTD integration for components
type Handler interface {
	Apply(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *LoggingScope) (*ctrl.Result, error)
	Remove(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *LoggingScope) (bool, error)
}

// NewLoggingScope creates and populates a new logging scope
func NewLoggingScope(ctx context.Context, cli client.Reader, fluentdImageOrverride string, esDetails clusters.ElasticsearchDetails) (*LoggingScope, error) {
	scope := LoggingScope{
		SecretNamespace: constants.VerrazzanoSystemNamespace,
	}

	if esDetails.URL != "" && esDetails.SecretName != "" {
		scope.ElasticSearchURL = esDetails.URL
		scope.SecretName = esDetails.SecretName
	}

	if len(fluentdImageOrverride) != 0 {
		scope.FluentdImage = fluentdImageOrverride
	} else {
		scope.FluentdImage = DefaultFluentdImage
	}
	if scope.ElasticSearchURL == "" {
		scope.ElasticSearchURL = DefaultElasticSearchURL
	}
	if scope.SecretName == "" {
		scope.SecretName = DefaultSecretName
	}
	return &scope, nil
}
