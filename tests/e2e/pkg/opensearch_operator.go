// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"go.uber.org/zap"
	"text/template"
)

type DNSConfig struct {
	EnvironmentName string
	DNSDomain       string
}

var openSearchCMTemplate = `apiVersion: v1
data:
  name: opensearch-operator
  chartPath: opensearch-operator
  namespace: verrazzano-logging
  overrides: |-
    ingress:
      openSearch:
        enable: true
        annotations:
          cert-manager.io/common-name: opensearch.logging.{{ .EnvironmentName }}.{{ .DNSDomain }}
          external-dns.alpha.kubernetes.io/target: verrazzano-ingress.{{ .EnvironmentName }}.{{ .DNSDomain }}
          external-dns.alpha.kubernetes.io/ttl: "60"
          kubernetes.io/tls-acme: "true"
          nginx.ingress.kubernetes.io/proxy-body-size: 65M
          nginx.ingress.kubernetes.io/rewrite-target: /$2
          nginx.ingress.kubernetes.io/service-upstream: "true"
          nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local
        path: /()(.*)
        ingressClassName: verrazzano-nginx
        host: opensearch.logging.{{ .EnvironmentName }}.{{ .DNSDomain }}
        serviceName: verrazzano-authproxy
        portNumber: 8775
        tls:
          - secretName: tls-opensearch-ingress
            hosts:
              - opensearch.logging.{{ .EnvironmentName }}.{{ .DNSDomain }}

      openSearchDashboards:
        enable: true
        annotations:
          cert-manager.io/common-name: osd.logging.{{ .EnvironmentName }}.{{ .DNSDomain }}
          external-dns.alpha.kubernetes.io/target: verrazzano-ingress.{{ .EnvironmentName }}.{{ .DNSDomain }}
          external-dns.alpha.kubernetes.io/ttl: "60"
          kubernetes.io/tls-acme: "true"
          nginx.ingress.kubernetes.io/proxy-body-size: 65M
          nginx.ingress.kubernetes.io/rewrite-target: /$2
          nginx.ingress.kubernetes.io/service-upstream: "true"
          nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local
        path: /()(.*)
        ingressClassName: verrazzano-nginx
        host: osd.logging.{{ .EnvironmentName }}.{{ .DNSDomain }}
        serviceName: verrazzano-authproxy
        portNumber: 8775
        tls:
          - secretName: tls-osd-ingress
            hosts:
              - osd.logging.{{ .EnvironmentName }}.{{ .DNSDomain }}
    openSearchCluster:
      enabled: true
      name: opensearch
      security:
        config:
          adminCredentialsSecret:
            name: admin-credentials-secret
          securityConfigSecret:
            name: securityconfig-secret
          adminSecret:
            name: opensearch-admin-cert
        tls:
          transport:
            generate: false
            secret:
              name: opensearch-node-cert
            adminDn: ["CN=admin,O=verrazzano"]
            nodesDn: ["CN=opensearch,O=verrazzano"]
          http:
            generate: false
            secret:
              name: opensearch-master-cert
      general:
        httpPort: 9200
        serviceName: opensearch
        version: 2.3.0
        drainDataNodes: true
        image: "iad.ocir.io/odsbuilddev/sandboxes/saket.m.mahto/opensearch-security:latest"
      dashboards:
        image: "iad.ocir.io/odsbuilddev/sandboxes/isha.girdhar/osd:latest"
        opensearchCredentialsSecret:
          name: admin-credentials-secret
        additionalConfig:
          server.name: opensearch-dashboards
          opensearch_security.auth.type: "proxy"
          opensearch_security.proxycache.user_header: "X-WEBAUTH-USER"
          opensearch_security.proxycache.roles_header: "x-proxy-roles"
          opensearch.requestHeadersAllowlist: "[\"securitytenant\",\"Authorization\",\"x-forwarded-for\",\"X-WEBAUTH-USER\",\"x-proxy-roles\"]"
          opensearch_security.multitenancy.enabled: "false"
        tls:
          enable: true
          generate: false
          secret:
            name: opensearch-dashboards-cert
        version: 2.3.0
        enable: true
        replicas: 1
      nodePools:
        - component: masters
          replicas: 3
          diskSize: "50Gi"
          resources:
            requests:
              memory: "1.4Gi"
          roles:
            - "cluster_manager"
        - component: data
          replicas: 3
          diskSize: "50Gi"
          resources:
            requests:
              memory: "4.8Gi"
          roles:
            - "data"
        - component: ingest
          replicas: 1
          resources:
            requests:
              memory: "2.5Gi"
          roles:
            - "ingest"
          persistence:
            emptyDir: {}
kind: ConfigMap
metadata:
  labels:
    experimental.verrazzano.io/configmap-kind: HelmComponent
    experimental.verrazzano.io/configmap-apiversion: v1beta2
  name: dev-opensearch
  namespace: default`

func InstallOpenSearchOperator(log *zap.SugaredLogger) error {
	//kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	//if err != nil {
	//	Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
	//	return err
	//}
	cr, err := GetVerrazzano()
	if err != nil {
		return err
	}
	currentEnvironmentName := GetEnvironmentName(cr)
	currentDNSDomain := GetDNS(cr)

	template, err := template.New("openSearchCMTemplate").Parse(openSearchCMTemplate)
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
		return err
	}
	var buffer bytes.Buffer
	err = template.Execute(&buffer, DNSConfig{
		EnvironmentName: currentEnvironmentName,
		DNSDomain:       currentDNSDomain,
	})
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
		return err
	}

	data := buffer.Bytes()
	err = resource.CreateOrUpdateResourceFromBytes(data, log)

	return err
}
