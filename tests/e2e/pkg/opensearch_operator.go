// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"context"
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"text/template"
	"time"
)

// OSOperatorOverrides are overrides to the opensearch-operator helm
type OSOperatorOverrides struct {
	EnvironmentName string
	DNSSuffix       string
	MasterReplicas  int
	DataReplicas    int
	IngestReplicas  int
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
          cert-manager.io/common-name: opensearch.logging.{{ .EnvironmentName }}.{{ .DNSSuffix }}
          external-dns.alpha.kubernetes.io/target: verrazzano-ingress.{{ .EnvironmentName }}.{{ .DNSSuffix }}
          external-dns.alpha.kubernetes.io/ttl: "60"
          kubernetes.io/tls-acme: "true"
          nginx.ingress.kubernetes.io/proxy-body-size: 65M
          nginx.ingress.kubernetes.io/rewrite-target: /$2
          nginx.ingress.kubernetes.io/service-upstream: "true"
          nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local
        path: /()(.*)
        ingressClassName: verrazzano-nginx
        host: opensearch.logging.{{ .EnvironmentName }}.{{ .DNSSuffix }}
        serviceName: verrazzano-authproxy
        portNumber: 8775
        tls:
          - secretName: tls-opensearch-ingress
            hosts:
              - opensearch.logging.{{ .EnvironmentName }}.{{ .DNSSuffix }}

      openSearchDashboards:
        enable: true
        annotations:
          cert-manager.io/common-name: osd.logging.{{ .EnvironmentName }}.{{ .DNSSuffix }}
          external-dns.alpha.kubernetes.io/target: verrazzano-ingress.{{ .EnvironmentName }}.{{ .DNSSuffix }}
          external-dns.alpha.kubernetes.io/ttl: "60"
          kubernetes.io/tls-acme: "true"
          nginx.ingress.kubernetes.io/proxy-body-size: 65M
          nginx.ingress.kubernetes.io/rewrite-target: /$2
          nginx.ingress.kubernetes.io/service-upstream: "true"
          nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local
        path: /()(.*)
        ingressClassName: verrazzano-nginx
        host: osd.logging.{{ .EnvironmentName }}.{{ .DNSSuffix }}
        serviceName: verrazzano-authproxy
        portNumber: 8775
        tls:
          - secretName: tls-osd-ingress
            hosts:
              - osd.logging.{{ .EnvironmentName }}.{{ .DNSSuffix }}
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
          replicas: {{ .MasterReplicas }}
          diskSize: "50Gi"
          resources:
            requests:
              memory: "1.4Gi"
          roles:
            - "cluster_manager"
        - component: data
          replicas: {{ .DataReplicas }}
          diskSize: "50Gi"
          resources:
            requests:
              memory: "4.8Gi"
          roles:
            - "data"
        - component: ingest
          replicas: {{ .IngestReplicas }}
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

// InstallOrUpdateOpenSearchOperator creates or updates the CM for the dev-controller
// to install or upgrade the opensearch-operator helm chart
func InstallOrUpdateOpenSearchOperator(log *zap.SugaredLogger, master, data, ingest int) error {
	cr, err := GetVerrazzano()
	if err != nil {
		return err
	}
	currentEnvironmentName := GetEnvironmentName(cr)
	currentDNSSuffix := fmt.Sprintf("%s.%s", GetIngressIP(cr), GetDNS(cr))

	tmpl, err := template.New("openSearchCMTemplate").Parse(openSearchCMTemplate)
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
		return err
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, OSOperatorOverrides{
		EnvironmentName: currentEnvironmentName,
		DNSSuffix:       currentDNSSuffix,
		MasterReplicas:  master,
		DataReplicas:    data,
		IngestReplicas:  ingest,
	})
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
		return err
	}

	err = resource.CreateOrUpdateResourceFromBytes(buffer.Bytes(), log)

	return err
}

// UninstallOpenSearchOperator delete the CM so that the dev-controller
// can uninstall opensearch-operator helm
func UninstallOpenSearchOperator() error {
	err := DeleteConfigMap("default", "dev-opensearch")
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
		return err
	}
	return nil
}

// EventuallyPodsReady check whether the required number of master, data and ingest pods are ready or not
func EventuallyPodsReady(log *zap.SugaredLogger, master, data, ingest int) {
	timeout := 15 * time.Minute
	pollInterval := 30 * time.Second
	gomega.Eventually(func() bool {
		if err := verifyReadyReplicas(master, data, ingest); err != nil {
			log.Errorf("opensearch pods not ready: %v", err)
			return false
		}
		return true
	}, timeout, pollInterval).Should(gomega.BeTrue())
}

func verifyReadyReplicas(master, data, ingest int) error {
	if err := assertPodsFound(master, labelSelector("masters")); err != nil {
		return err
	}
	if err := assertPodsFound(data, labelSelector("data")); err != nil {
		return err
	}
	if err := assertPodsFound(ingest, labelSelector("ingest")); err != nil {
		return err
	}
	return nil
}

func assertPodsFound(count int, selector string) error {
	kubeClientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	pods, err := kubeClientSet.CoreV1().Pods("verrazzano-logging").List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	if len(pods.Items) != count {
		return fmt.Errorf("expected %d pods, found %d", count, len(pods.Items))
	}
	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if !status.Ready {
				return fmt.Errorf("container %s/%s is not yet ready", pod.Name, status.Name)
			}
		}
	}
	return nil
}

func labelSelector(label string) string {
	return fmt.Sprintf("opster.io/opensearch-nodepool=%s", label)
}
