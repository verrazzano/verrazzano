// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"
	"reflect"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/util"
)

const (
	multiclusterTestNamespace = "multiclustertest"
	crdDir                    = "../../config/crd/bases"
	timeout                   = 2 * time.Minute
	pollInterval              = 40 * time.Millisecond
)

var (
	multiclusterCrds = []string{
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusternamespaces.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclustersecrets.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterconfigmaps.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclustercomponents.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterapplicationconfigurations.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterloggingscopes.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_verrazzanoprojects.yaml", crdDir),
	}
)

var _ = ginkgo.Describe("Testing Multi-Cluster CRDs", func() {
	ginkgo.It("MultiCluster CRDs can be applied", func() {
		for _, crd := range multiclusterCrds {
			_, stderr := util.Kubectl(fmt.Sprintf("apply -f %v", crd))
			gomega.Expect(stderr).To(gomega.Equal(""), fmt.Sprintf("Failed to apply CRD %v", crd))
		}
	})
	ginkgo.It("VerrazzanoProject can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
	ginkgo.It("Apply MultiClusterSecret creates K8S secret", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_secret_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcsecret, err := K8sClient.GetMultiClusterSecret(multiclusterTestNamespace, "mymcsecret")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return secretExistsWithData(multiclusterTestNamespace, "mymcsecret", mcsecret.Spec.Template.Data)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply MultiClusterComponent creates OAM component ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_component_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcComp, err := K8sClient.GetMultiClusterComponent(multiclusterTestNamespace, "mymccomp")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return componentExistsWithFields(multiclusterTestNamespace, "mymccomp", mcComp)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})

var _ = ginkgo.Describe("Testing MultiClusterConfigMap", func() {
	ginkgo.It("Apply MultiClusterConfigMap creates a ConfigMap ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_configmap_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "mymcconfigmap")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return configMapExistsMatchingMCConfigMap(
				multiclusterTestNamespace,
				"mymcconfigmap",
				mcConfigMap,
			)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply Invalid MultiClusterConfigMap results in Failed Status", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_configmap_INVALID.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		gomega.Eventually(func() bool {
			// Expecting a failed state value in the MultiClusterConfigMap since creation of
			// underlying config map should fail for invalid config map
			mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "invalid-mccm")
			return err == nil && mcConfigMap.Status.State == clustersv1alpha1.Failed
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})

var _ = ginkgo.Describe("Testing MultiClusterLoggingScope", func() {
	ginkgo.It("Apply MultiClusterLoggingScope creates a LoggingScope ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_loggingscope_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcLogScope, err := K8sClient.GetMultiClusterLoggingScope(multiclusterTestNamespace, "mymcloggingscope")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return loggingScopeExistsWithFields(multiclusterTestNamespace, "mymcloggingscope", mcLogScope)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})
var _ = ginkgo.Describe("Testing MultiClusterApplicationConfiguration", func() {
	ginkgo.It("MultiClusterApplicationConfiguration can be created ", func() {
		// First apply the hello-component referenced in this MultiCluster application config
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_appconf_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "multicluster app config should be applied successfully")
		mcAppConfig, err := K8sClient.GetMultiClusterAppConfig(multiclusterTestNamespace, "mymcappconf")
		gomega.Expect(err).To(gomega.BeNil(), "multicluster app config mymcappconf should exist")
		gomega.Eventually(func() bool {
			return appConfigExistsWithFields(multiclusterTestNamespace, "mymcappconf", mcAppConfig)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})

func appConfigExistsWithFields(namespace string, name string, multiClusterAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration) bool {
	fmt.Printf("Looking for OAM app config %v/%v\n", namespace, name)
	appConfig, err := K8sClient.GetOAMAppConfig(namespace, name)
	if err != nil {
		return false
	}
	areEqual := true
	for i, expectedComp := range multiClusterAppConfig.Spec.Template.Spec.Components {
		areEqual = areEqual &&
			appConfig.Spec.Components[i].ComponentName == expectedComp.ComponentName
	}
	if !areEqual {
		fmt.Println("Retrieved app config spec doesn't match multi cluster app config spec")
		return false
	}
	return true
}

func loggingScopeExistsWithFields(namespace string, name string, mcLogScope *clustersv1alpha1.MultiClusterLoggingScope) bool {
	fmt.Printf("Looking for LoggingScope %v/%v\n", namespace, name)
	logScope, err := K8sClient.GetLoggingScope(namespace, name)
	return err == nil && reflect.DeepEqual(logScope.Spec, mcLogScope.Spec.Template.Spec)
}

func componentExistsWithFields(namespace string, name string, multiClusterComp *clustersv1alpha1.MultiClusterComponent) bool {
	fmt.Printf("Looking for OAM Component %v/%v\n", namespace, name)
	component, err := K8sClient.GetOAMComponent(namespace, name)
	if err != nil {
		return false
	}
	areEqual := reflect.DeepEqual(component.Spec.Parameters, multiClusterComp.Spec.Template.Spec.Parameters)
	if !areEqual {
		fmt.Println("Retrieved component parameters don't match multi cluster component parameters")
		return false
	}
	compWorkload, err := clusters.ReadContainerizedWorkload(component.Spec.Workload)
	if err != nil {
		fmt.Printf("Retrieved OAM component workload could not be read %v\n", err.Error())
		return false
	}
	mcCompWorkload, err := clusters.ReadContainerizedWorkload(multiClusterComp.Spec.Template.Spec.Workload)
	if err != nil {
		fmt.Printf("MultiClusterComponent workload could not be read: %v\n", err.Error())
	}

	if reflect.DeepEqual(compWorkload, mcCompWorkload) {
		return true
	}
	fmt.Println("MultiClusterComponent Workload does not match retrieved OAM Component Workload")
	return false
}

func secretExistsWithData(namespace, name string, secretData map[string][]byte) bool {
	fmt.Printf("Looking for Kubernetes secret %v/%v\n", namespace, name)
	secret, err := K8sClient.GetSecret(namespace, name)
	return err == nil && reflect.DeepEqual(secret.Data, secretData)
}

func configMapExistsMatchingMCConfigMap(namespace, name string, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) bool {
	fmt.Printf("Looking for Kubernetes ConfigMap %v/%v\n", namespace, name)
	configMap, err := K8sClient.GetConfigMap(namespace, name)
	return err == nil &&
		reflect.DeepEqual(configMap.Data, mcConfigMap.Spec.Template.Data) &&
		reflect.DeepEqual(configMap.BinaryData, mcConfigMap.Spec.Template.BinaryData)
}
