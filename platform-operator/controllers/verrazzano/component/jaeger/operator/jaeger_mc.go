// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
	"time"
)

const (
	jaegerOSTLSCAKey   = "ca.crt"
	jaegerOSTLSKey     = "os.tls.key"
	jaegerOSTLSCertKey = "os.tls.cert"
)

const jaegerMCTemplate = `jaeger:
  create: true
  spec:
    annotations:
      sidecar.istio.io/inject: "true"
      proxy.istio.io/config: '{ "holdApplicationUntilProxyStarts": true }'
{{if .IsForceRecreate}}
      verrazzano.io/recreatedAt: "{{now.UnixMilli}}"
{{end}}
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

// jaegerMCData needed for template rendering
type jaegerMCData struct {
	ClusterName     string
	OpenSearchURL   string
	SecretName      string
	MutualTLS       bool
	IsForceRecreate bool
}

// renderManagedClusterInstance renders the values for a managed cluster Jaeger instance into the provided buffer
func renderManagedClusterInstance(client clipkg.Client, registrationSecret *corev1.Secret, buffer *bytes.Buffer) error {
	data, err := buildJaegerMCData(registrationSecret)
	if err != nil {
		return err
	}
	return executeJaegerMCTemplate(data, buffer)
}

func buildJaegerMCData(registrationSecret *corev1.Secret) (*jaegerMCData, error) {
	osURL, ok := registrationSecret.Data[mcconstants.JaegerOSURLKey]
	if !ok || len(osURL) < 1 {
		return nil, errors.New("unable to find OpenSearch URL for managed-cluster Jaeger")
	}
	mutalTLS := false
	tlsKey, ok := registrationSecret.Data[mcconstants.JaegerOSTLSKey]
	if ok && len(tlsKey) > 0 {
		mutalTLS = true
	}
	return &jaegerMCData{
		ClusterName:     string(registrationSecret.Data[constants.ClusterNameData]),
		OpenSearchURL:   string(osURL),
		SecretName:      mcconstants.JaegerManagedClusterSecretName,
		MutualTLS:       mutalTLS,
		IsForceRecreate: true,
	}, nil
}

func executeJaegerMCTemplate(data *jaegerMCData, buffer *bytes.Buffer) error {
	jaegerTemplate, err := template.New("jaeger").Funcs(template.FuncMap{
		"now": time.Now,
	}).Parse(jaegerMCTemplate)
	if err != nil {
		return fmt.Errorf("failed to create the Jaeger managed-cluster template: %v", err)
	}
	return jaegerTemplate.Execute(buffer, *data)
}

func createOrUpdateMCSecret(client clipkg.Client, registrationSecret *corev1.Secret) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcconstants.JaegerManagedClusterSecretName,
			Namespace: ComponentNamespace,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), client, secret, func() error {
		secret.Data = make(map[string][]byte)
		if _, exists := registrationSecret.Data[mcconstants.JaegerOSUsernameKey]; exists {
			secret.Data[mcconstants.JaegerOSUsernameKey] = registrationSecret.Data[mcconstants.JaegerOSUsernameKey]
		}
		if _, exists := registrationSecret.Data[mcconstants.JaegerOSPasswordKey]; exists {
			secret.Data[mcconstants.JaegerOSPasswordKey] = registrationSecret.Data[mcconstants.JaegerOSPasswordKey]
		}
		if _, exists := registrationSecret.Data[mcconstants.JaegerOSTLSCAKey]; exists {
			secret.Data[jaegerOSTLSCAKey] = registrationSecret.Data[mcconstants.JaegerOSTLSCAKey]
		}
		if _, exists := registrationSecret.Data[mcconstants.JaegerOSTLSKey]; exists {
			secret.Data[jaegerOSTLSKey] = registrationSecret.Data[mcconstants.JaegerOSTLSKey]
		}
		if _, exists := registrationSecret.Data[mcconstants.JaegerOSTLSCertKey]; exists {
			secret.Data[jaegerOSTLSCertKey] = registrationSecret.Data[mcconstants.JaegerOSTLSCertKey]
		}
		return nil
	})
	return err
}
