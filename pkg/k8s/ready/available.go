// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ready

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	typeDeployment  = "deployment"
	typeStatefulset = "statefulset"
	typeDaemonset   = "daemonset"
)

type (
	isObjectAvailableSig func(client clipkg.Client, nsn types.NamespacedName) error
	AvailabilityObjects  struct {
		StatefulsetNames    []types.NamespacedName
		DeploymentNames     []types.NamespacedName
		DeploymentSelectors []clipkg.ListOption
		DaemonsetNames      []types.NamespacedName
	}
)

func (c *AvailabilityObjects) IsAvailable(log vzlog.VerrazzanoLogger, client clipkg.Client) (string, bool) {
	if err := DeploymentsAreAvailable(client, c.DeploymentNames); err != nil {
		return handleNotAvailableError(log, err)
	}
	if err := DeploymentsAreAvailableBySelector(client, c.DeploymentSelectors); err != nil {
		return handleNotAvailableError(log, err)
	}
	if err := StatefulsetsAreAvailable(client, c.StatefulsetNames); err != nil {
		return handleNotAvailableError(log, err)
	}
	if err := DaemonsetsAreAvailable(client, c.DaemonsetNames); err != nil {
		return handleNotAvailableError(log, err)
	}
	return "", true
}

func handleNotAvailableError(log vzlog.VerrazzanoLogger, err error) (string, bool) {
	log.Progressf(err.Error())
	return err.Error(), false
}

// DeploymentsAreAvailable a list of deployments is available when the expected replicas is equal to the ready replicas
func DeploymentsAreAvailable(client clipkg.Client, deployments []types.NamespacedName) error {
	return objectsAreAvailable(client, deployments, isDeploymentAvailable)
}

// StatefulsetsAreAvailable a list of statefulsets is available when the expected replicas is equal to the ready replicas
func StatefulsetsAreAvailable(client clipkg.Client, statefulsets []types.NamespacedName) error {
	return objectsAreAvailable(client, statefulsets, isStatefulsetAvailable)
}

// DaemonsetsAreAvailable a list of daemonsets is available when the expected replicas is equal to the ready replicas
func DaemonsetsAreAvailable(client clipkg.Client, daemonsets []types.NamespacedName) error {
	return objectsAreAvailable(client, daemonsets, isDaemonsetAvailable)
}

func DeploymentsAreAvailableBySelector(client clipkg.Client, selectors []clipkg.ListOption) error {
	if len(selectors) < 1 {
		return nil
	}
	deploymentList := &appsv1.DeploymentList{}
	if err := client.List(context.TODO(), deploymentList, selectors...); err != nil {
		return handleListFailure(typeDeployment, err)
	}
	if deploymentList.Items == nil || len(deploymentList.Items) < 1 {
		return handleListNotFound(typeDeployment, selectors)
	}
	for _, deploy := range deploymentList.Items {
		nsn := types.NamespacedName{
			Namespace: deploy.Namespace,
			Name:      deploy.Name,
		}
		if err := handleReplicasNotReady(deploy.Status.ReadyReplicas, deploy.Status.Replicas, nsn, typeDeployment); err != nil {
			return err
		}
	}
	return nil
}

func objectsAreAvailable(client clipkg.Client, objectKeys []types.NamespacedName, objectAvailableFunc isObjectAvailableSig) error {
	for _, nsn := range objectKeys {
		if err := objectAvailableFunc(client, nsn); err != nil {
			return err
		}
	}
	return nil
}

func isDeploymentAvailable(client clipkg.Client, nsn types.NamespacedName) error {
	deploy := &appsv1.Deployment{}
	if err := client.Get(context.TODO(), nsn, deploy); err != nil {
		return handleGetError(err, nsn, typeDeployment)
	}
	return handleReplicasNotReady(deploy.Status.ReadyReplicas, deploy.Status.Replicas, nsn, typeDeployment)
}

func isStatefulsetAvailable(client clipkg.Client, nsn types.NamespacedName) error {
	sts := &appsv1.StatefulSet{}
	if err := client.Get(context.TODO(), nsn, sts); err != nil {
		return handleGetError(err, nsn, typeStatefulset)
	}
	return handleReplicasNotReady(sts.Status.ReadyReplicas, sts.Status.Replicas, nsn, typeStatefulset)
}

func isDaemonsetAvailable(client clipkg.Client, nsn types.NamespacedName) error {
	ds := &appsv1.DaemonSet{}
	if err := client.Get(context.TODO(), nsn, ds); err != nil {
		return handleGetError(err, nsn, typeDaemonset)
	}
	return handleReplicasNotReady(ds.Status.NumberReady, ds.Status.DesiredNumberScheduled, nsn, typeDaemonset)
}

func handleGetError(err error, nsn types.NamespacedName, objectType string) error {
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("waiting for %s %v to exist", objectType, nsn)
	}
	return fmt.Errorf("failed getting %s %v: %v", objectType, nsn, err)
}

func handleReplicasNotReady(ready, expected int32, nsn types.NamespacedName, objectType string) error {
	if ready != expected {
		return fmt.Errorf("%s %v not available: %d/%d replicas ready", objectType, nsn, ready, expected)
	}
	return nil
}

func handleListFailure(objectType string, err error) error {
	return fmt.Errorf("failed getting %s: %v", objectType, err)
}

func handleListNotFound(objectType string, selectors []clipkg.ListOption) error {
	return fmt.Errorf("waiting for %s matching selectors %v to exist", objectType, selectors)
}
