// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c weblogicComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleReadyWatches([]string{istio.ComponentName, fluentoperator.ComponentName})
}
