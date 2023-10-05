// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (o opensearchOperatorComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{
			nginx.ComponentName,
			cmconstants.ClusterIssuerComponentName,
			vmo.ComponentName,
			mysql.ComponentName,
		}),
		watch.GetModuleUpdatedWatches([]string{
			nginx.ComponentName,
			cmconstants.ClusterIssuerComponentName,
			vmo.ComponentName,
			mysql.ComponentName,
		}),
		watch.GetUpdateSecretWatch(common.SecuritySecretName, constants.VerrazzanoLoggingNamespace),
		watch.GetDeleteSecretWatch(common.SecuritySecretName, constants.VerrazzanoLoggingNamespace),
	)
}
