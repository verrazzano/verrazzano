// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"path"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// CreateOrDeleteFluentbitFilterAndParser create or delete the Fluentbit Filter & Parser Resource by applying/deleting the fluentbitFilterAndParserTemplate based on the delete flag.
func CreateOrDeleteFluentbitFilterAndParser(ctx spi.ComponentContext, fluentbitFilterAndParserTemplate, namespace string, delete bool) error {
	args := make(map[string]interface{})
	args["namespace"] = namespace
	templatePath := path.Join(config.GetThirdPartyManifestsDir(), "fluent-operator/"+fluentbitFilterAndParserTemplate)
	if delete {
		if err := k8sutil.NewYAMLApplier(ctx.Client(), "").DeleteFT(templatePath, args); err != nil && !meta.IsNoMatchError(err) {
			return ctx.Log().ErrorfNewErr("Failed Deleting Filter and Parser for Fluent Operator: %v", err)
		}
		return nil
	}
	if vzcr.IsFluentOperatorEnabled(ctx.EffectiveCR()) {
		if err := k8sutil.NewYAMLApplier(ctx.Client(), "").ApplyFT(templatePath, args); err != nil {
			return ctx.Log().ErrorfNewErr("Failed applying Filter and Parser for Fluent Operator: %v", err)
		}
	}
	return nil
}
