package main

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"io/ioutil"
	"istio.io/api/networking/v1beta1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var cli client.Reader

func main() {
	virtualservice := &vsapi.VirtualService{}
	ingressTrait := &vzapi.IngressTrait{}
	verrazzanoHelidonWorkload := &vzapi.VerrazzanoHelidonWorkload{}

	//Read OAM File
	appData, err := ioutil.ReadFile("/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/helidon-config-app.yaml")
	if err != nil {
		fmt.Println("Failed to read YAML file:", err)
		return
	}

	//Read Comp file
	compData, err := ioutil.ReadFile("/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/helidon-config-comp.yaml")
	if err != nil {
		fmt.Println("Failed to read YAML file:", err)
		return
	}

	// Unmarshal YAML data into IngressTrait struct
	//fmt.Println(string(appData))
	err = yaml.Unmarshal(appData, ingressTrait)
	if err != nil {
		fmt.Println("Failed to unmarshal YAML:", err)
		return
	}

	// Unmarshal Comp data data into helidon config

	err = yaml.Unmarshal(compData, verrazzanoHelidonWorkload)
	if err != nil {
		fmt.Println("Failed to unmarshal comp data:", err)
		return
	}
	//fmt.Println("workload", verrazzanoHelidonWorkload)
	//Doing conversion
	virtualservice.TypeMeta.APIVersion = ingressTrait.TypeMeta.APIVersion
	virtualservice.APIVersion = "v1"                                             //Assigning API version
	virtualservice.TypeMeta.Kind = "VirtualService"                              //Assigning Kind
	virtualservice.ObjectMeta.Name = ingressTrait.Namespace + "-virtual-service" //Assigning name
	virtualservice.ObjectMeta.Namespace = ingressTrait.ObjectMeta.Namespace      //Assigning namespace

	gateway := []string{}
	gateway = append(gateway, ingressTrait.ObjectMeta.Namespace+"-"+ingressTrait.ObjectMeta.Name+"-appconf-gw")
	virtualservice.Spec.Gateways = gateway //building gateway name

	hostLen := len(ingressTrait.Spec.Rules[0].Hosts)
	if hostLen > 0 {
		for i := 0; i < hostLen; i++ {
			virtualservice.Spec.Hosts = ingressTrait.Spec.Rules[0].Hosts //Assigning host
		}

	} else {
		//buildHostName
		cfg, _ := config.GetConfig()
		cli, _ := client.New(cfg, client.Options{})

		const externalDNSKey = "external-dns.alpha.kubernetes.io/target"
		const wildcardDomainKey = "verrazzano.io/dns.wildcard.domain"

		ingress := k8net.Ingress{}
		error := cli.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-ingress", Namespace: "verrazzano-system"}, &ingress)
		if error != nil {
			fmt.Println(error)
		}
		externalDNSAnno, ok := ingress.Annotations[externalDNSKey]
		if !ok || len(externalDNSAnno) == 0 {
			fmt.Errorf("Annotation %s missing from Verrazzano ingress, unable to generate DNS name", externalDNSKey)
		}
		domain := externalDNSAnno[len("verrazzano-ingress")+1:]
		suffix := ""
		wildcardDomainAnno, ok := ingress.Annotations[wildcardDomainKey]
		if ok {
			suffix = wildcardDomainAnno
		}

		//Build the domain name using Istio info
		if len(suffix) != 0 {
			domain, err = buildDomainNameForWildcard(cli, ingressTrait, suffix)
			if err != nil {
				fmt.Println("Hello in error len")
			}
		}
		//
		//fmt.Println("Printing")
		fmt.Println("Printing Hostname created - ", ingressTrait.Namespace+"."+domain)
		fmt.Println("Printing Port Number from HelidonWorkload - ", verrazzanoHelidonWorkload.Spec.DeploymentTemplate.PodSpec.Containers[0].Ports[0].ContainerPort)

		var hosts []string
		hosts = append(hosts, ingressTrait.Namespace+"."+domain)

		virtualservice.Spec.Hosts = hosts

	}
	//Assigning if Host Name provided
	destHostLen := len(ingressTrait.Spec.Rules[0].Destination.Host)

	if destHostLen > 0 {
		for i := 0; i < destHostLen; i++ {
			destination := &v1beta1.Destination{
				Host: ingressTrait.Spec.Rules[0].Destination.Host,
			}
			virtualservice.Spec.Http[0].Route[0].Destination = destination //not working
		}
	}

	if ingressTrait.Spec.Rules[0].Destination.Port != 0 {
		portSelector := &v1beta1.PortSelector{
			Number: ingressTrait.Spec.Rules[0].Destination.Port,
		}
		virtualservice.Spec.Http[0].Route[0].Destination.Port = portSelector
	} else {
		//Fetch port from Workloads

		//fmt.Println("checking")
		//portSelector := &v1beta1.PortSelector{Number: uint32(verrazzanoHelidonWorkload.Spec.DeploymentTemplate.PodSpec.Containers[0].Ports[0].ContainerPort)}
		//destination := &v1beta1.Destination{
		//	Port: portSelector,
		//}
		//
		//virtualservice.Spec.Http[0].Route[0].Destination = destination
		//fmt.Println("check", virtualservice.Spec.Http[0].Route[0].Destination.Port.Number)

		//http := &v1beta1.HTTPRoute{
		//	Route:
		//}
		//httpRoutes := []*v1beta1.HTTPRoute{http}
		//virtualservice.Spec.Http = httpRoutes
		//httpRouteDestination := &v1beta1.HTTPRouteDestination{}
		//httpRoutesDestination := []*v1beta1.HTTPRouteDestination{httpRouteDestination}
		//
		//virtualservice.Spec.Http[0].Route = httpRoutesDestination
		//destination := &v1beta1.Destination{}
		//virtualservice.Spec.Http[0].Route[0].Destination = destination
		//portSelector := &v1beta1.PortSelector{
		//	Number: uint32(verrazzanoHelidonWorkload.Spec.DeploymentTemplate.PodSpec.Containers[0].Ports[0].ContainerPort),
		//}
		//virtualservice.Spec.Http[0].Route[0].Destination.Port = portSelector
		//virtualservice.Spec.Http[0].Route[0].Destination.Port.Number = uint32(verrazzanoHelidonWorkload.Spec.DeploymentTemplate.PodSpec.Containers[0].Ports[0].ContainerPort)
		//fmt.Println("check", virtualservice.Spec.Http[0].Route[0].Destination.Port.Number)

	}

	//Assiging Path
	pathLen := len(ingressTrait.Spec.Rules[0].Paths)

	if pathLen > 0 {
		if ingressTrait.Spec.Rules[0].Paths[0].PathType == "exact" {
			matchRequest := &v1beta1.HTTPMatchRequest{
				Uri: &v1beta1.StringMatch{
					MatchType: &v1beta1.StringMatch_Exact{
						Exact: ingressTrait.Spec.Rules[0].Paths[0].Path,
					},
				},
			}
			matchRequests := []*v1beta1.HTTPMatchRequest{matchRequest}
			http := &v1beta1.HTTPRoute{
				Match: matchRequests,
			}
			httpRoutes := []*v1beta1.HTTPRoute{http}
			virtualservice.Spec.Http = httpRoutes
		}
		if ingressTrait.Spec.Rules[0].Paths[0].PathType == "prefix" {
			matchRequest := &v1beta1.HTTPMatchRequest{
				Uri: &v1beta1.StringMatch{
					MatchType: &v1beta1.StringMatch_Prefix{
						Prefix: ingressTrait.Spec.Rules[0].Paths[0].Path,
					},
				},
			}
			matchRequests := []*v1beta1.HTTPMatchRequest{matchRequest}
			http := &v1beta1.HTTPRoute{
				Match: matchRequests,
			}
			httpRoutes := []*v1beta1.HTTPRoute{http}
			virtualservice.Spec.Http = httpRoutes
		}
	}

	//Printing Virtual Service
	fmt.Print("Virtual Service-", virtualservice)

	//Write in a file
	virtualServiceYAML, err := yaml.Marshal(&virtualservice)
	if err != nil {
		fmt.Printf("Error while Marshaling. %v", err)
	}

	fileName := "/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/test.yaml"
	err = ioutil.WriteFile(fileName, virtualServiceYAML, 0644)
	if err != nil {
		panic("Unable to write data into the file")
	}

}

func buildDomainNameForWildcard(cli client.Reader, trait *vzapi.IngressTrait, suffix string) (string, error) {
	istioIngressGateway := "istio-ingressgateway"
	IstioSystemNamespace := "istio-system"
	istio := corev1.Service{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: istioIngressGateway, Namespace: IstioSystemNamespace}, &istio)
	if err != nil {
		return "", err
	}
	var IP string
	if istio.Spec.Type == corev1.ServiceTypeLoadBalancer || istio.Spec.Type == corev1.ServiceTypeNodePort {
		if len(istio.Spec.ExternalIPs) > 0 {
			IP = istio.Spec.ExternalIPs[0]
		} else if len(istio.Status.LoadBalancer.Ingress) > 0 {
			IP = istio.Status.LoadBalancer.Ingress[0].IP
		} else {
			return "", fmt.Errorf("%s is missing loadbalancer IP", istioIngressGateway)
		}
	} else {
		return "", fmt.Errorf("unsupported service type %s for istio_ingress", string(istio.Spec.Type))
	}
	domain := IP + "." + suffix
	return domain, nil
}
