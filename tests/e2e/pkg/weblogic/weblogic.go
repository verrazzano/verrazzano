// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/jsonpath"
)

const Healthy = "ok"

// ServerHealth contains the health information for a WebLogic server
type ServerHealth struct {
	ServerName string
	Health     string
}

// ListDomains returns the list of WebLogic domains in unstructured format
func ListDomains(namespace string) (*unstructured.UnstructuredList, error) {
	client := pkg.GetDynamicClient()
	domains, err := client.Resource(getScheme()).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return domains, nil
}

// GetDomain returns a WebLogic domains in unstructured format
func GetDomain(namespace string, name string) (*unstructured.Unstructured, error) {
	client := pkg.GetDynamicClient()
	domain, err := client.Resource(getScheme()).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return domain, nil
}

// GetHealthOfServers returns an array of ServerHealth objects
func GetHealthOfServers(uDomain *unstructured.Unstructured) ([]ServerHealth, error) {
	// Separator for list of servers
	const serverSep = ","
	// Separator for server fields
	const fieldSep = ":"
	// Template used to parse for the server health status
	const template = `{range .status.servers[*]}{.serverName}:{.health.overallHealth},{end}`

	// Get the string that has the server status health
	s, err := findData(uDomain, template)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("servers: %s\n", s)
	// Convert the string to a slice of ServerHealth
	servers := strings.Split(s, serverSep)
	if len(servers) == 0 {
		return nil, errors.New("No WebLogic servers found in domain CR")
	}
	var ss []ServerHealth
	for _, server := range servers {
		if len(server) == 0 {
			break
		}
		fields := strings.Split(server, fieldSep)
		if len(fields) == 0 {
			return nil, errors.New("WebLogic server fields not found in domain CR")
		}
		ss = append(ss, ServerHealth{
			ServerName: strings.TrimSpace(fields[0]),
			Health:     strings.TrimSpace(fields[1]),
		})
	}
	return ss, nil
}

// getScheme returns the WebLogic scheme needed to get unstructured data
func getScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "weblogic.oracle",
		Version:  "v8",
		Resource: "domains",
	}
}

// findData returns a string from the unstructured domain that matches the template
func findData(uDomain *unstructured.Unstructured, template string) (string, error) {
	// Convert the unstructured domain into domain object that can be parsed
	jsonDomain, err := uDomain.MarshalJSON()
	if err != nil {
		return "", err
	}
	var domain interface{}
	err = json.Unmarshal([]byte(jsonDomain), &domain)
	if err != nil {
		return "", err
	}
	// Parse the template
	j := jsonpath.New("domain")
	err = j.Parse(template)
	if err != nil {
		return "", err
	}
	// Execute the search for the server status
	buf := new(bytes.Buffer)
	err = j.Execute(buf, domain)
	if err != nil {
		return "", err
	}
	// Convert to ServerHealth structs
	s := buf.String()
	return s, nil
}
