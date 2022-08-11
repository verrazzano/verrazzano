// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"context"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
	"text/template"
)

const (
	jaegerOperatorName = "jaeger-operator"
	jaegerNamespace    = constants.VerrazzanoMonitoringNamespace
	jaegerAPIVersion   = "jaegertracing.io/v1"
	jaegerKind         = "Jaeger"
	jaegerName         = "jaeger-verrazzano-managed-cluster"
	jaegerOSTLSCAKey   = "ca.crt"
	jaegerOSTLSKey     = "os.tls.key"
	jaegerOSTLSCertKey = "os.tls.cert"
	// specField is the name of the spec field with unstructured
	specField = "spec"
)

// jaegerData needed for template rendering
type jaegerData struct {
	ClusterName   string
	OpenSearchURL string
	SecretName    string
	MutualTLS     bool
}

// A template to define Jaeger CR for creating a Jaeger instance in managed cluster.
const jaegerManagedClusterCRTemplate = `
apiVersion: ` + jaegerAPIVersion + `
kind: ` + jaegerKind + `
metadata:
  name: ` + jaegerName + `
  namespace: ` + jaegerNamespace + `
spec:
  annotations:
    sidecar.istio.io/inject: "true"
    proxy.istio.io/config: '{ "holdApplicationUntilProxyStarts": true }'
  ingress:
    enabled: false
  strategy: production
  query:
    replicas: 0
  collector:
    options:
      collector:
        tags: verrazzano_cluster={{.ClusterName}}
  storage:
    dependencies:
      enabled: false
    esIndexCleaner:
      enabled: false
    type: elasticsearch
    options:
      es:
        index-prefix: verrazzano
        server-urls: {{.OpenSearchURL}}
        tls:
          ca: /verrazzano/certificates/` + jaegerOSTLSCAKey + `
{{if .MutualTLS}}
          key: /verrazzano/certificates/` + jaegerOSTLSKey + `
          cert: /verrazzano/certificates/` + jaegerOSTLSCertKey + `
{{end}}
    secretName: {{.SecretName}}
  volumeMounts:
    - name: certificates
      mountPath: /verrazzano/certificates/
      readOnly: true
  volumes:
    - name: certificates
      secret:
        secretName: {{.SecretName}}
`

// Creates or Updates Jaeger CR
func (s *Syncer) configureJaegerCR() {
	// Skip the creation of Jaeger instance if Jaeger Operator is not installed
	if !s.isJaegerOperatorConfigured() {
		return
	}
	var osURL, clusterName string
	// Get the secret containing managed cluster name and OpenSearch URL
	secretNamespacedName := types.NamespacedName{Name: constants.MCRegistrationSecret,
		Namespace: constants.VerrazzanoSystemNamespace}
	sec := corev1.Secret{}
	err := s.LocalClient.Get(context.TODO(), secretNamespacedName, &sec)
	if err != nil {
		s.Log.Debugf("Not able to get the Multicluster registration secret")
		return
	}
	// The secret will contain the OpenSearch endpoint's URL if Jaeger instance exists on admin cluster
	openSearchURL, ok := sec.Data[mcconstants.JaegerOSURLKey]
	if !ok || len(openSearchURL) < 1 {
		s.Log.Debugf("Not able to find the Jaeger OpenSearch URL in the secret. Skipping the creation of Jaeger CR")
		return
	}
	osURL = string(openSearchURL)
	err = s.createJaegerSecret(mcconstants.JaegerManagedClusterSecretName, sec)
	if err != nil {
		s.Log.Errorf("Failed to create the Jaeger secret: %v", err)
		return
	}
	// Check if Jaeger instance is using mutual TLS to connect to OpenSearch
	mutualTLS := false
	tlsKey, ok := sec.Data[mcconstants.JaegerOSTLSKey]
	if ok && string(tlsKey) != "" {
		mutualTLS = true
	}
	clusterName = string(sec.Data[constants.ClusterNameData])
	// use template to populate Jaeger spec data
	jaegerTemplate, err := template.New("jaeger").Parse(jaegerManagedClusterCRTemplate)
	if err != nil {
		s.Log.Errorf("Failed to create the Jaeger template: %v", err)
		return
	}
	var b bytes.Buffer
	err = jaegerTemplate.Execute(&b, jaegerData{ClusterName: clusterName, OpenSearchURL: osURL,
		SecretName: mcconstants.JaegerManagedClusterSecretName, MutualTLS: mutualTLS})
	if err != nil {
		s.Log.Errorf("Failed to execute the Jaeger template: %v", err)
		return
	}

	// Unmarshal the YAML into an object
	u := &unstructured.Unstructured{Object: map[string]interface{}{}}
	err = yaml.Unmarshal(b.Bytes(), u)
	if err != nil {
		s.Log.Errorf("Failed to unmarshal the Jaeger CR yaml: %v", err)
		return

	}
	// Make a copy of the spec field
	jaegerSpec, _, err := unstructured.NestedFieldCopy(u.Object, "spec")
	if err != nil {
		s.Log.Errorf("Failed to make a copy of the Jaeger spec: %v", err)
		return
	}

	// Create or update the Jaeger CR.  Always replace the entire spec.
	var jaeger unstructured.Unstructured
	jaeger.SetAPIVersion(jaegerAPIVersion)
	jaeger.SetKind(jaegerKind)
	jaeger.SetName(jaegerName)
	jaeger.SetNamespace(jaegerNamespace)
	_, err = controllerutil.CreateOrUpdate(context.TODO(), s.LocalClient, &jaeger, func() error {
		if err := unstructured.SetNestedField(jaeger.Object, jaegerSpec, specField); err != nil {
			s.Log.Errorf("Unable to set the Jaeger spec: %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		s.Log.Errorf("Failed to create the Jaeger CR: %v", err)
		return
	}
	s.Log.Debugf("Created the Jaeger CR successfully")
}

// createJaegerSecret creates a Jaeger secret for storing credentials needed to access OpenSearch.
func (s *Syncer) createJaegerSecret(secName string, mcSecret corev1.Secret) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secName,
			Namespace: jaegerNamespace,
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), s.LocalClient, secret, func() error {
		secret.Data = make(map[string][]byte)
		if _, exists := mcSecret.Data[mcconstants.JaegerOSUsernameKey]; exists {
			secret.Data[mcconstants.JaegerOSUsernameKey] = mcSecret.Data[mcconstants.JaegerOSUsernameKey]
		}
		if _, exists := mcSecret.Data[mcconstants.JaegerOSPasswordKey]; exists {
			secret.Data[mcconstants.JaegerOSPasswordKey] = mcSecret.Data[mcconstants.JaegerOSPasswordKey]
		}
		if _, exists := mcSecret.Data[mcconstants.JaegerOSTLSCAKey]; exists {
			secret.Data[jaegerOSTLSCAKey] = mcSecret.Data[mcconstants.JaegerOSTLSCAKey]
		}
		if _, exists := mcSecret.Data[mcconstants.JaegerOSTLSKey]; exists {
			secret.Data[jaegerOSTLSKey] = mcSecret.Data[mcconstants.JaegerOSTLSKey]
		}
		if _, exists := mcSecret.Data[mcconstants.JaegerOSTLSCertKey]; exists {
			secret.Data[jaegerOSTLSCertKey] = mcSecret.Data[mcconstants.JaegerOSTLSCertKey]
		}
		return nil
	}); err != nil {
		s.Log.Errorf("Failed to create or update the %s secret: %v", secName, err)
		return err
	}
	return nil
}

func (s *Syncer) isJaegerOperatorConfigured() bool {
	namespaceName := types.NamespacedName{Name: jaegerOperatorName, Namespace: jaegerNamespace}
	deployment := appsv1.Deployment{}
	err := s.LocalClient.Get(context.TODO(), namespaceName, &deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			s.Log.Debugf("Jaeger Operator deployment %s/%s does not exit. Skipping Jaeger instance creation",
				jaegerNamespace, jaegerOperatorName)
			return false
		}
		s.Log.Errorf("Failed to check Jaeger Operator deployment %s/%s: %v", jaegerNamespace, jaegerOperatorName,
			err)
		return false
	}
	return true
}
