// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"fmt"
	"github.com/verrazzano/verrazzano/operator/internal/k8s"
	"github.com/verrazzano/verrazzano/operator/internal/util/helm"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// Needed for unit testing
var deleteJobFunc = k8s.DeleteJob

// The istioPreUpgrade function must delete the Istio Helm install job that may have been left over
// from the previous install.  This must be done or the helm install will fail because the Istio
// Helm post-hook won't be able to create the job
func IstioPreUpgrade(client clipkg.Client, _ string, namespace string, chartDir string) error {
	var log = ctrl.Log.WithName("upgrade")

	chartVersion, err := helm.GetChartVersion(chartDir)
	if err != nil {
		log.Error(err, fmt.Sprintf("Unable to get the chart version from %s", chartDir))
		return err
	}
	jobName := "istio-security-post-install-" + chartVersion.Version
	log.Info(fmt.Sprintf("Deleting Istio Helm post-install job %s", jobName))
	return deleteJobFunc(client, jobName, namespace)
}
