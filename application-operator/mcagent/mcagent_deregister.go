// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"errors"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	clustersapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isUnauthorized(err error) bool {
	groupErr := &discovery.ErrGroupDiscoveryFailed{}
	if errors.As(err, &groupErr) {
		for _, err := range groupErr.Groups {
			if apierrors.IsUnauthorized(err) {
				return true
			}
		}
	}
	return false
}

// syncMCAgentDeleteResources deletes the managed cluster resources if the correlating admin VMC gets deleted
func (s *Syncer) syncDeregistration() error {
	if shouldDeregister, err := s.verifyDeregister(); err != nil || !shouldDeregister {
		return err
	}

	s.Log.Infof("Verrazzano Managed Cluster %s/%s has been deleted, cleaning up managed cluster resources", constants.VerrazzanoMultiClusterNamespace, s.ManagedClusterName)
	mcAgentSec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCAgentSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	err := s.LocalClient.Delete(context.TODO(), &mcAgentSec)
	if client.IgnoreNotFound(err) != nil {
		s.Log.Errorf("Failed to delete the managed cluster agent secret %s/%s: %v", constants.MCAgentSecret, constants.VerrazzanoSystemNamespace, err)
		return err
	}

	mcRegSec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	err = s.LocalClient.Delete(context.TODO(), &mcRegSec)
	if client.IgnoreNotFound(err) != nil {
		s.Log.Errorf("Failed to delete the managed cluster registration secret %s/%s: %v", constants.MCRegistrationSecret, constants.VerrazzanoSystemNamespace, err)
		return err
	}
	return nil
}

func (s *Syncer) verifyDeregister() (bool, error) {
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := clustersapi.VerrazzanoManagedCluster{}

	if s.AdminClient == nil {
		return true, nil
	}
	err := s.AdminClient.Get(s.Context, vmcName, &vmc)
	//	if client.IgnoreNotFound(err) != nil && !strings.Contains(err.Error(), "clusters.verrazzano.io/v1alpha1: Unauthorized") {
	if client.IgnoreNotFound(err) != nil && !isUnauthorized(err) {
		reason, code := reasonAndCodeForError(err)
		s.Log.Infof("unwrap: %v, err: %v, reason: %v, code: %d", errors.Unwrap(err), err, reason, code)
		s.Log.Errorf("Failed to get the VMC resources %s/%s from the admin cluster: %v", constants.VerrazzanoMultiClusterNamespace, s.ManagedClusterName, err)
		return false, err
	}
	if err == nil && vmc.DeletionTimestamp.IsZero() {
		s.Log.Debugf("VMC resource %s/%s has been found and is not being deleted, skipping the MC Agent deregistration", constants.VerrazzanoMultiClusterNamespace, s.ManagedClusterName)
		return false, err
	}
	return true, nil
}

func reasonAndCodeForError(err error) (metav1.StatusReason, int32) {
	status, ok := err.(APIStatus)
	if ok || errors.As(err, &status) {
		return status.Status().Reason, status.Status().Code
	}
	return metav1.StatusReasonUnknown, 0
}

type APIStatus interface {
	Status() metav1.Status
}
