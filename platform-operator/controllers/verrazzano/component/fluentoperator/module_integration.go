// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentoperator

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c fluentOperatorComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetCreateSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
		watch.GetUpdateSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
		watch.GetDeleteSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
	)
}
