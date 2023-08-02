// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespacewatch

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// isVerrazzanoManagedNamespace checks if the given namespace is managed by Verrazzano
func isVerrazzanoManagedNamespace(ns *v1.Namespace) bool {
	_, verrazzanoSystemLabelExists := ns.Labels[constants.VerrazzanoManagedKey]
	value, rancherSystemLabelExists := ns.Annotations[rancher.RancherSysNS]
	if verrazzanoSystemLabelExists && !rancherSystemLabelExists {
		return true
	}
	if rancherSystemLabelExists && value != "true" && verrazzanoSystemLabelExists {
		return true
	}
	return false
}

// getVerrazzanoResource fetches a Verrazzano resource, if one exists
func getVerrazzanoResource(client clipkg.Client) (*vzapi.Verrazzano, error) {
	var err error
	vzList := &vzapi.VerrazzanoList{}
	if err = client.List(context.TODO(), vzList); err != nil {
		return nil, err
	}
	if len(vzList.Items) != 1 {
		return nil, fmt.Errorf("verrazzano resource list is not equal to 1")
	}
	return &vzList.Items[0], nil
}

func newLogger(vz *vzapi.Verrazzano) (vzlog.VerrazzanoLogger, error) {
	zaplog, err := log.BuildZapLoggerWithLevel(2, zapcore.ErrorLevel)
	if err != nil {
		return nil, err
	}
	// The ID below needs to be different from the main thread, so add a suffix.
	return vzlog.ForZapLogger(&vzlog.ResourceConfig{
		Name:           vz.Name,
		Namespace:      vz.Namespace,
		ID:             string(vz.UID) + "namespacewatch",
		Generation:     vz.Generation,
		ControllerName: "namespacewatcher",
	}, zaplog), nil
}
