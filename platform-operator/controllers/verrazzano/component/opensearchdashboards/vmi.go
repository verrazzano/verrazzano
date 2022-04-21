// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	system = "system"
)

//createVMIforOSD instantiates the VMI resource
func createVMIforOSD(ctx spi.ComponentContext) error {
	if !vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		return nil
	}

	effectiveCR := ctx.EffectiveCR()

	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
	}
	envName := vzconfig.GetEnvName(effectiveCR)

	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to get storage overrides: %v", err)
	}
	vmi := common.NewVMI()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), vmi, func() error {
		vmi.Labels = map[string]string{
			"k8s-app":            "verrazzano.io",
			"verrazzano.binding": system,
		}
		cr := ctx.EffectiveCR()
		vmi.Spec.URI = fmt.Sprintf("vmi.system.%s.%s", envName, dnsSuffix)
		vmi.Spec.IngressTargetDNSName = fmt.Sprintf("verrazzano-ingress.%s.%s", envName, dnsSuffix)
		vmi.Spec.ServiceType = "ClusterIP"
		vmi.Spec.AutoSecret = true
		vmi.Spec.SecretsName = constants.VMISecret
		vmi.Spec.CascadingDelete = true
		vmi.Spec.Kibana = newOpenSearchDashboards(cr)
		return nil
	})
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to update VMI: %v", err)
	}
	return nil
}

func newOpenSearchDashboards(cr *vzapi.Verrazzano) vmov1.Kibana {
	if cr.Spec.Components.Kibana == nil {
		return vmov1.Kibana{}
	}
	kibanaValues := cr.Spec.Components.Kibana
	opensearchDashboards := vmov1.Kibana{
		Enabled: kibanaValues.Enabled != nil && *kibanaValues.Enabled,
		Resources: vmov1.Resources{
			RequestMemory: "192Mi",
		},
	}
	return opensearchDashboards
}
