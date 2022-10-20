// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhookreadiness

import (
	"fmt"
	"go.uber.org/zap"
)

// StartReadinessServer to check webhook readiness
func StartReadinessServer(log *zap.SugaredLogger) {
	log.Info(fmt.Println("put something here"))

}
