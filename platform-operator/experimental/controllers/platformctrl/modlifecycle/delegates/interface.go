// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package delegates

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	modulesv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const ControllerLabel = "verrazzano.io/module"

type DelegateLifecycleReconciler interface {
	Reconcile(log vzlog.VerrazzanoLogger, client clipkg.Client, mlc *modulesv1beta2.ModuleLifecycle) (ctrl.Result, error)
}
