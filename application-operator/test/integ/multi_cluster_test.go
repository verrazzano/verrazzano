// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/util"
)

const (
	multiclusterTestNamespace = "multiclustertest"
	managedClusterName        = "managed1"
	crdDir                    = "../../config/crd/bases"
	timeout                   = 2 * time.Minute
	pollInterval              = 40 * time.Millisecond
	applicationOperator       = "verrazzano-application-operator"
	duration                  = 1 * time.Minute
)

var (
	multiclusterCrds = []string{
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
	ginkgo.It("Apply MultiClusterSecret with 2 placements remains pending", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_secret_2placements.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcsecret, err := K8sClient.GetMultiClusterSecret(multiclusterTestNamespace, "mymcsecret2")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return secretExistsWithData(multiclusterTestNamespace, "mymcsecret2", mcsecret.Spec.Template.Data)
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			// Verify we have the expected status update
			mcRetrievedSecret, err := K8sClient.GetMultiClusterSecret(multiclusterTestNamespace, "mymcsecret2")
			return err == nil && mcRetrievedSecret.Status.State == clustersv1alpha1.Pending &&
				isStatusAsExpected(mcRetrievedSecret.Status, clustersv1alpha1.DeployComplete, "created", clustersv1alpha1.Succeeded, "managed1")
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
		gomega.Eventually(func() bool {
			// Verify we have the expected status update
			mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "mymcconfigmap")
			return err == nil && isStatusAsExpected(mcConfigMap.Status, clustersv1alpha1.DeployComplete, "created", clustersv1alpha1.Succeeded, managedClusterName)
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
		gomega.Consistently(func() bool {
			// Verify the controller is not updating the status more than once with the failure,
			// and is adding exactly one cluster level status entry
			mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "invalid-mccm")
			return err == nil && isStatusAsExpected(mcConfigMap.Status, clustersv1alpha1.DeployFailed, "", clustersv1alpha1.Failed, managedClusterName)
		}, duration, pollInterval).Should(gomega.BeTrue())
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

var _ = ginkgo.Describe("Testing VerrazzanoProject validation", func() {
	ginkgo.It("VerrazzanoProject invalid namespace ", func() {
		// Apply VerrazzanoProject resource and expect to fail due to invalid namespace
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_invalid_namespace.yaml")
		gomega.Expect(stderr).To(gomega.ContainSubstring(fmt.Sprintf("Namespace for the resource must be %q", constants.VerrazzanoMultiClusterNamespace)))
	})
	ginkgo.It("VerrazzanoProject invalid namespaces list", func() {
		// Apply VerrazzanoProject resource and expect to fail due to invalid namespaces list
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_invalid_namespaces_list.yaml")
		gomega.Expect(stderr).To(gomega.ContainSubstring("missing required field \"namespaces\""))
	})
})

var _ = ginkgo.Describe("Testing VerrazzanoProject namespace generation", func() {
	ginkgo.It("Apply VerrazzanoProject with default namespace labels", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_namespace_default_labels.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be created successfully")
		gomega.Eventually(func() bool {
			namespace, err := K8sClient.GetNamespace("test-namespace-1")
			return appConfigExistsWithFields(multiclusterTestNamespace, "mymcappconf", mcAppConfig)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply VerrazzanoProject with default namespace labels", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_namespace_override_labels.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be updated successfully")
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
	compWorkload, err := clusterstest.ReadContainerizedWorkload(component.Spec.Workload)
	if err != nil {
		fmt.Printf("Retrieved OAM component workload could not be read %v\n", err.Error())
		return false
	}
	mcCompWorkload, err := clusterstest.ReadContainerizedWorkload(multiClusterComp.Spec.Template.Spec.Workload)
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

func createManagedClusterSecret() {
	createSecret := fmt.Sprintf(
		"create secret generic %s --from-literal=%s=%s -n %s",
		constants.MCRegistrationSecret,
		constants.ClusterNameData,
		managedClusterName,
		constants.VerrazzanoSystemNamespace)

	_, stderr := util.Kubectl(createSecret)
	if stderr != "" {
		ginkgo.Fail(fmt.Sprintf("failed to create secret %v: %v", constants.MCRegistrationSecret, stderr))
	}
}

func isStatusAsExpected(status clustersv1alpha1.MultiClusterResourceStatus,
	expectedConditionType clustersv1alpha1.ConditionType, conditionMsgContains string,
	expectedClusterState clustersv1alpha1.StateType,
	expectedClusterName string) bool {
	matchingConditionCount := 0
	matchingClusterStatusCount := 0
	for _, condition := range status.Conditions {
		if condition.Type == expectedConditionType && strings.Contains(condition.Message, conditionMsgContains) {
			matchingConditionCount++
		}
	}
	for _, clusterStatus := range status.Clusters {
		if clusterStatus.State == expectedClusterState &&
			clusterStatus.Name == expectedClusterName &&
			clusterStatus.LastUpdateTime != "" {
			matchingClusterStatusCount++
		}
	}
	return matchingConditionCount == 1 && matchingClusterStatusCount == 1
}

func setupMultiClusterTest() {
	isPodRunningYet := func() bool {
		return K8sClient.IsPodRunning(applicationOperator, constants.VerrazzanoSystemNamespace)
	}
	gomega.Eventually(isPodRunningYet, "2m", "5s").Should(gomega.BeTrue(),
		fmt.Sprintf("The %s pod should be in the Running state", constants.VerrazzanoSystemNamespace))

	_, stderr := util.Kubectl("create ns " + constants.VerrazzanoMultiClusterNamespace)
	if stderr != "" {
		ginkgo.Fail(fmt.Sprintf("failed to create namespace %v", constants.VerrazzanoMultiClusterNamespace))
	}

	_, stderr = util.Kubectl("create ns " + multiclusterTestNamespace)
	if stderr != "" {
		ginkgo.Fail(fmt.Sprintf("failed to create namespace %v", multiclusterTestNamespace))
	}

	createManagedClusterSecret()
}
