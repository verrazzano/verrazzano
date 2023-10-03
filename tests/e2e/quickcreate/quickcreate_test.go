// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package quickcreate

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	minimumVersion  = "1.7.0"
	waitTimeOut     = 30 * time.Minute
	pollingInterval = 30 * time.Second
)

var (
	client clipkg.Client
	ctx    *QCContext
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	// Get Kubeconfig information and create clients
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).To(BeNil())
	cfg, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	Expect(err).To(BeNil())
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	c, err := clipkg.New(cfg, clipkg.Options{
		Scheme: scheme,
	})
	Expect(err).To(BeNil())
	client = c

	// Create test context and setup
	ctx, err = newContext(client, clusterType)
	Expect(err).To(BeNil())
	err = ctx.setup()
	Expect(err).To(BeNil())

	t.Logs.Infof("Creating Cluster of type [%s]", ctx.ClusterType)
})
var afterSuite = t.AfterSuiteFunc(func() {
	if ctx == nil {
		return
	}
	Eventually(func() error {
		err := ctx.cleanupCAPICluster()
		if err != nil {
			t.Logs.Info(err)
		}
		return err
	}).WithPolling(pollingInterval).WithTimeout(waitTimeOut).ShouldNot(HaveOccurred())
	Eventually(func() error {
		err := ctx.deleteObject(ctx.namespaceObject())
		if err != nil {
			t.Logs.Info(err)
		}
		return err
	}).WithPolling(pollingInterval).WithTimeout(waitTimeOut).ShouldNot(HaveOccurred())
})
var _ = BeforeSuite(beforeSuite)
var _ = AfterSuite(afterSuite)

var _ = t.Describe("using the quick create api", func() {
	t.Context("with a kubeconfig", func() {
		kcpath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			t.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		}
		t.ItMinimumVersion("creates a usuable cluster", minimumVersion, kcpath, createCluster)
	})
})

func createCluster() {
	err := ctx.applyCluster()
	Expect(err).To(BeNil())
	Eventually(func() error {
		err := ctx.isClusterReady()
		if err != nil {
			t.Logs.Info(err)
		}
		return err
	}).WithPolling(pollingInterval).WithTimeout(waitTimeOut).ShouldNot(HaveOccurred())
}
