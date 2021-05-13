// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logging

import (
	"context"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	configMapAPIVersion = "v1"
	configMapKind       = "ConfigMap"
)

// LogInfo contains information needed for logging
type LogInfo struct {
	// The fluentd image
	FluentdImage string
}

// Handler abstracts the FLUENTD integration for components
type Handler interface {
	Apply(ctx context.Context, resource vzapi.QualifiedResourceRelation, info *LogInfo) (*ctrl.Result, error)
	Remove(ctx context.Context, resource vzapi.QualifiedResourceRelation, info *LogInfo) (bool, error)
}

// NewLogInfo creates and populates a new logging info
func NewLogInfo(fluentdImageOrverride string) (*LogInfo, error) {
	info := LogInfo{}
	if len(fluentdImageOrverride) != 0 {
		info.FluentdImage = fluentdImageOrverride
	} else {
		info.FluentdImage = DefaultFluentdImage
	}
	return &info, nil
}
