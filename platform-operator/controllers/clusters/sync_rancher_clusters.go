// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"bytes"
	"context"
	"time"

	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	createdByLabel      = "app.kubernetes.io/created-by"
	createdByVerrazzano = "verrazzano"
	localClusterName    = "local"
)

// clustersResponseHash is the hash of the Rancher clusters API response body, used to determine if processing can be
// skipped when there are no cluster changes
var clustersResponseHash []byte

type RancherClusterSyncer struct {
	client.Client
}

// StartSyncing starts the Rancher cluster synchronization loop
func (r *RancherClusterSyncer) StartSyncing() {
	log := r.initLogger()
	log.Info("Starting Rancher cluster synchronizing loop")

	// prevent a panic from taking down the operator
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Panic caught: %v", err)
		}
	}()

	for {
		r.syncRancherClusters(log)
		time.Sleep(time.Minute)
	}
}

// initLogger initializes the Verrazzano logger
func (r *RancherClusterSyncer) initLogger() vzlog.VerrazzanoLogger {
	zaplog, err := log.BuildZapLogger(2)
	if err != nil {
		// failed so fall back to the basic zap sugared logger
		zaplog = zap.S()
	}
	return vzlog.EnsureContext("rancher_cluster_sync").EnsureLogger("syncer", zaplog, zaplog)
}

// syncRancherClusters gets the list of clusters from Rancher and creates and deletes VMC resources
func (r *RancherClusterSyncer) syncRancherClusters(log vzlog.VerrazzanoLogger) {
	log.Info("Synchronizing Rancher clusters and VMCs")

	// first check to see if the Rancher admin secret exists, if not then either Rancher is not installed
	// or this is not an admin cluster, so just log a debug message and there is nothing else to do
	if _, err := getAdminSecret(r.Client); err != nil {
		log.Debug("Unable to get Rancher admin secret, either Rancher is not installed or this is not an admin cluster, skipping Rancher cluster sync")
		return
	}

	// call Rancher to get the list of clusters
	rc, err := newRancherConfig(r.Client, log)
	if err != nil {
		log.Errorf("Error connecting to Rancher admin server: %v", err)
		return
	}

	clusters, newClustersResponseHash, err := getAllClustersInRancher(rc, log)
	if err != nil {
		log.Errorf("Error getting cluster list from Rancher: %v", err)
		return
	}

	// if the Rancher response did not change, there is nothing to do
	if bytes.Equal(clustersResponseHash, newClustersResponseHash) {
		log.Debug("Rancher clusters response did not change, nothing to sync")
		return
	}

	// for every cluster (ignoring "local") make sure a VMC exists
	ensureErr := r.ensureVMCs(clusters, log)

	// for any auto-created VMC objects that do not have a Rancher cluster, delete the VMC resource
	deleteErr := r.deleteVMCs(clusters, log)

	// only update the hash if there were no errors, so that we retry on the next run of this function
	if ensureErr == nil && deleteErr == nil {
		clustersResponseHash = newClustersResponseHash
	}
}

// ensureVMCs ensures that there is a VMC resource for every cluster in Rancher, creating VMCs as necessary
func (r *RancherClusterSyncer) ensureVMCs(rancherClusters []rancherCluster, log vzlog.VerrazzanoLogger) error {
	for _, rancherCluster := range rancherClusters {
		// ignore the "local" cluster
		if rancherCluster.name == localClusterName {
			continue
		}

		cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: rancherCluster.name, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr); err != nil {
			if errors.IsNotFound(err) {
				log.Infof("Creating empty VMC for discovered Rancher cluster with name: %s", rancherCluster.name)
				vmc := newVMC(rancherCluster)
				if err := r.Create(context.TODO(), vmc); err != nil {
					log.Infof("Unable to create VMC with name %s: %v", rancherCluster.name, err)
					return err
				}
				continue
			}
			log.Infof("Unable to get VMC with name %s: %v", rancherCluster.name, err)
			return err
		}
	}

	return nil
}

// newVMC returns a minimally populated VMC object
func newVMC(rancherCluster rancherCluster) *clustersv1alpha1.VerrazzanoManagedCluster {
	return &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: v1.ObjectMeta{
			// VMC name is the Rancher cluster ID to guarantee uniqueness
			Name:      rancherCluster.id,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
			Labels: map[string]string{
				createdByLabel: createdByVerrazzano,
			},
		},
		Status: clustersv1alpha1.VerrazzanoManagedClusterStatus{
			RancherRegistration: clustersv1alpha1.RancherRegistration{
				ClusterID: rancherCluster.id,
			},
		},
	}
}

// deleteVMCs deletes any auto-created VMCs that are no longer in Rancher
func (r *RancherClusterSyncer) deleteVMCs(rancherClusters []rancherCluster, log vzlog.VerrazzanoLogger) error {
	// list the VMCs using a selector to only get the auto-created resources
	clusterList := &clustersv1alpha1.VerrazzanoManagedClusterList{}
	selector := &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{createdByLabel: createdByVerrazzano})}
	if err := r.List(context.TODO(), clusterList, &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace}, selector); err != nil {
		log.Infof("Unable to list VMCs: %v", err)
		return err
	}

	// for each VMC, if it does not exist in Rancher, delete it
	for i := range clusterList.Items {
		cluster := clusterList.Items[i] // avoids "G601: Implicit memory aliasing in for loop" linter error
		if cluster.Name == localClusterName {
			continue
		}
		// VMC name is the Rancher cluster ID
		if !clusterInRancher(cluster.Name, rancherClusters) {
			log.Infof("Deleting VMC %s because it is no longer in Rancher", cluster.Name)
			if err := r.Delete(context.TODO(), &cluster); err != nil {
				log.Infof("Unable to delete VMC: %v", err)
				return err
			}
		}
	}

	return nil
}

// clusterInRancher returns true if the cluster id is in the list of clusters in Rancher
func clusterInRancher(clusterID string, rancherClusters []rancherCluster) bool {
	for _, rancherCluster := range rancherClusters {
		if clusterID == rancherCluster.id {
			return true
		}
	}
	return false
}
