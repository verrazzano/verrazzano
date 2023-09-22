// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c KeycloakComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{
			nginx.ComponentName,
			common.IstioComponentName,
			cmconstants.CertManagerComponentName,
			mysql.ComponentName,
			fluentoperator.ComponentName,
			// ArgoCD and Rancher require Keycloak to be re-reconciled to build the client IDs
			common.ArgoCDName,
			common.RancherName,
		}),
		watch.GetModuleUpdatedWatches([]string{
			nginx.ComponentName,
			cmconstants.CertManagerComponentName,
			mysql.ComponentName,
			fluentoperator.ComponentName,
			// ArgoCD and Rancher require Keycloak to be re-reconciled to build the client IDs
			common.ArgoCDName,
			common.RancherName,
		}),
		watch.GetCreateSecretWatch(vzconst.ThanosInternalUserSecretName, vzconst.VerrazzanoMonitoringNamespace),
		watch.GetUpdateSecretWatch(vzconst.ThanosInternalUserSecretName, vzconst.VerrazzanoMonitoringNamespace),
	)
}
