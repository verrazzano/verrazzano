// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c vmoComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{
			nginx.ComponentName,
		}),
		watch.GetModuleUpdatedWatches([]string{
			nginx.ComponentName,
		}),
	)
}
