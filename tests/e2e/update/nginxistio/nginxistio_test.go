// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginxistio

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"
	"time"

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
	nginxLabelKey          = "app.kubernetes.io/component"
	nginxLabelValue        = "controller"
	istioAppLabelKey       = "app"
	istioIngressLabelValue = "istio-ingressgateway"
	istioEgressLabelValue  = "istio-egressgateway"
	waitTimeout            = 5 * time.Minute
	pollingInterval        = 5 * time.Second
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

type NginxIstioServicePortsModifier struct {
}

func (m NginxAutoscalingIstioRelicasAffintyModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Ingress == nil {
		cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	}
	if cr.Spec.Components.Istio == nil {
		cr.Spec.Components.Istio = &vzapi.IstioComponent{}
	}
	// update nginx
	nginxInstallArgs := cr.Spec.Components.Ingress.NGINXInstallArgs
	nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: "controller.autoscaling.enabled", Value: "true"})
	nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: "controller.autoscaling.minReplicas", Value: fmt.Sprint(m.nginxReplicas)})
	cr.Spec.Components.Ingress.NGINXInstallArgs = nginxInstallArgs
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
	nginxInstallArgs = append(nginxInstallArgs, vzapi.InstallArgs{Name: "controller.service.externalIPs", ValueList: []string{u.systemExternalLBIP}})
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
	istioInstallArgs = append(istioInstallArgs, vzapi.InstallArgs{Name: "gateways.istio-ingressgateway.externalIPs", ValueList: []string{u.applicationExternalLBIP}})
	cr.Spec.Components.Istio.IstioInstallArgs = istioInstallArgs
}

func (u NginxIstioServicePortsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Ingress == nil {
		cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	}
	cr.Spec.Components.Ingress.Ports = testNginxIngressPorts
	if cr.Spec.Components.Istio == nil {
		cr.Spec.Components.Istio = &vzapi.IstioComponent{}
	}
	if cr.Spec.Components.Istio.Ingress == nil {
		cr.Spec.Components.Istio.Ingress = &vzapi.IstioIngressSection{}
	}
	cr.Spec.Components.Istio.Ingress.Ports = testIstioIngressPorts
}

var t = framework.NewTestFramework("update nginx-istio")

var nodeCount uint32

var _ = t.BeforeSuite(func() {
	var err error
	nodeCount, err = pkg.GetNodeCount()
	if err != nil {
		Fail(err.Error())
	}
})

var _ = t.Describe("Update nginx-istio", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-nginx-istio verify", Label("f:platform-lcm.nginx-istio-verify"), func() {
		t.It("nginx-istio default replicas", func() {
			cr := update.GetCR()

			expectedIstioRunning := uint32(1)
			if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
				expectedIstioRunning = 2
			}
			update.ValidatePods(nginxLabelValue, nginxLabelKey, constants.IngressNamespace, uint32(1), false)
			update.ValidatePods(istioIngressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, expectedIstioRunning, false)
			update.ValidatePods(istioEgressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, expectedIstioRunning, false)
		})
	})

	t.Describe("verrazzano-nginx-istio update replicas", Label("f:platform-lcm.nginx-istio-update-replicas"), func() {
		t.It("nginx-istio update replicas", func() {
			istioCount := nodeCount - 1
			if nodeCount == 1 {
				istioCount = nodeCount
			}
			m := NginxAutoscalingIstioRelicasAffintyModifier{nginxReplicas: nodeCount, istioIngressReplicas: istioCount, istioEgressReplicas: istioCount}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			update.ValidatePods(nginxLabelValue, nginxLabelKey, constants.IngressNamespace, nodeCount, false)
			update.ValidatePods(istioIngressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, istioCount, false)
			update.ValidatePods(istioEgressLabelValue, istioAppLabelKey, constants.IstioSystemNamespace, istioCount, false)
		})
	})

	t.Describe("verrazzano-nginx-istio update service ports", Label("f:platform-lcm.nginx-istio-update-ports"), func() {
		t.It("nginx-istio update service ports", func() {
			m := NginxIstioServicePortsModifier{}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			validateServicePorts()
		})
	})

	t.Describe("verrazzano-nginx-istio update nodeport", Label("f:platform-lcm.nginx-istio-update-nodeport"), func() {
		t.It("nginx-istio update ingress type", func() {
			t.Logs.Info("Create external load balancers")
			sysIP, appIP, err := deployExternalLBs()
			if err != nil {
				Fail(err.Error())
			}

			t.Logs.Infof("Update nginx/istio ingresses to use NodePort with external load balancers: %s and %s", sysIP, appIP)
			m := NginxIstioNodePortModifier{systemExternalLBIP: sysIP, applicationExternalLBIP: appIP}
			err = update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			t.Logs.Info("Validate nginx/istio ingresses for NodePort and externalIPs")
			validateServiceTypeAndExternalIP(sysIP, appIP)
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

func validateServicePorts() {
	gomega.Eventually(func() error {
		var err error
		nginxIngress, err := pkg.GetService(constants.IngressNamespace, "ingress-controller-ingress-nginx-controller")
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(testNginxIngressPorts, nginxIngress.Spec.Ports) {
			return fmt.Errorf("expect nginx ingress with ports %v, but got %v", testNginxIngressPorts, nginxIngress.Spec.Ports)
		}
		istioIngress, err := pkg.GetService(constants.IstioSystemNamespace, "istio-ingressgateway")
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(testIstioIngressPorts, istioIngress.Spec.Ports) {
			return fmt.Errorf("expect istio ingress with ports %v, but got %v", testNginxIngressPorts, istioIngress.Spec.Ports)
		}
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "expect to get correct ports setting from nginx and istio services")
}

func validateServiceTypeAndExternalIP(sysIP, appIP string) {
	gomega.Eventually(func() error {
		var err error
		nginxIngress, err := pkg.GetService(constants.IngressNamespace, "ingress-controller-ingress-nginx-controller")
		if err != nil {
			return err
		}
		if nginxIngress.Spec.Type != corev1.ServiceTypeNodePort {
			return fmt.Errorf("expect nginx ingress with type NodePort, but got %v", nginxIngress.Spec.Type)
		}
		expectedSysIPs := []string{sysIP}
		if !reflect.DeepEqual(expectedSysIPs, nginxIngress.Spec.ExternalIPs) {
			return fmt.Errorf("expect nginx ingress with externalIPs %v, but got %v", expectedSysIPs, nginxIngress.Spec.ExternalIPs)
		}
		istioIngress, err := pkg.GetService(constants.IstioSystemNamespace, "istio-ingressgateway")
		if err != nil {
			return err
		}
		if istioIngress.Spec.Type != corev1.ServiceTypeNodePort {
			return fmt.Errorf("expect istio ingress with type NodePort, but got %v", istioIngress.Spec.Type)
		}
		expectedAppIPs := []string{appIP}
		if !reflect.DeepEqual(expectedAppIPs, istioIngress.Spec.ExternalIPs) {
			return fmt.Errorf("expect nginx ingress with externalIPs %v, but got %v", expectedAppIPs, istioIngress.Spec.ExternalIPs)
		}
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "expect to get correct type and externalIPs from nginx and istio services")
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
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get a loadBalancer for service")

	// Get the CR
	svc, err := pkg.GetService(ns, svcName)
	if err != nil {
		return "", err
	}
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		return svc.Status.LoadBalancer.Ingress[0].IP, nil
	}

	return "", fmt.Errorf("no IP is found for service %s/%s", ns, svcName)
}
