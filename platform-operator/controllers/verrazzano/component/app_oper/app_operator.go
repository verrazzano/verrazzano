// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package app_oper

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"os"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
)

// AppendApplicationOperatorOverrides Honor the APP_OPERATOR_IMAGE env var if set; this allows an explicit override
// of the verrazzano-application-operator image when set.
func AppendApplicationOperatorOverrides(_ *zap.SugaredLogger, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	envImageOverride := os.Getenv(constants.VerrazzanoAppOperatorImageEnvVar)
	if len(envImageOverride) == 0 {
		return kvs, nil
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   "image",
		Value: envImageOverride,
	})
	fmt.Println("Foo")
	return kvs, nil
}
