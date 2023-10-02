// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package quickcreate

import (
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/ocne"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	Ocneoci   = "ocneoci"
	Oke       = "oke"
	Namespace = "NAMESPACE"
)

var (
	//go:embed templates/ocneoci.goyaml
	ocneociTemplate []byte
	//go:embed templates/oke.goyaml
	okeTemplate []byte
	//go:embed templates/ociclusteridentity.goyaml
	ociClusterIdentity []byte

	//go:embed templates/verrazzanofleet-mc-profile.goyaml
	verrazzanoFleet []byte

	clusterTemplateMap = map[string][]byte{
		Ocneoci: ocneociTemplate,
		Oke:     okeTemplate,
	}

	okeClusterName      string
	okeClusterNamespace string
)

type (
	QCContext struct {
		ClusterType string
		Namespace   string
		Client      clipkg.Client
		RawObjects  []byte
		Parameters  input
	}
	input map[string]interface{}
)

func newContext(cli clipkg.Client, clusterType string) (*QCContext, error) {
	qc := &QCContext{
		ClusterType: clusterType,
		Namespace:   pkg.SimpleNameGenerator.New("qc-"),
		Client:      cli,
	}
	if err := qc.setDynamicValues(); err != nil {
		return nil, err
	}

	return qc, nil
}

func (qc *QCContext) setDynamicValues() error {
	rawObjects, parameters, err := qc.getInputValues()
	if err != nil {
		return fmt.Errorf("failed to load template: %v", err)
	}
	qc.Parameters = parameters
	qc.RawObjects = rawObjects
	qc.Parameters[Namespace] = qc.Namespace

	if qc.isOCICluster() {
		err = qc.Parameters.prepareOCI(qc.ClusterType)
		if err != nil {
			return err
		}
	}
	return nil
}

func (qc *QCContext) setup() error {
	if err := qc.Client.Create(context.Background(), qc.namespaceObject()); err != nil {
		return err
	}
	if qc.isOCICluster() {
		if err := qc.applyOCIClusterIdentity(); err != nil {
			return err
		}
	}
	return nil
}

func (qc *QCContext) applyOCIClusterIdentity() error {
	return k8sutil.NewYAMLApplier(qc.Client, "").ApplyBT(ociClusterIdentity, qc.Parameters)
}

func (qc *QCContext) applyVerrazzanoFleet() error {
	return k8sutil.NewYAMLApplier(qc.Client, "").ApplyBT(verrazzanoFleet, qc.Parameters)
}

func (qc *QCContext) applyCluster() error {
	return k8sutil.NewYAMLApplier(qc.Client, "").ApplyBT(qc.RawObjects, qc.Parameters)
}

func (qc *QCContext) getInputValues() ([]byte, input, error) {
	params, err := qc.newParameters()
	okeClusterName = params[ClusterID].(string)
	okeClusterNamespace = qc.Namespace
	if err != nil {
		return nil, nil, err
	}
	b, ok := clusterTemplateMap[qc.ClusterType]
	if !ok {
		return nil, nil, fmt.Errorf("invalid cluster type: %s", qc.ClusterType)
	}
	return b, params, nil
}

func (qc *QCContext) newParameters() (input, error) {
	var i input = map[string]interface{}{
		ClusterID: pkg.SimpleNameGenerator.New("qc-"),
	}
	if err := i.addFileContents(); err != nil {
		return nil, err
	}
	if err := i.addLatestOCNEVersion(qc.Client, qc.ClusterType); err != nil {
		return nil, err
	}
	return i, nil
}

func (qc *QCContext) deleteObject(o clipkg.Object) error {
	err := qc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: o.GetNamespace(),
		Name:      o.GetName(),
	}, o)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !o.GetDeletionTimestamp().IsZero() {
		return errors.New("object is being deleted")
	}
	if err := qc.Client.Delete(context.Background(), o); err != nil {
		return err
	}
	return errors.New("deleting object")
}

func (qc *QCContext) cleanupCAPICluster() error {
	name, ok := qc.Parameters[ClusterID].(string)
	if !ok {
		return nil
	}
	return qc.deleteObject(&v1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: qc.Namespace,
		},
	})
}

func (qc *QCContext) namespaceObject() clipkg.Object {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: qc.Namespace,
		},
	}
}

func (qc *QCContext) isClusterReady() error {
	cluster := &v1beta1.Cluster{}
	if err := qc.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: qc.Namespace,
		Name:      qc.Parameters[ClusterID].(string),
	}, cluster); err != nil {
		return err
	}
	if !cluster.Status.InfrastructureReady || !cluster.Status.ControlPlaneReady || cluster.Status.Phase != string(v1beta1.ClusterPhaseProvisioned) {
		return dumpClusterError(cluster)
	}
	return nil
}

func (i input) addLatestOCNEVersion(client clipkg.Client, clusterType string) error {
	if clusterType != Ocneoci {
		return nil
	}
	const (
		ocneConfigMapName      = "ocne-metadata"
		ocneConfigMapNamespace = "verrazzano-capi"
	)
	cm := &corev1.ConfigMap{}
	if err := client.Get(context.Background(), types.NamespacedName{
		Namespace: ocneConfigMapNamespace,
		Name:      ocneConfigMapName,
	}, cm); err != nil {
		return err
	}
	mapping, ok := cm.Data["mapping"]
	if !ok {
		return errors.New("no OCNE version mapping")
	}
	versions := map[string]*ocne.VersionDefaults{}
	if err := yaml.Unmarshal([]byte(mapping), &versions); err != nil {
		return err
	}
	var v1 *semver.SemVersion
	var ocneVersion string
	for k8sVersion, defaults := range versions {
		v2, err := semver.NewSemVersion(k8sVersion)
		if err != nil {
			return err
		}
		if v1 == nil || v2.IsGreaterThanOrEqualTo(v1) {
			v1 = v2
			ocneVersion = defaults.Release
		}
	}
	i[OcneVersion] = ocneVersion
	return nil
}

func (i input) addFileContents() error {
	files := []string{
		PubKey,
		APIKey,
	}
	for _, filevar := range files {
		path := os.Getenv(filevar)
		if len(path) < 1 {
			return fmt.Errorf("%s env var empty", filevar)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		i[filevar] = string(b)
		if filevar == APIKey {
			i.b64EncodeKV(APIKey, B64Key)
		}
	}
	return nil
}

func (i input) b64EncodeKV(key, encodedKey string) {
	i[encodedKey] = base64.StdEncoding.EncodeToString([]byte(i[key].(string)))
}

func (i input) addEnvValue(key string) error {
	value := os.Getenv(key)
	if len(value) < 1 {
		return fmt.Errorf("no value found for environment key %s", key)
	}
	i[key] = value
	return nil
}

func dumpClusterError(cluster *v1beta1.Cluster) error {
	sb := strings.Builder{}
	if cluster.Status.FailureMessage != nil {
		sb.WriteString(fmt.Sprintf("message: %s", *cluster.Status.FailureMessage))
	}
	if !cluster.Status.ControlPlaneReady {
		sb.WriteString("- control plane is not ready")
	}
	if !cluster.Status.InfrastructureReady {
		sb.WriteString("- infrastructure is not ready")
	}
	for _, cond := range cluster.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			sb.WriteString(fmt.Sprintf("- condition[%s]:", cond.Type))
			sb.WriteString(fmt.Sprintf(" reason: %s", cond.Reason))
			sb.WriteString(fmt.Sprintf(" message: %s", cond.Message))
		}
	}
	return errors.New(sb.String())
}
