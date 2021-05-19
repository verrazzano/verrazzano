// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/jsonpath"
)

// ListDomains returns the list of WebLogic domains in unstructured format
func ListDomains(namespace string, kubeconfigPath string) (*unstructured.UnstructuredList, error) {
	client, err := dynamic.NewForConfig(pkg.GetKubeConfigGivenPath(kubeconfigPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}
	scheme := schema.GroupVersionResource{
		Group:    "weblogic.oracle",
		Version:  "v8",
		Resource: "domains",
	}
	res, err := client.Resource(scheme).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	return res, nil
}

func checkServerStatus(u *unstructured.Unstructured) error {
	// Convert the unstructured domain into domain object that can be parsed
	jsonDomain, err := u.MarshalJSON()
	if err != nil {
		return err
	}
	var domain interface{}
	err = json.Unmarshal([]byte(jsonDomain), &domain)
	if err != nil {
		return err
	}

	// Parse the template
	const template = `{range .status.servers[*]}{.health.overallHealth} {end}`
	j := jsonpath.New("domain")
	err = j.Parse(template)
	if err != nil {
		return err
	}

	// Execute the search for the data
	buf := new(bytes.Buffer)
	err = j.Execute(buf, domain)
	if err != nil {
		return err
	}

	s := buf.String()
	fmt.Printf("health: %s\n", s)
	return nil
}
