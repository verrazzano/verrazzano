// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var metricsTemplate = &vzapi.MetricsTemplate{
	TypeMeta: k8smeta.TypeMeta{
		Kind:       vzconst.MetricsTemplateKind,
		APIVersion: vzconst.MetricsTemplateAPIVersion,
	},
	ObjectMeta: k8smeta.ObjectMeta{
		Namespace: testMetricsTemplateNamespace,
		Name:      testMetricsTemplateName,
	},
}

var metricsBinding = &vzapi.MetricsBinding{
	ObjectMeta: k8smeta.ObjectMeta{
		Namespace: testMetricsBindingNamespace,
		Name:      testMetricsBindingName,
	},
	Spec: vzapi.MetricsBindingSpec{
		MetricsTemplate: vzapi.NamespaceName{
			Namespace: testMetricsTemplateNamespace,
			Name:      testMetricsTemplateName,
		},
		PrometheusConfigMap: vzapi.NamespaceName{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      testConfigMapName,
		},
		Workload: vzapi.Workload{
			Name: testDeploymentName,
			TypeMeta: k8smeta.TypeMeta{
				Kind:       vzconst.DeploymentWorkloadKind,
				APIVersion: deploymentGroup + "/" + deploymentVersion,
			},
		},
	},
}

var plainWorkload = &k8sapps.Deployment{
	ObjectMeta: k8smeta.ObjectMeta{
		Namespace: testMetricsBindingNamespace,
		Name:      testDeploymentName,
	},
}

// The namespace has to contain labels for the template
var plainNs = &k8score.Namespace{
	ObjectMeta: k8smeta.ObjectMeta{
		Name: testMetricsBindingNamespace,
		Labels: map[string]string{
			"test": "test",
		},
	},
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	// _ = clientgoscheme.AddToScheme(scheme)
	_ = k8sapps.AddToScheme(scheme)
	//	vzapi.AddToScheme(scheme)
	_ = k8score.AddToScheme(scheme)
	//	certapiv1alpha2.AddToScheme(scheme)
	_ = k8net.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	log := zap.S().With("test")
	scheme := newScheme()
	reconciler := Reconciler{
		Client:  c,
		Log:     log,
		Scheme:  scheme,
		Scraper: "istio-system/prometheus",
	}
	return reconciler
}
