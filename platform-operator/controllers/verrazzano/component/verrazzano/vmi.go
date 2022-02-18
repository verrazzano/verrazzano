// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

const (
	nodeExporter = "node-exporter"
	system       = "system"
)

//createVMI instantiates the VMI resource and the Grafana Dashboards configmap
func createVMI(ctx spi.ComponentContext) error {
	if !isVMOEnabled(ctx.EffectiveCR()) {
		return nil
	}

	if err := createGrafanaConfigMaps(ctx); err != nil {
		return err
	}
	values := &verrazzanoValues{}
	if err := appendVerrazzanoValues(ctx, values); err != nil {
		return err
	}
	storage, err := findStorageOverride(ctx.EffectiveCR())
	if err != nil {
		return err
	}

	existingVMI := getExistingVMI(ctx.Client())
	vmi := newVMI()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), vmi, func() error {
		vmi.Labels = map[string]string{
			"k8s-app":            "verrazzano.io",
			"verrazzano.binding": system,
		}
		cr := ctx.EffectiveCR()
		vmi.Spec.URI = fmt.Sprintf("vmi.system.%s.%s", values.Config.EnvName, values.Config.DNSSuffix)
		vmi.Spec.IngressTargetDNSName = fmt.Sprintf("verrazzano-ingress.%s.%s", values.Config.EnvName, values.Config.DNSSuffix)
		vmi.Spec.ServiceType = "ClusterIP"
		vmi.Spec.AutoSecret = true
		vmi.Spec.SecretsName = ComponentName
		vmi.Spec.CascadingDelete = true
		vmi.Spec.Grafana = newGrafana(cr, storage, existingVMI)
		vmi.Spec.Prometheus = newPrometheus(cr, storage, existingVMI)
		opensearch, err := newOpenSearch(cr, storage, existingVMI)
		if err != nil {
			return err
		}
		vmi.Spec.Elasticsearch = *opensearch
		vmi.Spec.Kibana = newOpenSearchDashboards(cr)
		return nil
	})
	return err
}

func getExistingVMI(cli client.Client) *vmov1.VerrazzanoMonitoringInstance {
	existingVMI := &vmov1.VerrazzanoMonitoringInstance{}
	namespacedName := types.NamespacedName{Name: system, Namespace: globalconst.VerrazzanoSystemNamespace}
	err := cli.Get(context.TODO(), namespacedName, existingVMI)
	if err != nil {
		return nil
	}
	return existingVMI
}

func newVMI() *vmov1.VerrazzanoMonitoringInstance {
	return &vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      system,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
	}
}

func newGrafana(cr *vzapi.Verrazzano, storage *resourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance) vmov1.Grafana {
	if cr.Spec.Components.Grafana == nil {
		return vmov1.Grafana{}
	}
	grafanaValues := cr.Spec.Components.Grafana
	grafana := vmov1.Grafana{
		Enabled:              grafanaValues.Enabled != nil && *grafanaValues.Enabled,
		DashboardsConfigMap:  "system-dashboards",
		DatasourcesConfigMap: "vmi-system-datasources",
		Resources: vmov1.Resources{
			RequestMemory: "48Mi",
		},
		Storage: vmov1.Storage{},
	}
	if vmi != nil {
		grafana.Storage.PvcNames = vmi.Spec.Grafana.Storage.PvcNames
	}

	if storage != nil {
		grafana.Storage.Size = storage.Storage
	}

	return grafana
}

func newPrometheus(cr *vzapi.Verrazzano, storage *resourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance) vmov1.Prometheus {
	if cr.Spec.Components.Prometheus == nil {
		return vmov1.Prometheus{}
	}
	prometheusValues := cr.Spec.Components.Prometheus
	prometheus := vmov1.Prometheus{
		Enabled: prometheusValues.Enabled != nil && *prometheusValues.Enabled,
		Resources: vmov1.Resources{
			RequestMemory: "128Mi",
		},
		Storage: vmov1.Storage{},
	}

	if vmi != nil {
		prometheus.Storage.PvcNames = vmi.Spec.Prometheus.Storage.PvcNames
	}

	if storage != nil {
		prometheus.Storage.Size = storage.Storage
	}

	return prometheus
}

func newOpenSearch(cr *vzapi.Verrazzano, storage *resourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance) (*vmov1.Elasticsearch, error) {
	if cr.Spec.Components.Elasticsearch == nil {
		return &vmov1.Elasticsearch{}, nil
	}
	opensearchValues := cr.Spec.Components.Elasticsearch
	opensearch := &vmov1.Elasticsearch{
		Enabled: opensearchValues.Enabled != nil && *opensearchValues.Enabled,
		Storage: vmov1.Storage{},
		MasterNode: vmov1.ElasticsearchNode{
			Resources: vmov1.Resources{},
		},
		IngestNode: vmov1.ElasticsearchNode{
			Resources: vmov1.Resources{
				RequestMemory: "4.8Gi",
			},
		},
		DataNode: vmov1.ElasticsearchNode{
			Resources: vmov1.Resources{
				RequestMemory: "2.5Gi",
			},
		},
	}

	if storage != nil {
		opensearch.Storage.Size = storage.Storage
	}

	if vmi != nil {
		opensearch.Storage.PvcNames = vmi.Spec.Elasticsearch.Storage.PvcNames
	}

	intSetter := func(val *int32, arg vzapi.InstallArgs) error {
		intVal, err := strconv.Atoi(arg.Value)
		if err != nil {
			return err
		}
		*val = int32(intVal)
		return nil
	}

	// The install args were designed for helm chart, not controller code.
	// The switch statement is a shim around this design.
	for _, arg := range opensearchValues.ESInstallArgs {
		switch arg.Name {
		case "nodes.master.replicas":
			if err := intSetter(&opensearch.MasterNode.Replicas, arg); err != nil {
				return nil, err
			}
		case "nodes.master.requests.memory":
			opensearch.MasterNode.Resources.RequestMemory = arg.Value
		case "nodes.ingest.replicas":
			if err := intSetter(&opensearch.IngestNode.Replicas, arg); err != nil {
				return nil, err
			}
		case "nodes.ingest.requests.memory":
			opensearch.IngestNode.Resources.RequestMemory = arg.Value
		case "nodes.data.replicas":
			if err := intSetter(&opensearch.DataNode.Replicas, arg); err != nil {
				return nil, err
			}
		case "nodes.data.requests.memory":
			opensearch.DataNode.Resources.RequestMemory = arg.Value
		case "nodes.data.requests.storage":
			opensearch.Storage.Size = arg.Value
		}
	}

	return opensearch, nil
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

func setupSharedVMIResources(ctx spi.ComponentContext) error {
	return ensureVMISecret(ctx.Client())
}

func ensureVMISecret(cli client.Client) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), cli, secret, func() error {
		if secret.Data["username"] == nil || secret.Data["password"] == nil {
			secret.Data["username"] = []byte(ComponentName)
			pw, err := password.GeneratePassword(16)
			if err != nil {
				return err
			}
			secret.Data["password"] = []byte(pw)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
