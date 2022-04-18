// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"k8s.io/apimachinery/pkg/api/resource"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	system = "system"
)

//createVMI instantiates the VMI resource
func createVMI(ctx spi.ComponentContext) error {
	if !vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		return nil
	}

	effectiveCR := ctx.EffectiveCR()

	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
	}
	envName := vzconfig.GetEnvName(effectiveCR)

	storage, err := findStorageOverride(ctx.EffectiveCR())
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to get storage overrides: %v", err)
	}
	vmi := newVMI()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), vmi, func() error {
		var existingVMI *vmov1.VerrazzanoMonitoringInstance = nil
		if len(vmi.Spec.URI) > 0 {
			existingVMI = vmi.DeepCopy()
		}

		vmi.Labels = map[string]string{
			"k8s-app":            "verrazzano.io",
			"verrazzano.binding": system,
		}
		cr := ctx.EffectiveCR()
		vmi.Spec.URI = fmt.Sprintf("vmi.system.%s.%s", envName, dnsSuffix)
		vmi.Spec.IngressTargetDNSName = fmt.Sprintf("verrazzano-ingress.%s.%s", envName, dnsSuffix)
		vmi.Spec.ServiceType = "ClusterIP"
		vmi.Spec.AutoSecret = true
		vmi.Spec.SecretsName = verrazzanoSecretName
		vmi.Spec.CascadingDelete = true
		hasDataNodeOverride := hasNodeStorageOverride(ctx.ActualCR(), "nodes.data.requests.storage")
		hasMasterNodeOverride := hasNodeStorageOverride(ctx.ActualCR(), "nodes.master.requests.storage")
		opensearch, err := newOpenSearch(cr, storage, existingVMI, hasDataNodeOverride, hasMasterNodeOverride)
		if err != nil {
			return err
		}
		vmi.Spec.Elasticsearch = *opensearch
		vmi.Spec.Kibana = newOpenSearchDashboards(cr)
		return nil
	})
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to update VMI: %v", err)
	}
	return nil
}

func newVMI() *vmov1.VerrazzanoMonitoringInstance {
	return &vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      system,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
	}
}

func hasNodeStorageOverride(cr *vzapi.Verrazzano, override string) bool {
	openSearch := cr.Spec.Components.Elasticsearch
	if openSearch == nil {
		return false
	}
	for _, arg := range openSearch.ESInstallArgs {
		if arg.Name == override {
			return true
		}
	}

	return false
}

//newOpenSearch creates a new OpenSearch resource for the VMI
// The storage settings for OpenSearch nodes follow this order of precedence:
// 1. ESInstallArgs values
// 2. VolumeClaimTemplate overrides
// 3. Profile values (which show as ESInstallArgs in the ActualCR)
// The data node storage may be changed on update. The master node storage may NOT.
func newOpenSearch(cr *vzapi.Verrazzano, storage *verrazzano.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, hasDataOverride, hasMasterOverride bool) (*vmov1.Elasticsearch, error) {
	if cr.Spec.Components.Elasticsearch == nil {
		return &vmov1.Elasticsearch{}, nil
	}
	opensearchComponent := cr.Spec.Components.Elasticsearch
	opensearch := &vmov1.Elasticsearch{
		Enabled: opensearchComponent.Enabled != nil && *opensearchComponent.Enabled,
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
		// adapt the VPO node list to VMI node list
		Nodes: nodeAdapter(vmi, cr.Spec.Components.Elasticsearch.Nodes, storage),
	}

	// Proxy any ISM policies to the VMI
	for _, policy := range opensearchComponent.Policies {
		opensearch.Policies = append(opensearch.Policies, *policy.DeepCopy())
	}

	// Set the values in the OpenSearch object from the Verrazzano component InstallArgs
	if err := populateOpenSearchFromInstallArgs(opensearch, opensearchComponent); err != nil {
		return nil, err
	}

	setVolumeClaimOverride := func(nodeStorage *vmov1.Storage, hasInstallOverride bool) *vmov1.Storage {
		// Use the volume claim override IFF it is present AND the user did not specify a data node storage override
		if !hasInstallOverride && storage != nil && len(storage.Storage) > 0 {
			nodeStorage = &vmov1.Storage{
				Size: storage.Storage,
			}
		}
		return nodeStorage
	}
	opensearch.MasterNode.Storage = setVolumeClaimOverride(opensearch.MasterNode.Storage, hasMasterOverride)
	opensearch.DataNode.Storage = setVolumeClaimOverride(opensearch.DataNode.Storage, hasDataOverride)

	if vmi != nil {
		// We currently do not support resizing master node PVC
		opensearch.MasterNode.Storage = &vmi.Spec.Elasticsearch.Storage
		if vmi.Spec.Elasticsearch.MasterNode.Storage != nil {
			opensearch.MasterNode.Storage = vmi.Spec.Elasticsearch.MasterNode.Storage.DeepCopy()
		}
		// PVC Names should be preserved
		if vmi.Spec.Elasticsearch.DataNode.Storage != nil {
			opensearch.DataNode.Storage.PvcNames = vmi.Spec.Elasticsearch.DataNode.Storage.PvcNames
		}
	}

	return opensearch, nil
}

//populateOpenSearchFromInstallArgs loops through each of the install args and sets their value in the corresponding
// OpenSearch object
func populateOpenSearchFromInstallArgs(opensearch *vmov1.Elasticsearch, opensearchComponent *vzapi.ElasticsearchComponent) error {
	intSetter := func(val *int32, arg vzapi.InstallArgs) error {
		var intVal int32
		_, err := fmt.Sscan(arg.Value, &intVal)
		if err != nil {
			return err
		}
		*val = intVal
		return nil
	}
	// The install args were designed for helm chart, not controller code.
	// The switch statement is a shim around this design.
	for _, arg := range opensearchComponent.ESInstallArgs {
		switch arg.Name {
		case "nodes.master.replicas":
			if err := intSetter(&opensearch.MasterNode.Replicas, arg); err != nil {
				return err
			}
		case "nodes.master.requests.memory":
			opensearch.MasterNode.Resources.RequestMemory = arg.Value
		case "nodes.ingest.replicas":
			if err := intSetter(&opensearch.IngestNode.Replicas, arg); err != nil {
				return err
			}
		case "nodes.ingest.requests.memory":
			opensearch.IngestNode.Resources.RequestMemory = arg.Value
		case "nodes.data.replicas":
			if err := intSetter(&opensearch.DataNode.Replicas, arg); err != nil {
				return err
			}
		case "nodes.data.requests.memory":
			opensearch.DataNode.Resources.RequestMemory = arg.Value
		case "nodes.data.requests.storage":
			opensearch.DataNode.Storage = &vmov1.Storage{
				Size: arg.Value,
			}
		case "nodes.master.requests.storage":
			opensearch.MasterNode.Storage = &vmov1.Storage{
				Size: arg.Value,
			}
		}
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

func setupSharedVMIResources(ctx spi.ComponentContext) error {
	err := ensureVMISecret(ctx.Client())
	if err != nil {
		return err
	}
	return ensureBackupSecret(ctx.Client())
}

func ensureVMISecret(cli client.Client) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      verrazzanoSecretName,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), cli, secret, func() error {
		if secret.Data["username"] == nil || secret.Data["password"] == nil {
			secret.Data["username"] = []byte(verrazzanoSecretName)
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

func ensureBackupSecret(cli client.Client) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      verrazzanoBackupScrtName,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), cli, secret, func() error {
		// Populating dummy keys for access and secret key so that they are never empty
		if secret.Data[objectstoreAccessKey] == nil || secret.Data[objectstoreAccessSecretKey] == nil {
			key, err := password.GeneratePassword(32)
			if err != nil {
				return err
			}
			secret.Data[objectstoreAccessKey] = []byte(key)
			secret.Data[objectstoreAccessSecretKey] = []byte(key)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func nodeAdapter(vmi *vmov1.VerrazzanoMonitoringInstance, nodes []vzapi.OpenSearchNode, storage *verrazzano.ResourceRequestValues) []vmov1.ElasticsearchNode {
	getQuantity := func(q *resource.Quantity) string {
		if q == nil || q.String() == "0" {
			return ""
		}
		return q.String()
	}
	var vmoNodes []vmov1.ElasticsearchNode
	for _, node := range nodes {
		var storageSize string
		if storage != nil && storage.Storage != "" {
			storageSize = storage.Storage
		}
		if node.Storage != nil {
			storageSize = node.Storage.Size
		}
		resources := vmov1.Resources{}
		if node.Resources != nil {
			resources.RequestCPU = getQuantity(node.Resources.Requests.Cpu())
			resources.LimitCPU = getQuantity(node.Resources.Limits.Cpu())
			resources.RequestMemory = getQuantity(node.Resources.Requests.Memory())
			resources.LimitMemory = getQuantity(node.Resources.Limits.Memory())
		}
		vmoNode := vmov1.ElasticsearchNode{
			Name:      node.Name,
			JavaOpts:  "",
			Replicas:  node.Replicas,
			Roles:     node.Roles,
			Resources: resources,
			Storage: &vmov1.Storage{
				Size: storageSize,
			},
		}
		// if the node was present in an existing VMI and has PVC names, they should be carried over
		setPVCNames(vmi, &vmoNode)
		vmoNodes = append(vmoNodes, vmoNode)
	}
	return vmoNodes
}

//setPVCNames persists any PVC names from an existing VMI
func setPVCNames(vmi *vmov1.VerrazzanoMonitoringInstance, node *vmov1.ElasticsearchNode) {
	if vmi != nil {
		for _, nodeGroup := range vmi.Spec.Elasticsearch.Nodes {
			if nodeGroup.Name == node.Name {
				if nodeGroup.Storage != nil {
					node.Storage.PvcNames = nodeGroup.Storage.PvcNames
				}
				return
			}
		}
	}
}
