// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/jsonpath"
)

const Healthy = "ok"
const overallHealth = "overallHealth"

// GetDomain returns a WebLogic domains in unstructured format
func GetDomain(namespace string, name string) (*unstructured.Unstructured, error) {
	client, err := pkg.GetDynamicClient()
	if err != nil {
		return nil, err
	}
	domain, err := client.Resource(getScheme()).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return domain, nil
}

// GetDomainInCluster returns a WebLogic domains in unstructured format
func GetDomainInCluster(namespace string, name string, kubeconfigPath string) (*unstructured.Unstructured, error) {
	client, err := pkg.GetDynamicClientInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	domain, err := client.Resource(getScheme()).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return domain, nil
}

// GetHealthOfServers returns a slice of strings, each item representing the health of a server in the domain
func GetHealthOfServers(uDomain *unstructured.Unstructured) ([]string, error) {
	// jsonpath template used to extract the server info
	const template = `{.status.servers[*].health}`

	// Get the string that has the server status health
	results, err := findData(uDomain, template)

	if err != nil {
		return nil, err
	}

	var serverHealth []string
	for i := range results {
		for j := range results[i] {
			if results[i][j].CanInterface() {
				// Get the underlying interface object out of the reflect.Value object in results[i][j]
				serverHealthIntf := results[i][j].Interface()

				// Cast the interface{} object to a map[string]interface{} (since we know that is what the health object is)
				// (if type is wrong, instead of panicking the cast will return false for the 2nd return value)
				healthMap, ok := serverHealthIntf.(map[string]interface{})
				if !ok {
					pkg.Log(pkg.Error, fmt.Sprintf("Could not get WebLogic server result # %d as a map, its type is %v", i, reflect.TypeOf(serverHealthIntf)))
					continue
				}
				// append the overallHealth value ("ok" or otherwise) to a string slice, one entry per server.
				serverHealth = append(serverHealth, healthMap[overallHealth].(string))
			} else {
				pkg.Log(pkg.Error, fmt.Sprintf("Could not convert WebLogic server result # %d to an interface, its type is %s", i, results[i][j].Type().String()))
			}
		}
	}

	if len(serverHealth) == 0 {
		return nil, errors.New("No WebLogic servers found in domain CR")
	}
	return serverHealth, nil
}

// getScheme returns the WebLogic scheme needed to get unstructured data
func getScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "weblogic.oracle",
		Version:  "v8",
		Resource: "domains",
	}
}

// findData returns the results for the specified value from the unstructured domain that matches the template
func findData(uDomain *unstructured.Unstructured, template string) ([][]reflect.Value, error) {
	// Convert the unstructured domain into domain object that can be parsed
	jsonDomain, err := uDomain.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var domain interface{}
	err = json.Unmarshal([]byte(jsonDomain), &domain)
	if err != nil {
		return nil, err
	}
	// Parse the template
	j := jsonpath.New("domain")
	err = j.Parse(template)
	if err != nil {
		return nil, err
	}

	return j.FindResults(domain)
}
