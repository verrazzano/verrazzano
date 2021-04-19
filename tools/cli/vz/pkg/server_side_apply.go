// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var decoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

func ServerSideApply(config *rest.Config, yaml string) error {

	// get a RESTMapper
	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disc))

	// get dynamic client
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	// decode YAML into unstructured.Unstructured
	obj := &unstructured.Unstructured{}
	_, gvk, err := decoder.Decode([]byte(yaml), nil, obj)
	if err != nil {
		return err
	}

	// find GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// get the rest interface for that GVR
	var ri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resource
		ri = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// cluster-wide resource
		ri = dyn.Resource(mapping.Resource)
	}

	// marshall object into JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// create or update the object on the server side
	// types.ApplyPatchType indicated server side apply
	_, err = ri.Patch(context.Background(), obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "verrazzano", // required field
	})
	return err

}
