// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/k8s"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/util"
)

const verrazzanoAppOperator = "verrazzano-application-operator"
const verrazzanoSystem = "verrazzano-system"

const appService = "hello-workload"
const appPodPrefix = "hello-workload"
const appDeployment = "hello-workload"
const appNamespace = "hello"
const appLoggingScopeSecret = "log-scope-secret"

var for2s = 2 * time.Second
var for10s = 10 * time.Second
var for3m = 3 * time.Minute
var for5m = 5 * time.Minute
var K8sClient k8s.Client

var _ = BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewClient()
	if err != nil {
		Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}

	// Do set up for multi cluster tests
	setupMultiClusterTest()

})

var _ = AfterSuite(func() {
	// ensuring that this code runs after the other tests.  Multiple MC resources exist in the namespace
	// as well as their associated wrapped resources.  This is the only location where this cleanup is guaranteed to
	// be executed post all the resource deployments.
	_, stderr := util.Kubectl("delete ns multiclustertest")
	Expect(stderr).To(Equal(""), "kubectl namespace deletion completed")
	Eventually(func() bool {
		return !K8sClient.DoesNamespaceExist("multiclustertest")
	}, timeout, pollInterval).Should(BeTrue())
})

var _ = Describe("Custom Resource Definition for OAM controller runtime", func() {
	It("applicationconfigurations.core.oam.dev exists", func() {
		Expect(K8sClient.DoesCRDExist("applicationconfigurations.core.oam.dev")).To(BeTrue(),
			"The applicationconfigurations.core.oam.dev CRD should exist")
	})
	It("components.core.oam.dev exists", func() {
		Expect(K8sClient.DoesCRDExist("components.core.oam.dev")).To(BeTrue(),
			"The components.core.oam.dev CRD should exist")
	})
	It("containerizedworkloads.core.oam.dev  exists", func() {
		Expect(K8sClient.DoesCRDExist("containerizedworkloads.core.oam.dev")).To(BeTrue(),
			"The containerizedworkloads.core.oam.dev  CRD should exist")
	})
	It("healthscopes.core.oam.dev exists", func() {
		Expect(K8sClient.DoesCRDExist("healthscopes.core.oam.dev")).To(BeTrue(),
			"The healthscopes.core.oam.dev CRD should exist")
	})
	It("ingresstraits.oam.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("ingresstraits.oam.verrazzano.io")).To(BeTrue(),
			"The ingresstraits.oam.verrazzano.io  CRD should exist")
	})
	It("manualscalertraits.core.oam.dev exists", func() {
		Expect(K8sClient.DoesCRDExist("manualscalertraits.core.oam.dev")).To(BeTrue(),
			"The manualscalertraits.core.oam.dev  CRD should exist")
	})
	It("scopedefinitions.core.oam.dev exists", func() {
		Expect(K8sClient.DoesCRDExist("scopedefinitions.core.oam.dev")).To(BeTrue(),
			"The scopedefinitions.core.oam.dev  CRD should exist")
	})
})

var _ = Describe("Custom Resource Definition for Verrazzano CRDs", func() {
	It("ingresstraits.oam.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("ingresstraits.oam.verrazzano.io")).To(BeTrue(),
			"The ingresstraits.oam.verrazzano.io CRD should exist")
	})
})

var _ = Describe("verrazzano-application namespace resources ", func() {
	It(fmt.Sprintf("Namespace %s exists", verrazzanoSystem), func() {
		Expect(K8sClient.DoesNamespaceExist(verrazzanoSystem)).To(BeTrue(),
			"The namespace should exist")
	})

	It(fmt.Sprintf("ServiceAccount %s exists", verrazzanoAppOperator), func() {
		Expect(K8sClient.DoesServiceAccountExist(verrazzanoAppOperator, verrazzanoSystem)).To(BeTrue(),
			"The Verrazzano operator service account should exist")
	})
	It(fmt.Sprintf("Deployment %s exists", verrazzanoAppOperator), func() {
		Expect(K8sClient.DoesDeploymentExist(verrazzanoAppOperator, verrazzanoSystem)).To(BeTrue(),
			"The Verrazzano operator doesn't exist")
	})
	It(fmt.Sprintf("Pod prefixed by %s exists", verrazzanoAppOperator), func() {
		Eventually(isOperatorRunning, for3m).Should(BeTrue(),
			"The Verrazzano operator pod is not running")
		Eventually(operatorServiceExists, for3m).Should(BeTrue(),
			"The Verrazzano operator service is not running")
	})
})

var _ = Describe("Testing hello app lifecycle", func() {
	It("application namespace is created", func() {
		command := fmt.Sprintf("create ns %s", appNamespace)
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
	//FLUENTD sidecar needs app's explicit logging scope secret to be present in app NS
	It("Explicit logging scope secret is manually created in application namespace", func() {
		password, err := genPassword(10)
		Expect(err).To(BeNil(), "Expected password generation to succeed.")
		command := fmt.Sprintf("create secret generic %s -n %s --from-literal=password=%s --from-literal=username=someUser",
			appLoggingScopeSecret, appNamespace, password)
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
	It("apply component should result in a component in app namespace", func() {
		_, stderr := util.Kubectl("apply -f testdata/hello-comp.yaml")
		Expect(stderr).To(Equal(""))
		//	Eventually(appComponentExists, for2s).Should(BeTrue())
	})
	It("apply app config should result in a app config in app namespace", func() {
		Eventually(createAppConfig, for3m).Should(BeTrue())
		Eventually(appConfigExists, for2s).Should(BeTrue())
	})
	It("hello deployment should be updated ", func() {
		Eventually(appDeploymentUpdated, for10s).Should(BeTrue())
	})
	It("hello service should exist ", func() {
		Eventually(appServiceExists, for10s).Should(BeTrue(),
			"The hello service should exist")
	})
	It("update app config should result in a app config in app namespace", func() {
		Eventually(updateAppConfig, for3m).Should(BeTrue())
		Eventually(appConfigExists, for2s).Should(BeTrue())
	})
	It("hello deployment should be updated ", func() {
		Eventually(appDeploymentUpdated, for5m).Should(BeTrue())
	})

	It("deleting app config", func() {
		Eventually(canDeleteAppConfig, for5m).Should(BeTrue())
	})
	It("deleting app component", func() {
		Eventually(canDeleteAppComponent, for5m).Should(BeTrue())
	})
	It("deleting logging scope secret", func() {
		command := fmt.Sprintf("delete secret %s -n %s", appLoggingScopeSecret, appNamespace)
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
	It("hello deployment should  not exist ", func() {
		Eventually(appDeploymentExists, for5m).Should(BeFalse())
	})
	It("hello pod should not exist ", func() {
		Eventually(appPodExists, for5m).Should(BeFalse())
	})
	It("hello service should not exist ", func() {
		Eventually(doesServiceExist, for5m).Should(BeFalse())
	})
	It("application namespace is deleted", func() {
		command := fmt.Sprintf("delete ns %s", appNamespace)
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
})

// // Helper functions
func isOperatorRunning() bool {
	return K8sClient.IsPodRunning(verrazzanoAppOperator, verrazzanoSystem)
}

func operatorServiceExists() bool {
	return K8sClient.DoesServiceExist(verrazzanoAppOperator, verrazzanoSystem)
}

func createAppConfig() bool {
	_, stderr := util.Kubectl("apply -f testdata/hello-app-v0.yaml")
	return stderr == ""
}

func updateAppConfig() bool {
	_, stderr := util.Kubectl("apply -f testdata/hello-app-v1.yaml")
	return stderr == ""
}

func appDeploymentExists() bool {
	return K8sClient.DoesDeploymentExist(appDeployment, appNamespace)
}

func appDeploymentUpdated() bool {
	return K8sClient.IsDeploymentUpdated(appDeployment, appNamespace)
}

func appPodExists() bool {
	return K8sClient.DoesPodExist(appPodPrefix, appNamespace)
}

func appServiceExists() bool {
	return K8sClient.DoesServiceExist(appService, appNamespace)
}

func canDeleteAppConfig() bool {
	command := fmt.Sprintf("delete appconfig -n %s hello-app", appNamespace)
	_, stderr := util.Kubectl(command)
	return stderr == ""
}

func canDeleteAppComponent() bool {
	command := fmt.Sprintf("delete component -n %s hello-component", appNamespace)
	_, stderr := util.Kubectl(command)
	return stderr == ""
}

func doesServiceExist() bool {
	return K8sClient.DoesServiceExist(appService, appNamespace)
}

func appConfigExists() bool {
	appConfig, err := K8sClient.GetAppConfig(appNamespace, "hello-app")
	if err != nil || appConfig == nil || len(appConfig.Spec.Components) == 0 {
		return false
	}
	for _, trait := range appConfig.Spec.Components[0].Traits {
		var rawTrait map[string]interface{}
		json.Unmarshal(trait.Trait.Raw, &rawTrait)
		if rawTrait["apiVersion"] == v1alpha1.SchemeGroupVersion.String() &&
			rawTrait["kind"] == v1alpha1.MetricsTraitKind {
			return true
		}
	}
	return false
}

var passwordChars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// genPassword generates a password string of a specific size.
func genPassword(passSize int) (string, error) {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i := 0; i < passSize; i++ {
		b.WriteRune(passwordChars[nBig.Int64()])
	}
	return b.String(), nil
}
