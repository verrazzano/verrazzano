// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
)

// GetWatchDescriptors Returns a watch on the whole VZ CR spec, as the VZ "module" looks at a lot of settings; for now
// just watch for any changes and trigger this to reconcile
func (c verrazzanoComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	wd := watch.GetModuleInstalledWatches([]string{common.IstioComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName, authproxy.ComponentName, fluentoperator.ComponentName})
	return append(wd, watch.GetVerrazzanoSpecWatch()...)
}
