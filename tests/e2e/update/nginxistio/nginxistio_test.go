// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginxistio

import (
	"bytes"
	"context"
	"fmt"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
	"strings"
	"text/template"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/update"

	"github.com/verrazzano/verrazzano/pkg/constants"
)

const (
	nginxLabelKey            = "app.kubernetes.io/component"
	nginxLabelValue          = "controller"
	istioAppLabelKey         = "app"
	istioIngressLabelValue   = "istio-ingressgateway"
	istioEgressLabelValue    = "istio-egressgateway"
	nginxIngressServiceName  = "ingress-controller-ingress-nginx-controller"
	istioIngressServiceName  = "istio-ingressgateway"
	nginxExternalIPArg       = "controller.service.externalIPs"
	istioExternalIPArg       = "gateways.istio-ingressgateway.externalIPs"
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 10 * time.Second
	ociLBShapeAnnotation     = "service.beta.kubernetes.io/oci-load-balancer-shape"
	nginxLBShapeArg          = "controller.service.annotations.\"service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\""
	istioLBShapeArg          = "gateways.istio-ingressgateway.serviceAnnotations.\"service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\""
	nginxArgPrefixForAnno    = "controller.service.annotations."
	istioArgPrefixForAnno    = "gateways.istio-ingressgateway.serviceAnnotations."
	nginxTestAnnotationName  = "name-n"
	nginxTestAnnotationValue = "value-n"
	istioTestAnnotationName  = "name-i"
	istioTestAnnotationValue = "value-i"
	newReplicas              = 3
)

var testNginxIngressPorts = []corev1.ServicePort{
	{
		Name:     "https",
		Protocol: "TCP",
		Port:     443,
		NodePort: 31443,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: "https",
		},
	},
}

var testIstioIngressPorts = []corev1.ServicePort{
	{
		Name:       "https",
		Protocol:   "TCP",
		Port:       443,
		NodePort:   32443,
		TargetPort: intstr.FromInt(8443),
	},
}

type externalLBsTemplateData struct {
	ServerList string
}

type NginxAutoscalingIstioRelicasAffintyModifier struct {
	nginxReplicas        uint32
	istioIngressReplicas uint32
	istioEgressReplicas  uint32
}

type NginxIstioNodePortModifier struct {
	systemExternalLBIP      string
	applicationExternalLBIP string
}

type NginxIstioLoadBalancerModifier struct {
}

type NginxIstioIngressServiceAnnotationModifier struct {
	nginxLBShape string
	istioLBShape string
}

func (m NginxAutoscalingIstioRelicasAffintyModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Ingress == nil {
		cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	}
	if cr.Spec.Components.Istio == nil {
		cr.Spec.Components.Istio = &vzapi.IstioComponent{}
	}
	// update nginx
	//nginxInstallArgs := cr.Spec.Components.Ingress.NGINXInstallArgs
	//nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: "controller.autoscaling.enabled", Value: "true"})
	//nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: "controller.autoscaling.minReplicas", Value: fmt.Sprint(m.nginxReplicas)})
	overrides := []vzapi.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: []byte(fmt.Sprintf("{\"controller\": {\"autoscaling\": {\"enabled\": \"true\", \"minReplicas\": %v}}}", m.nginxReplicas)),
			},
		},
	}
	cr.Spec.Components.Ingress.ValueOverrides = overrides
	// update istio ingress
	if cr.Spec.Components.Istio.Ingress == nil {
		cr.Spec.Components.Istio.Ingress = &vzapi.IstioIngressSection{}
	}
	if cr.Spec.Components.Istio.Ingress.Kubernetes == nil {
		cr.Spec.Components.Istio.Ingress.Kubernetes = &vzapi.IstioKubernetesSection{}
	}
	cr.Spec.Components.Istio.Ingress.Kubernetes.Replicas = m.istioIngressReplicas
	// istio 1.11.4 has a bug handling this particular Affinity
	// it works fine if istio is installed with it
	// but it fails updating istio with it even though running pods has met replicaCount, istio is trying to schedule more
	// which results in pending pods
	//
	//if cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity == nil {
	//	cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity = &corev1.Affinity{}
	//}
	//if cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity == nil {
	//	cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	//}
	//requiredIngressAntiAffinity := cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	//requiredIngressAntiAffinity = append(requiredIngressAntiAffinity, corev1.PodAffinityTerm{
	//	LabelSelector: &metav1.LabelSelector{
	//		MatchLabels: nil,
	//		MatchExpressions: []metav1.LabelSelectorRequirement{
	//			{
	//				Key:      istioAppLabelKey,
	//				Operator: "In",
	//				Values: []string{
	//					istioIngressLabelValue,
	//				},
	//			},
	//		},
	//	},
	//	TopologyKey: "kubernetes.io/hostname",
	//})
	//cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = requiredIngressAntiAffinity
	// update istio ingress
	if cr.Spec.Components.Istio.Egress == nil {
		cr.Spec.Components.Istio.Egress = &vzapi.IstioEgressSection{}
	}
	if cr.Spec.Components.Istio.Egress.Kubernetes == nil {
		cr.Spec.Components.Istio.Egress.Kubernetes = &vzapi.IstioKubernetesSection{}
	}
	cr.Spec.Components.Istio.Egress.Kubernetes.Replicas = m.istioEgressReplicas
	// istio 1.11.4 has a bug handling this particular Affinity
	// it works fine if istio is installed with it
	// but it fails updating istio with it even though running pods has met replicaCount, istio is trying to schedule more
	// which results in pending pods
	//if cr.Spec.Components.Istio.Egress.Kubernetes.Affinity == nil {
	//	cr.Spec.Components.Istio.Egress.Kubernetes.Affinity = &corev1.Affinity{}
	//}
	//if cr.Spec.Components.Istio.Egress.Kubernetes.Affinity.PodAntiAffinity == nil {
	//	cr.Spec.Components.Istio.Egress.Kubernetes.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	//}
	//requiredEgressAntiAffinity := cr.Spec.Components.Istio.Egress.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	//requiredEgressAntiAffinity = append(requiredEgressAntiAffinity, corev1.PodAffinityTerm{
	//	LabelSelector: &metav1.LabelSelector{
	//		MatchLabels: nil,
	//		MatchExpressions: []metav1.LabelSelectorRequirement{
	//			{
	//				Key:      istioAppLabelKey,
	//				Operator: "In",
	//				Values: []string{
	//					istioEgressLabelValue,
	//				},
	//			},
	//		},
	//	},
	//	TopologyKey: "kubernetes.io/hostname",
	//})
	//cr.Spec.Components.Istio.Egress.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = requiredEgressAntiAffinity
}

func (u NginxIstioNodePortModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Ingress == nil {
		cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	}
	cr.Spec.Components.Ingress.Ports = testNginxIngressPorts
	cr.Spec.Components.Ingress.Type = vzapi.NodePort
	nginxInstallArgs := cr.Spec.Components.Ingress.NGINXInstallArgs
	nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: nginxExternalIPArg, ValueList: []string{u.systemExternalLBIP}})
	cr.Spec.Components.Ingress.NGINXInstallArgs = nginxInstallArgs
	if cr.Spec.Components.Istio == nil {
		cr.Spec.Components.Istio = &vzapi.IstioComponent{}
	}
	if cr.Spec.Components.Istio.Ingress == nil {
		cr.Spec.Components.Istio.Ingress = &vzapi.IstioIngressSection{}
	}
	cr.Spec.Components.Istio.Ingress.Ports = testIstioIngressPorts
	cr.Spec.Components.Istio.Ingress.Type = vzapi.NodePort
	istioInstallArgs := cr.Spec.Components.Istio.IstioInstallArgs
	istioInstallArgs = append(istioInstallArgs, vzapi.InstallArgs{Name: istioExternalIPArg, ValueList: []string{u.applicationExternalLBIP}})
	cr.Spec.Components.Istio.IstioInstallArgs = istioInstallArgs
}

func (u NginxIstioLoadBalancerModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Ingress == nil {
		cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	}
	cr.Spec.Components.Ingress.Type = vzapi.LoadBalancer
	var nginxInstallArgs []vzapi.InstallArgs
	for _, arg := range cr.Spec.Components.Ingress.NGINXInstallArgs {
		if arg.Name != nginxExternalIPArg {
			nginxInstallArgs = append(nginxInstallArgs, arg)
		}
	}
	cr.Spec.Components.Ingress.NGINXInstallArgs = nginxInstallArgs
	if cr.Spec.Components.Istio == nil {
		cr.Spec.Components.Istio = &vzapi.IstioComponent{}
	}
	if cr.Spec.Components.Istio.Ingress == nil {
		cr.Spec.Components.Istio.Ingress = &vzapi.IstioIngressSection{}
	}
	cr.Spec.Components.Istio.Ingress.Type = vzapi.LoadBalancer
	var istioInstallArgs []vzapi.InstallArgs
	for _, arg := range cr.Spec.Components.Istio.IstioInstallArgs {
		if arg.Name != istioExternalIPArg {
			istioInstallArgs = append(istioInstallArgs, arg)
		}
	}
	cr.Spec.Components.Istio.IstioInstallArgs = istioInstallArgs
}

func (u NginxIstioIngressServiceAnnotationModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Ingress == nil {
		cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	}
	cr.Spec.Components.Ingress.Type = vzapi.LoadBalancer
	nginxInstallArgs := cr.Spec.Components.Ingress.NGINXInstallArgs
	nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: nginxLBShapeArg, Value: u.nginxLBShape})
	nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: nginxArgPrefixForAnno + nginxTestAnnotationName, Value: nginxTestAnnotationValue})
	cr.Spec.Components.Ingress.NGINXInstallArgs = nginxInstallArgs
	if cr.Spec.Components.Istio == nil {
		cr.Spec.Components.Istio = &vzapi.IstioComponent{}
	}
	if cr.Spec.Components.Istio.Ingress == nil {
		cr.Spec.Components.Istio.Ingress = &vzapi.IstioIngressSection{}
	}
	cr.Spec.Components.Istio.Ingress.Type = vzapi.LoadBalancer
	istioInstallArgs := cr.Spec.Components.Istio.IstioInstallArgs
	istioInstallArgs = append(istioInstallArgs, vzapi.InstallArgs{Name: istioLBShapeArg, Value: u.istioLBShape})
	istioInstallArgs = append(istioInstallArgs, vzapi.InstallArgs{Name: istioArgPrefixForAnno + istioTestAnnotationName, Value: istioTestAnnotationValue})
	cr.Spec.Components.Istio.IstioInstallArgs = istioInstallArgs
}

var t = framework.NewTestFramework("update nginx-istio")

var systemExternalIP, applicationExternalIP string

var _ = t.BeforeSuite(func() {
	var err error
	systemExternalIP, applicationExternalIP, err = deployExternalLBs()
	if err != nil {
		Fail(err.Error())
	}
})

var _ = t.Describe("Update nginx-istio", Serial, Ordered, Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-nginx-istio verify", Label("f:platform-lcm.nginx-istio-verify"), func() {
		t.It("nginx-istio default replicas", func() {
			cr := update.GetCR()

			expectedIstioRunning := uint32(1)
			expectedNGINXRunning := uint32(1)
			if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
				expectedIstioRunning = 2
				expectedNGINXRunning = 2
			}
			update.ValidatePods(nginxLabelValue, nginxLabelKey, constants.IngressNamespace, expectedNGINXRunning, false)
			update.ValidatePods(istioIngressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, expectedIstioRunning, false)
			update.ValidatePods(istioEgressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, expectedIstioRunning, false)
		})
	})

	t.Describe("verrazzano-nginx-istio update ingress service annotations", Label("f:platform-lcm.nginx-istio-update-annotations"), func() {
		t.It("nginx-istio update ingress service annotations", func() {
			m := NginxIstioIngressServiceAnnotationModifier{nginxLBShape: "flexible", istioLBShape: "10Mbps"}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			validateServiceAnnotations(m)
		})
	})

	t.Describe("verrazzano-nginx-istio update replicas", Label("f:platform-lcm.nginx-istio-update-replicas"), func() {
		t.It("nginx-istio update replicas", func() {
			m := NginxAutoscalingIstioRelicasAffintyModifier{nginxReplicas: newReplicas, istioIngressReplicas: newReplicas, istioEgressReplicas: newReplicas}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			update.ValidatePods(nginxLabelValue, nginxLabelKey, constants.IngressNamespace, newReplicas, false)
			update.ValidatePods(istioIngressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, newReplicas, false)
			update.ValidatePods(istioEgressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, newReplicas, false)
		})
	})

	t.Describe("verrazzano-nginx-istio update nodeport", Label("f:platform-lcm.nginx-istio-update-nodeport"), func() {
		t.It("nginx-istio update ingress type to nodeport", func() {
			t.Logs.Infof("Update nginx/istio ingresses to use NodePort type with external load balancers: %s and %s", systemExternalIP, applicationExternalIP)
			m := NginxIstioNodePortModifier{systemExternalLBIP: systemExternalIP, applicationExternalLBIP: applicationExternalIP}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			t.Logs.Info("Validate nginx/istio ingresses for NodePort type and externalIPs")
			validateServiceNodePortAndExternalIP(systemExternalIP, applicationExternalIP)
		})
	})

	t.Describe("verrazzano-nginx-istio update loadbalancer", Label("f:platform-lcm.nginx-istio-update-loadbalancer"), func() {
		t.It("nginx-istio update ingress type to loadbalancer", func() {
			t.Logs.Infof("Update nginx/istio ingresses to use LoadBalancer type")
			m := NginxIstioLoadBalancerModifier{}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			t.Logs.Info("Validate nginx/istio ingresses for LoadBalancer type and loadBalancer IP")
			validateServiceLoadBalancer()
		})
	})

	t.Describe("verrazzano-nginx-istio update nodeport 2", Label("f:platform-lcm.nginx-istio-update-nodeport-2"), func() {
		t.It("nginx-istio update ingress type to nodeport 2", func() {
			t.Logs.Infof("Update nginx/istio ingresses to use NodePort type with external load balancers: %s and %s", systemExternalIP, applicationExternalIP)
			m := NginxIstioNodePortModifier{systemExternalLBIP: systemExternalIP, applicationExternalLBIP: applicationExternalIP}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			t.Logs.Info("Validate nginx/istio ingresses for NodePort type and externalIPs")
			validateServiceNodePortAndExternalIP(systemExternalIP, applicationExternalIP)
		})
	})
})

func deployExternalLBs() (string, string, error) {
	_, err := pkg.CreateNamespaceWithAnnotations("external-lb", map[string]string{}, map[string]string{})
	if err != nil {
		return "", "", err
	}

	systemServerList, applicationServerList, err := buildServerLists()
	if err != nil {
		return "", "", err
	}

	applyResource("testdata/external-lb/system-external-lb-cm.yaml", &externalLBsTemplateData{ServerList: systemServerList})
	applyResource("testdata/external-lb/system-external-lb.yaml", &externalLBsTemplateData{})
	applyResource("testdata/external-lb/system-external-lb-svc.yaml", &externalLBsTemplateData{})
	applyResource("testdata/external-lb/application-external-lb-cm.yaml", &externalLBsTemplateData{ServerList: applicationServerList})
	applyResource("testdata/external-lb/application-external-lb.yaml", &externalLBsTemplateData{})
	applyResource("testdata/external-lb/application-external-lb-svc.yaml", &externalLBsTemplateData{})

	sysIP, err := getServiceLoadBalancerIP("external-lb", "system-external-lb-svc")
	if err != nil {
		return "", "", err
	}

	appIP, err := getServiceLoadBalancerIP("external-lb", "application-external-lb-svc")
	if err != nil {
		return "", "", err
	}

	return sysIP, appIP, nil
}

func buildServerLists() (string, string, error) {
	nodes, err := pkg.ListNodes()
	if err != nil {
		return "", "", err
	}
	if len(nodes.Items) < 1 {
		return "", "", fmt.Errorf("can not find node in the cluster")
	}
	var serverListNginx, serverListIstio string
	for _, node := range nodes.Items {
		if len(node.Status.Addresses) < 1 {
			return "", "", fmt.Errorf("can not find address in the node")
		}
		serverListNginx = serverListNginx + fmt.Sprintf("           server %s:31443;\n", node.Status.Addresses[0].Address)
		serverListIstio = serverListIstio + fmt.Sprintf("           server %s:32443;\n", node.Status.Addresses[0].Address)
	}
	return serverListNginx, serverListIstio, nil
}

func applyResource(resourceFile string, templateData *externalLBsTemplateData) {
	file, err := pkg.FindTestDataFile(resourceFile)
	if err != nil {
		Fail(err.Error())
	}
	fileTemplate, err := template.ParseFiles(file)
	if err != nil {
		Fail(err.Error())
	}
	var buff bytes.Buffer
	err = fileTemplate.Execute(&buff, templateData)
	if err != nil {
		Fail(err.Error())
	}

	err = pkg.CreateOrUpdateResourceFromBytes(buff.Bytes())
	if err != nil {
		Fail(err.Error())
	}
}

func getServiceLoadBalancerIP(ns, svcName string) (string, error) {
	gomega.Eventually(func() error {
		svc, err := pkg.GetService(ns, svcName)
		if err != nil {
			return err
		}
		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			return fmt.Errorf("loadBalancer for service %s/%s is not ready yet", ns, svcName)
		}
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), fmt.Sprintf("Expected to get a loadBalancer for service %s/%s", ns, svcName))

	// Get the CR
	svc, err := pkg.GetService(ns, svcName)
	if err != nil {
		return "", fmt.Errorf("can not get IP for service %s/%s due to error: %v", ns, svcName, err.Error())
	}
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		return svc.Status.LoadBalancer.Ingress[0].IP, nil
	}

	return "", fmt.Errorf("no IP is found for service %s/%s", ns, svcName)
}

func validateServiceAnnotations(m NginxIstioIngressServiceAnnotationModifier) {
	gomega.Eventually(func() error {
		var err error
		nginxIngress, err := pkg.GetService(constants.IngressNamespace, nginxIngressServiceName)
		if err != nil {
			return err
		}
		if nginxIngress.Annotations[nginxTestAnnotationName] != nginxTestAnnotationValue {
			return fmt.Errorf("expect nginx ingress annotation %v with %v, but got %v", nginxTestAnnotationName, nginxTestAnnotationValue, nginxIngress.Annotations[nginxTestAnnotationName])
		}
		if nginxIngress.Annotations[ociLBShapeAnnotation] != m.nginxLBShape {
			return fmt.Errorf("expect nginx ingress annotation %v with value %v, but got %v", ociLBShapeAnnotation, m.nginxLBShape, nginxIngress.Annotations[ociLBShapeAnnotation])
		}
		istioIngress, err := pkg.GetService(constants.IstioSystemNamespace, istioIngressServiceName)
		if err != nil {
			return err
		}
		if istioIngress.Annotations[istioTestAnnotationName] != istioTestAnnotationValue {
			return fmt.Errorf("expect istio ingress annotation %v with %v, but got %v", istioTestAnnotationName, istioTestAnnotationValue, istioIngress.Annotations[istioTestAnnotationName])
		}
		if istioIngress.Annotations[ociLBShapeAnnotation] != m.istioLBShape {
			return fmt.Errorf("expect istio ingress annotation %v with value %v, but got %v", ociLBShapeAnnotation, m.istioLBShape, istioIngress.Annotations[ociLBShapeAnnotation])
		}
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "expect to get correct ports setting from nginx and istio services")
}

func validateServiceNodePortAndExternalIP(expectedSystemExternalIP, expectedApplicationExternalIP string) {
	gomega.Eventually(func() error {
		// validate Nginx Ingress service
		var err error
		nginxIngress, err := pkg.GetService(constants.IngressNamespace, nginxIngressServiceName)
		if err != nil {
			return err
		}
		if nginxIngress.Spec.Type != corev1.ServiceTypeNodePort {
			return fmt.Errorf("expect nginx ingress with type NodePort, but got %v", nginxIngress.Spec.Type)
		}
		if !reflect.DeepEqual(testNginxIngressPorts, nginxIngress.Spec.Ports) {
			return fmt.Errorf("expect nginx ingress with ports %v, but got %v", testNginxIngressPorts, nginxIngress.Spec.Ports)
		}
		expectedSysIPs := []string{expectedSystemExternalIP}
		if !reflect.DeepEqual(expectedSysIPs, nginxIngress.Spec.ExternalIPs) {
			return fmt.Errorf("expect nginx ingress with externalIPs %v, but got %v", expectedSysIPs, nginxIngress.Spec.ExternalIPs)
		}

		// validate Istio Ingress Service
		istioIngress, err := pkg.GetService(constants.IstioSystemNamespace, istioIngressServiceName)
		if err != nil {
			return err
		}
		if istioIngress.Spec.Type != corev1.ServiceTypeNodePort {
			return fmt.Errorf("expect istio ingress with type NodePort, but got %v", istioIngress.Spec.Type)
		}
		if !reflect.DeepEqual(testIstioIngressPorts, istioIngress.Spec.Ports) {
			return fmt.Errorf("expect istio ingress with ports %v, but got %v", testNginxIngressPorts, istioIngress.Spec.Ports)
		}
		expectedAppIPs := []string{expectedApplicationExternalIP}
		if !reflect.DeepEqual(expectedAppIPs, istioIngress.Spec.ExternalIPs) {
			return fmt.Errorf("expect istio ingress with externalIPs %v, but got %v", expectedAppIPs, istioIngress.Spec.ExternalIPs)
		}

		// validate Ingress Host
		err = validateIngressHost(expectedSystemExternalIP, "keycloak", "keycloak")
		if err != nil {
			return err
		}
		err = validateIngressHost(expectedSystemExternalIP, "verrazzano-ingress", "verrazzano-system")
		if err != nil {
			return err
		}

		// validate application Host
		err = validateApplicationHost(expectedApplicationExternalIP)
		if err != nil {
			return err
		}

		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "expect to get NodePort type and externalIPs from nginx and istio services")
}

func validateServiceLoadBalancer() {
	gomega.Eventually(func() error {
		// validate Nginx Ingress service
		var err error
		nginxIngress, err := pkg.GetService(constants.IngressNamespace, nginxIngressServiceName)
		if err != nil {
			return err
		}
		if nginxIngress.Spec.Type != corev1.ServiceTypeLoadBalancer {
			return fmt.Errorf("expect nginx ingress with type LoadBalancer, but got %v", nginxIngress.Spec.Type)
		}
		nginxLBIP, err := getServiceLoadBalancerIP(constants.IngressNamespace, nginxIngressServiceName)
		if err != nil {
			return err
		}
		if len(nginxLBIP) == 0 {
			return fmt.Errorf("invalid loadBalancer IP %s for nginx", nginxLBIP)
		}

		// validate Istio Ingress Service
		istioIngress, err := pkg.GetService(constants.IstioSystemNamespace, istioIngressServiceName)
		if err != nil {
			return err
		}
		if istioIngress.Spec.Type != corev1.ServiceTypeLoadBalancer {
			return fmt.Errorf("expect istio ingress with type LoadBalancer, but got %v", istioIngress.Spec.Type)
		}
		istioLBIP, err := getServiceLoadBalancerIP(constants.IstioSystemNamespace, istioIngressServiceName)
		if err != nil {
			return err
		}
		if len(istioLBIP) == 0 {
			return fmt.Errorf("invalid loadBalancer IP %s for istio", istioLBIP)
		}

		// validate Ingress Host
		err = validateIngressHost(nginxLBIP, "keycloak", "keycloak")
		if err != nil {
			return err
		}
		err = validateIngressHost(nginxLBIP, "verrazzano-ingress", "verrazzano-system")
		if err != nil {
			return err
		}

		// validate application Host
		err = validateApplicationHost(istioLBIP)
		if err != nil {
			return err
		}

		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "expect to get LoadBalancer type and loadBalancer IP from nginx and istio services")
}

func validateIngressHost(expectedIP, ingressName, ns string) error {
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	clientset, err := pkg.GetKubernetesClientsetForCluster(kubeConfigPath)
	if err != nil {
		return err
	}
	ingress, err := clientset.NetworkingV1().Ingresses(ns).Get(context.TODO(), ingressName, v1.GetOptions{})
	if err != nil {
		return err
	}
	if len(ingress.Spec.Rules) == 0 {
		return fmt.Errorf("expect Ingress %s/%s to have at least one host", ns, ingressName)
	}
	host := ingress.Spec.Rules[0].Host
	if !strings.Contains(host, expectedIP) {
		return fmt.Errorf("expect Ingress %s/%s Host %s to contain IP %s", ns, ingressName, host, expectedIP)
	}
	return nil
}

func validateApplicationHost(expectedIP string) error {
	host, err := k8sutil.GetHostnameFromGateway("hello-helidon", "")
	if err != nil {
		return err
	}
	if !strings.Contains(host, expectedIP) {
		return fmt.Errorf("expect hello-helidon HOST %s to contain IP %s", host, expectedIP)
	}
	return nil
}
