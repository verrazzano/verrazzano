// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/k8s"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/util"
)

const verrazzanoOperator = "verrazzano-application-operator"
const verrazzanoSystem = "verrazzano-system"

const appService = "hello-workload"
const appPodPrefix = "hello-workload"
const appDeployment = "hello-workload"
const appNamespace = "hello"

var fewSeconds = 2 * time.Second
var tenSeconds = 10 * time.Second
var thirtySeconds = 30 * time.Second
var threeMins = 3 * time.Minute
var fiveMins = 5 * time.Minute
var K8sClient k8s.Client

var _ = BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewClient()
	if err != nil {
		Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}
})

var _ = AfterSuite(func() {
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

	It(fmt.Sprintf("ServiceAccount %s exists", verrazzanoOperator), func() {
		Expect(K8sClient.DoesServiceAccountExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator service account should exist")
	})
	It(fmt.Sprintf("Deployment %s exists", verrazzanoOperator), func() {
		Expect(K8sClient.DoesDeploymentExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator doesn't exist")
	})
	It(fmt.Sprintf("Pod prefixed by %s exists", verrazzanoOperator), func() {
		Eventually(isOperatorRunning, threeMins).Should(BeTrue(),
			"The verrazzano operator pod is not urnning")
		Eventually(operatorServiceExists, threeMins).Should(BeTrue(),
			"The verrazzano operator service is not urnning")
	})
})

var _ = Describe("Testing hello app lifecycle", func() {
	//FLUENTD sidecar needs verrazzano secret
	It("verrazzano secret is created in verrazzano-system namespace", func() {
		command := fmt.Sprintf("create secret generic verrazzano -n verrazzano-system --from-literal=password=%s --from-literal=username=verrazzano", genPassword(10))
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
	It("application namespace is created", func() {
		command := fmt.Sprintf("create ns %s", appNamespace)
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
	It("apply component should result in a component in app namespace", func() {
		_, stderr := util.Kubectl("apply -f testdata/hello-comp.yaml")
		Expect(stderr).To(Equal(""))
		//	Eventually(appComponentExists, fewSeconds).Should(BeTrue())
	})
	It("apply loggingscope should result in a loggingscope in app namespace", func() {
		_, stderr := util.Kubectl("apply -f testdata/loggingscope.yaml")
		Expect(stderr).To(Equal(""))
		//	Eventually(appComponentExists, fewSeconds).Should(BeTrue())
	})
	It("apply app config should result in a app config in app namespace", func() {
		Eventually(createAppConfig, threeMins).Should(BeTrue())
		Eventually(appConfigExists, fewSeconds).Should(BeTrue())
	})
	It("hello deployment should be updated ", func() {
		Eventually(appDeploymentUpdated, tenSeconds).Should(BeTrue())
	})
	It("logging sidecar exists in app pod ", func() {
		Eventually(fluentdSidecarExists, fiveMins).Should(BeTrue())
	})
	It("hello service should exist ", func() {
		Eventually(appServiceExists, tenSeconds).Should(BeTrue(),
			"The hello service should exist")
	})

	It("update app config should result in a app config in app namespace", func() {
		Eventually(updateAppConfig, threeMins).Should(BeTrue())
		Eventually(appConfigExists, fewSeconds).Should(BeTrue())
	})
	It("hello deployment should be updated ", func() {
		Eventually(appDeploymentUpdated, fiveMins).Should(BeTrue())
	})
	It("logging sidecar exists in updated app pod ", func() {
		Eventually(fluentdSidecarExists, fiveMins).Should(BeTrue())
	})

	It("deleting app config", func() {
		Eventually(canDeleteAppConfig, fiveMins).Should(BeTrue())
	})
	It("deleting app component", func() {
		Eventually(canDeleteAppComponent, fiveMins).Should(BeTrue())
	})
	It("deleting app loggingscope", func() {
		Eventually(canDeleteAppLoggingScope, fiveMins).Should(BeTrue())
	})
	It("deleting verrazzano secret", func() {
		command := fmt.Sprintf("delete secret verrazzano -n verrazzano-system")
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
	It("hello deployment should  not exist ", func() {
		Eventually(appDeploymentExists, fiveMins).Should(BeFalse())
	})
	It("hello pod should not exist ", func() {
		Eventually(appPodExists, fiveMins).Should(BeFalse())
	})
	It("hello service should not exist ", func() {
		Eventually(doesServiceExist, fiveMins).Should(BeFalse())
	})
	It("application namespace is deleted", func() {
		command := fmt.Sprintf("delete ns %s", appNamespace)
		_, stderr := util.Kubectl(command)
		Expect(stderr).To(Equal(""))
	})
})

//// Helper functions
func appNsExists() bool {
	return K8sClient.DoesNamespaceExist(appNamespace)
}

func isOperatorRunning() bool {
	return K8sClient.IsPodRunning(verrazzanoOperator, verrazzanoSystem)
}

func operatorServiceExists() bool {
	return K8sClient.DoesServiceExist(verrazzanoOperator, verrazzanoSystem)
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

func fluentdSidecarExists() bool {
	return K8sClient.DoesContainerExist(appNamespace, appPodPrefix, "fluentd")
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

func canDeleteAppLoggingScope() bool {
	command := fmt.Sprintf("delete loggingscope -n %s hello-loggingscope", appNamespace)
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
		if rawTrait["apiVersion"] == v1alpha1.GroupVersion.String() &&
			rawTrait["kind"] == v1alpha1.MetricsTraitKind {
			return true
		}
	}
	return false
}

var passwordChars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func genPassword(passSize int) string {
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < passSize; i++ {
		b.WriteRune(passwordChars[rand.Intn(len(passwordChars))])
	}
	return b.String()
}
