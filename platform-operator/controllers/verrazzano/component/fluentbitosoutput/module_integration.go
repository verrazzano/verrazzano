// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentbitosoutput

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c fluentbitOpensearchOutput) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return append(
		watch.GetModuleInstalledWatches([]string{fluentoperator.ComponentName}),
		watch.GetSecretWatch(vzconst.VerrazzanoSystemNamespace, vzconst.MCRegistrationSecret)...,
	)
}
