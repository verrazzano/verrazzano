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
	timeout                   = 5 * time.Second
	pollInterval              = 20 * time.Millisecond
)

var (
	multiclusterCrds = []string{
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusternamespaces.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclustersecrets.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterconfigmaps.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclustercomponents.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterapplicationconfigurations.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterloggingscopes.yaml", crdDir),
	}
)

var _ = ginkgo.Describe("Testing Multi-Cluster CRDs", func() {
	ginkgo.It("MultiCluster CRDs can be applied", func() {
		for _, crd := range multiclusterCrds {
			_, stderr := util.Kubectl(fmt.Sprintf("apply -f %v", crd))
			gomega.Expect(stderr).To(gomega.Equal(""), fmt.Sprintf("Failed to apply CRD %v", crd))
		}
		_, stderr := util.Kubectl("create ns " + multiclusterTestNamespace)
		gomega.Expect(stderr).To(gomega.Equal(""), fmt.Sprintf("failed to create namespace %v", multiclusterTestNamespace))
	})
	ginkgo.It("MultiClusterNamespace can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_namespace_sample.yaml")
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
	ginkgo.It("MultiClusterConfigMap can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_configmap_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
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
	ginkgo.It("MultiClusterApplicationConfiguration can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_appconf_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
	ginkgo.It("MultiClusterLoggingScope can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/multicluster_loggingscope_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
})

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

func secretExistsWithData(namespace string, name string, secretData map[string][]byte) bool {
	fmt.Printf("Looking for Kubernetes secret %v/%v\n", namespace, name)
	secret, err := K8sClient.GetSecret(multiclusterTestNamespace, "mymcsecret")
	return err == nil && reflect.DeepEqual(secret.Data, secretData)
}
