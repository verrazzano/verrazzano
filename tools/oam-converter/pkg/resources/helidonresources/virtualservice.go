// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	vs "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/virtualservice"
	istio "istio.io/api/networking/v1beta1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func createVirtualServiceFromHelidonWorkload(appNamespace string, rule vzapi.IngressRule,
	allHostsForTrait []string, name string, gateway *vsapi.Gateway, helidonWorkload *unstructured.Unstructured) (*vsapi.VirtualService, error) {
	virtualService := &vsapi.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: consts.VirtualServiceAPIVersion,
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: appNamespace,
			Name:      name,
		},
	}
	return mutateVirtualServiceFromHelidonWorkload(virtualService, rule, allHostsForTrait, gateway, helidonWorkload)
}

// mutateVirtualService mutates the output virtual service resource
func mutateVirtualServiceFromHelidonWorkload(virtualService *vsapi.VirtualService, rule vzapi.IngressRule, allHostsForTrait []string, gateway *vsapi.Gateway, helidonWorkload *unstructured.Unstructured) (*vsapi.VirtualService, error) {
	virtualService.Spec.Gateways = []string{gateway.Name}
	virtualService.Spec.Hosts = allHostsForTrait
	matches := []*istio.HTTPMatchRequest{}
	paths := vs.GetPathsFromRule(rule)
	for _, path := range paths {
		matches = append(matches, &istio.HTTPMatchRequest{
			Uri: vs.CreateVirtualServiceMatchURIFromIngressTraitPath(path)})
	}
	dest, err := createDestinationFromRuleOrHelidonWorkload(rule, helidonWorkload)
	if err != nil {
		print(err)
		return nil, err
	}
	route := istio.HTTPRoute{
		Match: matches,
		Route: []*istio.HTTPRouteDestination{dest}}
	virtualService.Spec.Http = []*istio.HTTPRoute{&route}

	return virtualService, nil
}
