// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespacewatch

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	channelBufferSize = 100
)

// namespaceWatcher - holds global instance of NamespacesWatcher.  Required by namespaces util
// functions that don't have access to the NamespacesWatcher context.
var namespaceWatcher *NamespacesWatcher

// NamespacesWatcher periodically checks if new namespaces are added
type NamespacesWatcher struct {
	client   clipkg.Client
	tickTime time.Duration
	log      *zap.SugaredLogger
	shutdown chan int
}

// NewNamespaceWatcher - instantiate a NamespacesWatcher context
func NewNamespaceWatcher(c clipkg.Client, duration time.Duration) *NamespacesWatcher {
	namespaceWatcher = &NamespacesWatcher{
		client:   c,
		tickTime: duration,
		log:      zap.S().With(log.FieldController, "namespacewatcher"),
	}
	return namespaceWatcher
}

func GetNamespaceWatcher() *NamespacesWatcher {
	return namespaceWatcher
}

// Start starts the NamespacesWatcher if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (nw *NamespacesWatcher) Start() {
	if nw.shutdown != nil {
		// already running, so nothing to do
		return
	}
	nw.shutdown = make(chan int, channelBufferSize)

	// goroutine watches namespace resource every p.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		ticker := time.NewTicker(nw.tickTime)
		for {
			select {
			case <-ticker.C:
				// timer event causes namespaces update
				clusterName, rancherSystemProjectID, err := nw.getRancherSystemProjectID()
				if err != nil {
					nw.log.Errorf("%v", err)
				}
				if rancherSystemProjectID != "" && clusterName != "" {
					err = nw.MoveSystemNamespacesToRancherSystemProject(rancherSystemProjectID, clusterName)
					if err != nil {
						nw.log.Errorf("%v", err)
					}
				}
			case <-nw.shutdown:
				// shutdown event causes termination
				ticker.Stop()
				return
			}
		}
	}()
}

// MoveSystemNamespacesToRancherSystemProject Updates the label & annotation with Rancher System Project ID
// For namespaces that have label "verrazzano.io/namespace"
// And that does not have a label "management.cattle.io/system-namespace"
// "management.cattle.io/system-namespace" namespace label indicates it is managed by Rancher
func (nw *NamespacesWatcher) MoveSystemNamespacesToRancherSystemProject(rancherSystemProjectID string, clusterName string) error {

	isRancherReady, err := nw.IsRancherReady()
	if err != nil {
		return err
	}
	if isRancherReady {
		var rancherSystemProjectIDAnnotation = clusterName + ":" + rancherSystemProjectID

		namespaceList := &v1.NamespaceList{}
		err := nw.client.List(context.TODO(), namespaceList, &clipkg.ListOptions{})
		if err != nil {
			return err
		}

		for i := range namespaceList.Items {
			if namespaceList.Items[i].Labels == nil {
				namespaceList.Items[i].Labels = map[string]string{}
			}
			if namespaceList.Items[i].Annotations == nil {
				namespaceList.Items[i].Annotations = map[string]string{}
			}
			_, rancherProjectIDAnnotationExists := namespaceList.Items[i].Annotations[RancherProjectIDLabelKey]
			if isVerrazzanoManagedNamespace(&(namespaceList.Items[i])) && !rancherProjectIDAnnotationExists {
				nw.log.Infof("Updating the Labels and Annotations of a Namespace %v with Rancher System Project ID %v", namespaceList.Items[i].Namespace, rancherSystemProjectID)
				namespaceList.Items[i].Annotations[RancherProjectIDLabelKey] = rancherSystemProjectIDAnnotation
				namespaceList.Items[i].Labels[RancherProjectIDLabelKey] = rancherSystemProjectID
				if err = nw.client.Update(context.TODO(), &(namespaceList.Items[i]), &clipkg.UpdateOptions{}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Pause pauses the NamespaceWatch if it was running.
// It is safe to call Pause multiple times
func (nw *NamespacesWatcher) Pause() {
	if nw.shutdown != nil {
		close(nw.shutdown)
		nw.shutdown = nil
	}
}
