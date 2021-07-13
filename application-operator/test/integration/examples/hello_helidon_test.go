// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package examples

import (
	"context"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"time"
)

var (
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 5 * time.Second
)

var _ = Describe("Hello Helidon Application Integration Test", func() {

	BeforeEach(func() {
	})

	AfterEach(func() {
	})

	Context("Application with configured ingress traits", func() {
		It("Should create the ingresses and associated certs/secrets correctly", func() {
			By("Creating the hello-helidon namespace successfully")
			helloNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hello-helidon",
					Labels: map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"},
				},
			}
			Expect(k8sClient.Create(context.Background(), &helloNamespace)).Should(Succeed())

			By("Defining the hello-helidon component successfully")
			helloDeploymentTemplate := v1alpha1.DeploymentTemplate{
				Metadata: metav1.ObjectMeta{
					Name: "hello-helidon-deployment",
				},
				PodSpec:  corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "hello-helidon-container",
							Image: "ghcr.io/verrazzano/example-helidon-greet-app-v1:0.1.12-1-20210409130027-707ecc4",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:      	   "http",
								},
							},
						},
					},
				},
			}
			helidonWorkload := v1alpha1.VerrazzanoHelidonWorkload{
				TypeMeta:   metav1.TypeMeta{
					Kind:       "VerrazzanoHelidonWorkload",
					APIVersion: "oam.verrazzano.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   "hello-helidon-workload",
					Labels: map[string]string{"app": "hello-helidon"},
				},
				Spec:       v1alpha1.VerrazzanoHelidonWorkloadSpec{DeploymentTemplate: helloDeploymentTemplate},
			}
			compSpec := v1alpha2.Component{
				TypeMeta:   metav1.TypeMeta{
					Kind:       "Component",
					APIVersion: "core.oam.dev/v1alpha2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-helidon-component",
					Namespace: "hello-helidon",
				},
				Spec:       v1alpha2.ComponentSpec{
					Workload: runtime.RawExtension{
						Object: &helidonWorkload,
					},
				},
			}
			scraperName := "verrazzano-system/vmi-system-prometheus-0"
			By("Defining the hello-helidon application configuration successfully")
			appSpec := v1alpha2.ApplicationConfiguration{
				TypeMeta:   metav1.TypeMeta{
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "hello-helidon-appconf",
					Namespace:   "hello-helidon",
					Annotations: map[string]string{"version": "v1.0.0", "description": "Hello Helidon Application"},
				},
				Spec:       v1alpha2.ApplicationConfigurationSpec{Components: []v1alpha2.ApplicationConfigurationComponent{
					{
						ComponentName: "hello-helidon-component",
						Traits: []v1alpha2.ComponentTrait{
							{
								Trait: runtime.RawExtension{
									Object: &v1alpha1.MetricsTrait{
										TypeMeta:   metav1.TypeMeta{
											Kind:       "MetricsTrait",
											APIVersion: "oam.verrazzano.io/v1alpha1",
										},
										Spec:       v1alpha1.MetricsTraitSpec{
											Scraper: &scraperName,
										},
									},
								},
							},
							{
								Trait: runtime.RawExtension{
									Object: &v1alpha1.IngressTrait{
										TypeMeta:   metav1.TypeMeta{
											Kind:       "IngressTrait",
											APIVersion: "oam.verrazzano.io/v1alpha1",
										},
										ObjectMeta: metav1.ObjectMeta{Name: "hello-helidon-ingress"},
										Spec:       v1alpha1.IngressTraitSpec{
											Rules: []v1alpha1.IngressRule{
												{
													Paths: []v1alpha1.IngressPath{
														{
															Path:     "/greet",
															PathType: "Prefix",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}},
			}
			By("Creating the component successfully")
			Expect(k8sClient.Create(context.Background(), &compSpec)).Should(Succeed())

			By("Creating the application successfully")
			Expect(k8sClient.Create(context.Background(), &appSpec)).Should(Succeed())

			By("Validating the existence of the application pods")
			Eventually(isHelloHelidonPodRunning, waitTimeout, pollingInterval).Should(BeTrue())

			By("Validating the existence of the service")
			Eventually(isHelloHelidonServiceReady, waitTimeout, pollingInterval).Should(BeTrue())

			By("Validating the existence of the certificate")
			Eventually(isHelloHelidonCertificateReady, waitTimeout, pollingInterval).Should(BeTrue())

			By("Validating the existence of the gateway")
			Eventually(isHelloHelidonGatewayReady, waitTimeout, pollingInterval).Should(BeTrue())

			By("Validating the existence of the virtual service")
			Eventually(isHelloHelidonVirtualServiceReady, waitTimeout, pollingInterval).Should(BeTrue())

			By("Deleting the application configuration successfully")
			Expect(k8sClient.Delete(context.Background(), &appSpec)).Should(Succeed())

			By("Deleting the component successfully")
			Expect(k8sClient.Delete(context.Background(), &compSpec)).Should(Succeed())

			By("Validating the removal of the gateway")
			Eventually(isHelloHelidonGatewayReady, waitTimeout, pollingInterval).Should(BeFalse())

			By("Validating the removal of the virtual service")
			Eventually(isHelloHelidonVirtualServiceReady, waitTimeout, pollingInterval).Should(BeFalse())

			Expect(k8sClient.Delete(context.Background(), &helloNamespace)).Should(Succeed())

			By("Validating the removal of the namespace")
			Eventually(func() bool {
				return doesNamespaceExist("hello-helidon")
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})

func isHelloHelidonPodRunning() bool {
	return pkg.PodsRunning("hello-helidon", expectedPodsHelloHelidon)
}

func isHelloHelidonServiceReady() bool {
	svc := corev1.Service{}
	name := types.NamespacedName{
		Namespace: "hello-helidon",
		Name:      "hello-helidon-deployment",
	}
	err := k8sClient.Get(context.TODO(), name, &svc)
	if err != nil {
		return false
	}
	if len(svc.Spec.Ports) > 0 {
		return svc.Spec.Ports[0].Port == 8080 && svc.Spec.Ports[0].TargetPort == intstr.FromInt(8080)
	}
	return false
}

func isHelloHelidonCertificateReady() bool {
	cert := certapiv1alpha2.Certificate{}
	name := types.NamespacedName{
		Namespace: "istio-system",
		Name:      "hello-helidon-hello-helidon-appconf-cert",
	}
	err := k8sClient.Get(context.TODO(), name, &cert)
	if err != nil {
		return false
	}
	return cert.Name == name.Name
}

func isHelloHelidonGatewayReady() bool {
	gateway := v1alpha3.Gateway{}
	name := types.NamespacedName{
		Namespace: "hello-helidon",
		Name:      "hello-helidon-hello-helidon-appconf-gw",
	}
	err := k8sClient.Get(context.TODO(), name, &gateway)
	if err != nil {
		return false
	}
	return gateway.Name == name.Name
}

func isHelloHelidonVirtualServiceReady() bool {
	vs := v1alpha3.VirtualService{}
	name := types.NamespacedName{
		Namespace: "hello-helidon",
		Name:      "hello-helidon-ingress-rule-0-vs",
	}
	err := k8sClient.Get(context.TODO(), name, &vs)
	if err != nil {
		return false
	}
	return vs.Name == name.Name
}

func doesNamespaceExist(namespace string) bool {
	ns := corev1.Namespace{}
	name := types.NamespacedName{
		Name:      namespace,
	}
	err := k8sClient.Get(context.TODO(), name, &ns)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
	}
	return ns.Status.Phase == corev1.NamespaceActive || ns.Status.Phase == corev1.NamespaceTerminating
}
