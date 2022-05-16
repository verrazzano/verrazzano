// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

// deleteClusterRepos - temporary work around for Rancher issue 36914. On upgrade or Rancher
// the setting of useBundledSystemChart does not appear to be honored, and the downloaded
// helm charts for the previous release of Rancher are used (instead of the charts on the Rancher
// container image).
func deleteClusterRepos(log vzlog.VerrazzanoLogger) error {

	config, err := ctrl.GetConfig()
	if err != nil {
		log.Debugf("Rancher Pre-Upgrade: Failed getting config: %v", err)
		return err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Debugf("Rancher Pre-Upgrade: Failed creating dynamic client: %v", err)
		return err
	}

	// Configure the GVR
	gvr := schema.GroupVersionResource{
		Group:    "catalog.cattle.io",
		Version:  "v1",
		Resource: "clusterrepos",
	}

	// List of clusterrepos to delete
	names := []string{"rancher-charts", "rancher-rke2-charts", "rancher-partner-charts"}
	for _, name := range names {
		err = dynamicClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Debugf("Rancher Pre-Upgrade: Failed deleting clusterrepos.catalog.cattle.io %s: %v", name, err)
			return err
		}
		log.Infof("Rancher Pre-Upgrade: Deleted clusterrepos.catalog.cattle.io %s", name)
	}

	// Reconfigure the GVR
	gvr = schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "settings",
	}

	// Delete settings.management.cattle.io chart-default-branch
	name := "chart-default-branch"
	err = dynamicClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Debugf("Rancher Pre-Upgrade: Failed deleting settings.management.cattle.io %s: %v", name, err)
		return err
	}
	log.Infof("Rancher Pre-Upgrade: Deleted settings.management.cattle.io %s", name)

	return nil
}
