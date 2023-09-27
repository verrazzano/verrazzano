// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c DexComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{
			nginx.ComponentName,
			cmconstants.CertManagerComponentName,
			// Dex doesn't have fluentbit stuff yet, but probably will
			fluentoperator.ComponentName,
		}),
		watch.GetModuleUpdatedWatches([]string{
			nginx.ComponentName,
		}),
	)
}
