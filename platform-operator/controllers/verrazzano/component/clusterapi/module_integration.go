// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c clusterAPIComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleInstalledWatches([]string{cmconstants.CertManagerComponentName, cmconstants.ClusterIssuerComponentName})
}
