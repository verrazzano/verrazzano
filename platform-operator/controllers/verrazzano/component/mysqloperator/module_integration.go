// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c mysqlOperatorComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{istio.ComponentName, fluentoperator.ComponentName}),
		GetMySQLCreateJobWatch(),
	)
}

// GetMySQLCreateJobWatch watches for a job creation from mysql operator
func GetMySQLCreateJobWatch() []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: source.Kind{Type: &batchv1.Job{}},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Created {
				return false
			}
			return IsMysqlOperatorJob(cli, *wev.NewWatchedObject.(*batchv1.Job), vzlog.DefaultLogger())
		},
	})
	return watches
}
