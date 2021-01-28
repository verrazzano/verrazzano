package pkg

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"strings"
	yourl "net/url"

	corev1 "k8s.io/api/core/v1"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"

)

const istioSystemNamespace string = "istio-system"

func GetKindIngress() string {
	fmt.Println("Obtaining KIND control plane address info ...")
	addrHost := ""
	var addrPort int32

	pods := ListPods(istioSystemNamespace)
	//pods, _ := clientset.CoreV1().Pods(istioSystemNamespace).List(context.TODO, metav1.ListOptions{})
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, "istio-ingressgateway-") {
			addrHost = pods.Items[i].Status.HostIP
		}
	}

	ingressgateway := findIstioIngressGatewaySvc(false)
	fmt.Println("ingressgateway for KIND cluster is ", ingressgateway)
	for _, eachPort := range ingressgateway.Spec.Ports {
		if eachPort.Port == 80 {
			fmt.Printf("KIND cluster - found ingressgateway port %d with nodeport %d, name %s\n", eachPort.Port, eachPort.NodePort, eachPort.Name)
			addrPort = eachPort.NodePort
		}
	}

	if addrHost == "" {
		fmt.Println("KIND control plane address is empty")
		return ""
	} else {
		ingressAddr := fmt.Sprintf("%s:%d", addrHost, addrPort)
		fmt.Printf("KIND ingress address is %s\n", ingressAddr)
		return ingressAddr
	}
}

func findIstioIngressGatewaySvc(requireLoadBalancer bool) corev1.Service {
	svcList := ListServices(istioSystemNamespace)
	//svcList, _ := clientset.Namespace(istioSystemNamespace).ListServices()
	var ingressgateway corev1.Service
	for i := range svcList.Items {
		svc := svcList.Items[i]
		fmt.Println("Service name: ", svc.Name, ", LoadBalancer: ", svc.Status.LoadBalancer, ", Ingress: ", svc.Status.LoadBalancer.Ingress)
		if strings.Contains(svc.Name, "ingressgateway") {
			if !requireLoadBalancer {
				fmt.Println("Found ingress gateway: ", svc.Name)
				ingressgateway = svc
			} else {
				if svc.Status.LoadBalancer.Ingress != nil {
					fmt.Println("Found ingress gateway: ", svc.Name)
					ingressgateway = svc
				}
			}
		}
	}
	return ingressgateway
}

func Lookup(url string) bool {
	parsed, err := yourl.Parse(url)
	if err != nil {
		Log(Info, fmt.Sprintf("Error parse %v error: %v", url, err))
		return false
	}
	_, err = net.LookupHost(parsed.Host)
	if err != nil {
		Log(Info, fmt.Sprintf("Error LookupHost %v error: %v", url, err))
		return false
	}
	return true
}

//SecretsCreated checks if all the secrets identified by names are created
func SecretsCreated(namespace string, names ...string) bool {
	secrets := ListSecrets(namespace)
	missing := missingSecrets(secrets.Items, names...)
	Log(Info, fmt.Sprintf("Secrets %v were NOT created in %v", missing, namespace))
	return (len(missing) == 0)
}

func missingSecrets(secrets []corev1.Secret, namePrefixes ...string) []string {
	var missing = []string{}
	for _, name := range namePrefixes {
		if !secretExists(secrets, name) {
			missing = append(missing, name)
		}
	}
	return missing
}

func secretExists(secrets []corev1.Secret, namePrefix string) bool {
	for i := range secrets {
		if strings.HasPrefix(secrets[i].Name, namePrefix) {
			return true
		}
	}
	return false
}

// ListCertificates lists certificates in namespace
func ListCertificates(namespace string) (*certapiv1alpha2.CertificateList, error) {
	certs, err := CertManagerClient().Certificates(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not get list of certificates: %v\n", err.Error()))
	}
	// dump out namespace data to file
	logData := ""
	for i := range certs.Items {
		logData = logData + certs.Items[i].Name + "\n"
	}
	CreateLogFile(fmt.Sprintf("%v-certificates", namespace), logData)
	return certs, err
}

// ListIngress lists ingresses in namespace
func ListIngresses(namespace string) (*extensionsv1beta1.IngressList, error) {
	ingresses, err := GetKubernetesClientset().ExtensionsV1beta1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not get list of ingresses: %v\n", err.Error()))
	}
	// dump out namespace data to file
	logData := ""
	for i := range ingresses.Items {
		logData = logData + ingresses.Items[i].Name + "\n"
	}
	CreateLogFile(fmt.Sprintf("%v-ingresses", namespace), logData)
	return ingresses, err
}