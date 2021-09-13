// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"os"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
)

// appendApplicationOperatorOverrides Honor the APP_OPERATOR_IMAGE env var if set; this allows an explicit override
// of the verrazzano-application-operator image when set.
func appendApplicationOperatorOverrides(_ *zap.SugaredLogger, _ string, _ string, _ string, kvs []keyValue) ([]keyValue, error) {
	envImageOverride := os.Getenv(constants.VerrazzanoAppOperatorImageEnvVar)
	if len(envImageOverride) == 0 {
		return kvs, nil
	}
	kvs = append(kvs, keyValue{
		key:   "image",
		value: envImageOverride,
	})
	return kvs, nil
}
