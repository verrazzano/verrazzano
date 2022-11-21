// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package opensearch

import (
	"context"
	"fmt"
	"testing"

	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	profilesRelativePath     = "../../../../manifests/profiles"
	testCaseInstallArgsErr   = "frominstallargserr"
	testCaseGeneralOverrides = "overrides"
	systemesdata             = "system-es-data"
	podtemplatehash          = "pod-template-hash"
	depKube                  = "deployment.kubernetes.io/revision"
	systemesingest           = "system-es-ingest"
)

var dnsComponents = vzapi.ComponentSpec{
	DNS: &vzapi.DNSComponent{
		External: &vzapi.External{Suffix: "blah"},
	},
}

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Elasticsearch: &vzapi.ElasticsearchComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

// TestPreUpgrade tests the OpenSearch PreUpgrade call
// GIVEN an OpenSearch component
//
//	WHEN I call PreUpgrade with defaults
//	THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = "../../../../helm_config"
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestComponentNamespace tests the OpenSearch ComponentNamespace call
// GIVEN an OpenSearch component
//
//	WHEN I call ComponentNamespace with defaults
//	THEN the component namespace is returned
func TestComponentNamespace(t *testing.T) {
	NameSpace := NewComponent().Namespace()
	assert.Equal(t, NameSpace, constants.VerrazzanoSystemNamespace)

}

// TestShouldInstallBeforeUpgrade tests the ShouldInstallBeforeUpgrade call
// GIVEN an OpenSearch component
//
//	WHEN I call ShouldInstallBeforeUpgrade with defaults
//	THEN false is returned
func TestShouldInstallBeforeUpgrade(t *testing.T) {
	val := NewComponent().ShouldInstallBeforeUpgrade()
	assert.Equal(t, val, false)

}

// TestGetDependencies tests the GetDependencies call
// GIVEN an OpenSearch component
//
//	WHEN I call GetDependencies with defaults
//	THEN a string array containing different dependencies is returned
func TestGetDependencies(t *testing.T) {
	strArray := NewComponent().GetDependencies()
	expArray := []string{"verrazzano-network-policies", "verrazzano-monitoring-wls"}
	assert.Equal(t, expArray, strArray)

}

// TestGetMinVerrazzanoVersion tests the GetMinVerrazzanoVersion call
// GIVEN an OpenSearch component
//
//	WHEN I call GetMinVerrazzanoVersion with defaults
//	THEN a string containing MinVerrazzanoVersion is returned
func TestGetMinVerrazzanoVersion(t *testing.T) {
	minVer := NewComponent().GetMinVerrazzanoVersion()
	expVer := constants.VerrazzanoVersion1_0_0
	assert.Equal(t, expVer, minVer)

}

// TestGetJSONName tests the GetJSONName call
// GIVEN an OpenSearch component
//
//	WHEN I call GetJSONName with defaults
//	THEN a string containing JSONName is returned
func TestGetJSONName(t *testing.T) {
	jsonName := NewComponent().GetJSONName()
	assert.Equal(t, jsonName, "opensearch")

}

// TestGetOverrides tests the GetOverrides call
// GIVEN an OpenSearch component and a runtime object
//
//	WHEN I call GetOverrides with defaults
//	THEN an interface containing overrides is returned
func TestGetOverrides(t *testing.T) {
	tests := []struct {
		name   string
		object runtime.Object
		want   interface{}
	}{
		{
			"TestGetOverridesWithInvalidOverrides",
			nil,
			[]vzapi.Overrides{},
		},
		{
			"TestGetOverrides With No Opensearch",
			&vzapi.Verrazzano{},
			[]vzapi.Overrides{},
		},
		{
			"TestGetOverridesWithV1Beta CR with no Opensearch",
			&installv1beta1.Verrazzano{},
			[]installv1beta1.Overrides{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewComponent().GetOverrides(tt.object))

		})
	}
}

// TestValidateInstall tests the ValidateInstall call
// GIVEN an OpenSearch component and a vzapi
//
//	WHEN I call ValidateInstall with defaults
//	THEN an bool value is returned
func TestValidateInstall(t *testing.T) {
	var tests = []struct {
		name     string
		vz       *vzapi.Verrazzano
		hasError bool
	}{
		{
			"no duplication when component has no args",
			emptyComponent,
			false,
		},
		{
			"no duplication when component does not duplicate args",
			createVZ(&vzapi.ElasticsearchComponent{
				ESInstallArgs: createInstallArgs(3, 3, 3),
				Nodes: []vzapi.OpenSearchNode{
					createNG("a", 1, nil),
					createNG("b", 2, nil),
					createNG("c", 3, nil),
				},
			}),
			false,
		},
		{
			"duplication when component has Node groups with the same name",
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					createNG("master", 3, nil),
					createNG("data", 3, nil),
					createNG("ingest", 3, nil),
					createNG("master", 3, nil),
				},
			}),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewComponent().ValidateInstall(tt.vz); (err != nil) != tt.hasError {
				assert.Error(t, err)
			}
		})
	}

}

// TestValidateInstallV1Beta1 tests the ValidateInstallV1Beta1 call
// GIVEN an OpenSearch component and a installv1beta1
//
//	WHEN I call ValidateInstallV1Beta1 with defaults
//	THEN an bool value is returned
func TestValidateInstallV1Beta1(t *testing.T) {
	var tests = []struct {
		name     string
		vz       *vzapi.Verrazzano
		hasError bool
	}{
		{
			"no duplication when component has no args",
			emptyComponent,
			false,
		},
		{
			"no duplication when component does not duplicate args",
			createVZ(&vzapi.ElasticsearchComponent{
				ESInstallArgs: createInstallArgs(3, 3, 3),
				Nodes: []vzapi.OpenSearchNode{
					createNG("a", 1, nil),
					createNG("b", 2, nil),
					createNG("c", 3, nil),
				},
			}),
			false,
		},
		{
			"duplication when component has Node groups with the same name",
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					createNG("master", 3, nil),
					createNG("data", 3, nil),
					createNG("ingest", 3, nil),
					createNG("master", 3, nil),
				},
			}),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1beta1vz := &installv1beta1.Verrazzano{}
			assert.NoError(t, tt.vz.ConvertTo(v1beta1vz))
			if err := NewComponent().ValidateInstallV1Beta1(v1beta1vz); (err != nil) != tt.hasError {
				assert.Error(t, err)
			}
		})
	}

}

// TestMonitorOverrides tests the MonitorOverrides call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call MonitorOverrides with defaults
//	THEN an true is returned
func TestMonitorOverrides(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	val := NewComponent().MonitorOverrides(ctx)
	assert.Equal(t, val, true)
}

// TestIsOperatorInstallSupported tests the IsOperatorInstallSupported call
// GIVEN an OpenSearch component
//
//	WHEN I call IsOperatorInstallSupported with defaults
//	THEN an true is returned
func TestIsOperatorInstallSupported(t *testing.T) {
	val := NewComponent().IsOperatorInstallSupported()
	assert.Equal(t, val, true)
}

// TestIsInstalled tests the IsInstalled call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call IsInstalled with defaults
//	THEN a value showing the status of installation and nil error is returned
func TestIsInstalled(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset,
				Labels:    map[string]string{"app": masterAppName},
			},
		},
	).Build()

	vz := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	val, err := NewComponent().IsInstalled(ctx)
	assert.Equal(t, val, true)
	assert.NoError(t, err)

	clientFake := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctxnew := spi.NewFakeContext(clientFake, vz, nil, false)
	val, err = NewComponent().IsInstalled(ctxnew)
	assert.Equal(t, val, false)
	assert.NoError(t, err)

}

// TestReconcile tests the Reconcile call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call Reconcile with defaults
//	THEN a nil error is returned
func TestReconcile(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	err := NewComponent().Reconcile(ctx)
	assert.NoError(t, err)
}

// TestIsOperatorUninstallSupported tests the IsOperatorUninstallSupported call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call IsOperatorUninstallSupported with defaults
//	THEN a false value is returned
func TestIsOperatorUninstallSupported(t *testing.T) {
	val := NewComponent().IsOperatorUninstallSupported()
	assert.Equal(t, val, false)
}

// TestPreUninstall tests the PreUninstall call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call PreUninstall with defaults
//	THEN a nil error is returned
func TestPreUninstall(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	err := NewComponent().PreUninstall(ctx)
	assert.NoError(t, err)
}

// TestUninstall tests the Uninstall call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call Uninstall with defaults
//	THEN a nil error is returned
func TestUninstall(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	err := NewComponent().Uninstall(ctx)
	assert.NoError(t, err)
}

// TestPostUninstall tests the PostUninstall call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call PostUninstall with defaults
//	THEN a nil error is returned
func TestPostUninstall(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	err := NewComponent().PostUninstall(ctx)
	assert.NoError(t, err)
}

// TestIsAvailable tests the IsAvailable call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call IsAvailable with defaults
//	THEN a value showing availability of opensearch and reason for its unavailability, if any are returned
func TestIsAvailable(t *testing.T) {
	tests := []struct {
		name              string
		component         opensearchComponent
		args              spi.ComponentContext
		expectedReason    string
		expectedAvailable vzapi.ComponentAvailability
	}{
		{
			"TestIsAvailable",
			opensearchComponent{},
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false),
			"",
			vzapi.ComponentAvailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, available := tt.component.IsAvailable(tt.args)
			assert.Equal(t, reason, tt.expectedReason)
			assert.Equal(t, available, tt.expectedAvailable)
		})
	}

}

// TestIsReadyForComponent tests the IsReady call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call IsReady with defaults
//	THEN a bool value is returned
func TestIsReadyForComponent(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Console:       &vzapi.ConsoleComponent{Enabled: &falseValue},
		Fluentd:       &vzapi.FluentdComponent{Enabled: &falseValue},
		Kibana:        &vzapi.KibanaComponent{Enabled: &falseValue},
		Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
		Prometheus:    &vzapi.PrometheusComponent{Enabled: &falseValue},
		Grafana:       &vzapi.GrafanaComponent{Enabled: &falseValue},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      vmoDeployment,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}).Build()
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.False(t, NewComponent().IsReady(ctx))

	clientnew := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-0", esDataDeployment),
				Labels: map[string]string{
					"app":   systemesdata,
					"index": "0",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":   systemesdata,
						"index": "0",
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-0-95d8c5d96-m6mbr", esDataDeployment),
				Labels: map[string]string{
					podtemplatehash: "95d8c5d96",
					"app":           systemesdata,
					"index":         "0",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        fmt.Sprintf("%s-0-95d8c5d96", esDataDeployment),
				Annotations: map[string]string{depKube: "1"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-1", esDataDeployment),
				Labels: map[string]string{
					"app":   systemesdata,
					"index": "1",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":   systemesdata,
						"index": "1",
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-1-95d8c5d96-m6mbr", esDataDeployment),
				Labels: map[string]string{
					podtemplatehash: "95d8c5d96",
					"app":           systemesdata,
					"index":         "1",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        fmt.Sprintf("%s-1-95d8c5d96", esDataDeployment),
				Annotations: map[string]string{depKube: "1"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esIngestDeployment,
				Labels:    map[string]string{"app": systemesingest},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": systemesingest},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 2,
				Replicas:          2,
				UpdatedReplicas:   2,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esIngestDeployment + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podtemplatehash: "95d8c5d96",
					"app":           systemesingest,
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esIngestDeployment + "-95d8c5d96-x1v76",
				Labels: map[string]string{
					podtemplatehash: "95d8c5d96",
					"app":           systemesingest,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        esIngestDeployment + "-95d8c5d96",
				Annotations: map[string]string{depKube: "1"},
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset,
				Labels:    map[string]string{"app": masterAppName},
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": masterAppName},
				},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   1,
				UpdatedReplicas: 1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset + "-0",
				Labels: map[string]string{
					"app":                      masterAppName,
					"controller-revision-hash": "test-95d8c5d96",
				},
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: ComponentNamespace}},
		&appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-95d8c5d96",
				Namespace: ComponentNamespace,
			},
			Revision: 1,
		}).Build()

	vznew := &vzapi.Verrazzano{}
	vz.Spec.Components = vzapi.ComponentSpec{
		Elasticsearch: &vzapi.ElasticsearchComponent{
			ESInstallArgs: []vzapi.InstallArgs{
				{
					Name:  "nodes.master.replicas",
					Value: "1",
				},
				{
					Name:  "nodes.data.replicas",
					Value: "2",
				},
				{
					Name:  "nodes.ingest.replicas",
					Value: "2",
				},
			},
		},
	}
	ctxnew := spi.NewFakeContext(clientnew, vznew, nil, false)
	assert.True(t, NewComponent().IsReady(ctxnew))

}

// TestPreInstall tests the OpenSearch PreInstall call
// GIVEN an OpenSearch component
//
//	WHEN I call PreInstall when dependencies are met
//	THEN no error is returned
func TestPreInstall(t *testing.T) {
	c := createPreInstallTestClient()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	err := NewComponent().PreInstall(ctx)
	assert.NoError(t, err)
}

// TestInstall tests the OpenSearch Install call
// GIVEN an OpenSearch component
//
//	WHEN I call Install when dependencies are met
//	THEN no error is returned
func TestInstall(t *testing.T) {
	c := createPreInstallTestClient()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, nil, false)
	err := NewComponent().Install(ctx)
	assert.NoError(t, err)
}

// TestPostInstall tests the OpenSearch PostInstall call
// GIVEN an OpenSearch component
//
//	WHEN I call PostInstall
//	THEN no error is returned
func TestPostInstall(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, nil, false)
	vzComp := NewComponent()

	// PostInstall will fail because the expected VZ ingresses are not present in cluster
	err := vzComp.PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	// now get all the ingresses for VZ and add them to the fake K8S and ensure that PostInstall succeeds
	// when all the ingresses are present in the cluster
	vzIngressNames := vzComp.(opensearchComponent).GetIngressNames(ctx)
	for _, ingressName := range vzIngressNames {
		_ = c.Create(context.TODO(), &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName.Name, Namespace: ingressName.Namespace},
		})
	}
	for _, certName := range vzComp.(opensearchComponent).GetCertificateNames(ctx) {
		time := metav1.Now()
		_ = c.Create(context.TODO(), &certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certName.Name, Namespace: certName.Namespace},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
				},
			},
		})
	}
	err = vzComp.PostInstall(ctx)
	assert.NoError(t, err)
}

// TestPostInstallCertsNotReady tests the OpenSearch PostInstall call
// GIVEN an OpenSearch component
//
//	WHEN I call PostInstall and the certificates aren't ready
//	THEN a retryable error is returned
func TestPostInstallCertsNotReady(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, nil, false)
	vzComp := NewComponent()

	// PostInstall will fail because the expected VZ ingresses are not present in cluster
	err := vzComp.PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	// now get all the ingresses for VZ and add them to the fake K8S and ensure that PostInstall succeeds
	// when all the ingresses are present in the cluster
	vzIngressNames := vzComp.(opensearchComponent).GetIngressNames(ctx)
	for _, ingressName := range vzIngressNames {
		_ = c.Create(context.TODO(), &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName.Name, Namespace: ingressName.Namespace},
		})
	}
	for _, certName := range vzComp.(opensearchComponent).GetCertificateNames(ctx) {
		time := metav1.Now()
		_ = c.Create(context.TODO(), &certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certName.Name, Namespace: certName.Namespace},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
				},
			},
		})
	}
	err = vzComp.PostInstall(ctx)

	expectedErr := spi2.RetryableError{
		Source:    vzComp.Name(),
		Operation: "Check if certificates are ready",
	}
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestGetCertificateNames tests the OpenSearch GetCertificateNames call
// GIVEN an OpenSearch component
//
//	WHEN I call GetCertificateNames
//	THEN the correct number of certificate names are returned based on what is enabled
func TestGetCertificateNames(t *testing.T) {
	vmiEnabled := true
	vz := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: "blah"},
				},
				Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &vmiEnabled},
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)
	vzComp := NewComponent()

	certNames := vzComp.GetCertificateNames(ctx)
	assert.Len(t, certNames, 1, "Unexpected number of cert names")
}

// TestUpgrade tests the OpenSearch Upgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN an OpenSearch component upgrading from 1.1.0 to 1.2.0
//
//	WHEN I call Upgrade
//	THEN no error is returned
func TestUpgrade(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Version:    "v1.2.0",
			Components: dnsComponents,
		},
		Status: vzapi.VerrazzanoStatus{Version: "1.1.0"},
	}, nil, false)
	err := NewComponent().Upgrade(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the OpenSearch PostUpgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN an OpenSearch component upgrading from 1.1.0 to 1.2.0
//
//	WHEN I call PostUpgrade
//	THEN no error is returned
func TestPostUpgrade(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
		Status: vzapi.VerrazzanoStatus{
			Version: "v1.1.0",
		},
	}, nil, false)
	comp := NewComponent()

	// PostUpgrade will fail because the expected VZ ingresses are not present in cluster
	err := comp.PostUpgrade(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	// now get all the ingresses for VZ and add them to the fake K8S and ensure that PostUpgrade succeeds
	// when all the ingresses are present in the cluster
	vzIngressNames := comp.(opensearchComponent).GetIngressNames(ctx)
	for _, ingressName := range vzIngressNames {
		_ = c.Create(context.TODO(), &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName.Name, Namespace: ingressName.Namespace},
		})
	}
	for _, certName := range comp.(opensearchComponent).GetCertificateNames(ctx) {
		time := metav1.Now()
		_ = c.Create(context.TODO(), &certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certName.Name, Namespace: certName.Namespace},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
				},
			},
		})
	}
	err = comp.PostUpgrade(ctx)
	assert.NoError(t, err)
}

func createPreInstallTestClient(extraObjs ...client.Object) client.Client {
	objs := []client.Object{}
	objs = append(objs, extraObjs...)
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(objs...).Build()
	return c
}

// TestIsEnabledNilOpenSearch tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OpenSearch component is enabled
//	THEN true is returned
func TestIsEnabledNilOpenSearch(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OpenSearch component is nil
//	THEN true is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OpenSearch component enabled is nil
//	THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OpenSearch component is explicitly enabled
//	THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OpenSearch component is explicitly disabled
//	THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

func getBoolPtr(b bool) *bool {
	return &b
}

func TestValidateUpdate(t *testing.T) {
	disabled := false
	var pvc1Gi, _ = resource.ParseQuantity("1Gi")
	var pvc2Gi, _ = resource.ParseQuantity("2Gi")
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-installargs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "emptyDir to PVC in defaultVolumeSource",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "PVC to emptyDir in volumeSource",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "resize pvc in defaultVolumeSource",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc1Gi,
									},
								},
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc2Gi,
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "resize pvc in ESInstallArgs",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{
								{
									Name:  "nodes.data.requests.storage",
									Value: "1Gi",
								},
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{
								{
									Name:  "nodes.data.requests.storage",
									Value: "2Gi",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disable-opensearch",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &disabled},
					},
				},
			},
			wantErr: true,
		},
		{
			// Change to OS installargs allowed, persistence changes are supported
			name: "change-os-installargs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			err := c.ValidateUpdate(tt.old, tt.new)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
