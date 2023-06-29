// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import "time"

type Cluster struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Labels            struct {
			ClusterXK8SIoClusterName string `json:"cluster.x-k8s.io/cluster-name"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		ClusterNetwork struct {
			Pods struct {
				CidrBlocks []string `json:"cidrBlocks"`
			} `json:"pods"`
			ServiceDomain string `json:"serviceDomain"`
			Services      struct {
				CidrBlocks []string `json:"cidrBlocks"`
			} `json:"services"`
		} `json:"clusterNetwork"`
		ControlPlaneEndpoint struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"controlPlaneEndpoint"`
		ControlPlaneRef struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			Namespace  string `json:"namespace"`
		} `json:"controlPlaneRef"`
		InfrastructureRef struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			Namespace  string `json:"namespace"`
		} `json:"infrastructureRef"`
	} `json:"spec"`
	Status struct {
		Conditions []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		ControlPlaneReady bool `json:"controlPlaneReady"`
		FailureDomains    struct {
			Field1 struct {
				Attributes struct {
					AvailabilityDomain string `json:"AvailabilityDomain"`
				} `json:"attributes"`
				ControlPlane bool `json:"controlPlane"`
			} `json:"1"`
			Field2 struct {
				Attributes struct {
					AvailabilityDomain string `json:"AvailabilityDomain"`
				} `json:"attributes"`
				ControlPlane bool `json:"controlPlane"`
			} `json:"2"`
			Field3 struct {
				Attributes struct {
					AvailabilityDomain string `json:"AvailabilityDomain"`
				} `json:"attributes"`
				ControlPlane bool `json:"controlPlane"`
			} `json:"3"`
		} `json:"failureDomains"`
		InfrastructureReady bool   `json:"infrastructureReady"`
		ObservedGeneration  int    `json:"observedGeneration"`
		Phase               string `json:"phase"`
	} `json:"status"`
}

type OCNEControlPlane struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Labels            struct {
			ClusterXK8SIoClusterName string `json:"cluster.x-k8s.io/cluster-name"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		OwnerReferences []struct {
			APIVersion         string `json:"apiVersion"`
			BlockOwnerDeletion bool   `json:"blockOwnerDeletion"`
			Controller         bool   `json:"controller"`
			Kind               string `json:"kind"`
			Name               string `json:"name"`
			UID                string `json:"uid"`
		} `json:"ownerReferences"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		ControlPlaneConfig struct {
			ClusterConfiguration struct {
				APIServer struct {
					CertSANs []string `json:"certSANs"`
				} `json:"apiServer"`
				ControllerManager struct {
				} `json:"controllerManager"`
				DNS struct {
					ImageRepository string `json:"imageRepository"`
					ImageTag        string `json:"imageTag"`
				} `json:"dns"`
				Etcd struct {
					Local struct {
						ImageRepository string `json:"imageRepository"`
						ImageTag        string `json:"imageTag"`
					} `json:"local"`
				} `json:"etcd"`
				ImageRepository string `json:"imageRepository"`
				Networking      struct {
				} `json:"networking"`
				Scheduler struct {
				} `json:"scheduler"`
			} `json:"clusterConfiguration"`
			Format             string `json:"format"`
			ImageConfiguration struct {
				Proxy struct {
					HTTPProxy  string `json:"httpProxy"`
					HTTPSProxy string `json:"httpsProxy"`
					NoProxy    string `json:"noProxy"`
				} `json:"proxy"`
			} `json:"imageConfiguration"`
			InitConfiguration struct {
				LocalAPIEndpoint struct {
				} `json:"localAPIEndpoint"`
				NodeRegistration struct {
					CriSocket        string `json:"criSocket"`
					KubeletExtraArgs struct {
						CloudProvider string `json:"cloud-provider"`
						ProviderID    string `json:"provider-id"`
					} `json:"kubeletExtraArgs"`
				} `json:"nodeRegistration"`
			} `json:"initConfiguration"`
			JoinConfiguration struct {
				Discovery struct {
				} `json:"discovery"`
				NodeRegistration struct {
					CriSocket        string `json:"criSocket"`
					KubeletExtraArgs struct {
						CloudProvider string `json:"cloud-provider"`
						ProviderID    string `json:"provider-id"`
					} `json:"kubeletExtraArgs"`
				} `json:"nodeRegistration"`
			} `json:"joinConfiguration"`
		} `json:"controlPlaneConfig"`
		MachineTemplate struct {
			InfrastructureRef struct {
				APIVersion string `json:"apiVersion"`
				Kind       string `json:"kind"`
				Name       string `json:"name"`
				Namespace  string `json:"namespace"`
			} `json:"infrastructureRef"`
			Metadata struct {
			} `json:"metadata"`
		} `json:"machineTemplate"`
		ModuleOperator struct {
			Enabled bool `json:"enabled"`
		} `json:"moduleOperator"`
		Replicas        int `json:"replicas"`
		RolloutStrategy struct {
			RollingUpdate struct {
				MaxSurge int `json:"maxSurge"`
			} `json:"rollingUpdate"`
			Type string `json:"type"`
		} `json:"rolloutStrategy"`
		Version string `json:"version"`
	} `json:"spec"`
	Status struct {
		Conditions []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		Initialized         bool   `json:"initialized"`
		ObservedGeneration  int    `json:"observedGeneration"`
		Ready               bool   `json:"ready"`
		ReadyReplicas       int    `json:"readyReplicas"`
		Replicas            int    `json:"replicas"`
		Selector            string `json:"selector"`
		UnavailableReplicas int    `json:"unavailableReplicas"`
		UpdatedReplicas     int    `json:"updatedReplicas"`
		Version             string `json:"version"`
	} `json:"status"`
}

type Machine struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			ControlplaneClusterXK8SIoOcneClusterConfiguration string `json:"controlplane.cluster.x-k8s.io/ocne-cluster-configuration"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Labels            struct {
			ClusterXK8SIoClusterName      string `json:"cluster.x-k8s.io/cluster-name"`
			ClusterXK8SIoControlPlane     string `json:"cluster.x-k8s.io/control-plane"`
			ClusterXK8SIoControlPlaneName string `json:"cluster.x-k8s.io/control-plane-name"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		OwnerReferences []struct {
			APIVersion         string `json:"apiVersion"`
			BlockOwnerDeletion bool   `json:"blockOwnerDeletion"`
			Controller         bool   `json:"controller"`
			Kind               string `json:"kind"`
			Name               string `json:"name"`
			UID                string `json:"uid"`
		} `json:"ownerReferences"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		Bootstrap struct {
			ConfigRef struct {
				APIVersion string `json:"apiVersion"`
				Kind       string `json:"kind"`
				Name       string `json:"name"`
				Namespace  string `json:"namespace"`
				UID        string `json:"uid"`
			} `json:"configRef"`
			DataSecretName string `json:"dataSecretName"`
		} `json:"bootstrap"`
		ClusterName       string `json:"clusterName"`
		FailureDomain     string `json:"failureDomain"`
		InfrastructureRef struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			Namespace  string `json:"namespace"`
			UID        string `json:"uid"`
		} `json:"infrastructureRef"`
		NodeDeletionTimeout string `json:"nodeDeletionTimeout"`
		ProviderID          string `json:"providerID"`
		Version             string `json:"version"`
	} `json:"spec"`
	Status struct {
		Addresses []struct {
			Address string `json:"address"`
			Type    string `json:"type"`
		} `json:"addresses"`
		BootstrapReady         bool      `json:"bootstrapReady"`
		CertificatesExpiryDate time.Time `json:"certificatesExpiryDate"`
		Conditions             []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		InfrastructureReady bool      `json:"infrastructureReady"`
		LastUpdated         time.Time `json:"lastUpdated"`
		NodeInfo            struct {
			Architecture            string `json:"architecture"`
			BootID                  string `json:"bootID"`
			ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
			KernelVersion           string `json:"kernelVersion"`
			KubeProxyVersion        string `json:"kubeProxyVersion"`
			KubeletVersion          string `json:"kubeletVersion"`
			MachineID               string `json:"machineID"`
			OperatingSystem         string `json:"operatingSystem"`
			OsImage                 string `json:"osImage"`
			SystemUUID              string `json:"systemUUID"`
		} `json:"nodeInfo"`
		NodeRef struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			UID        string `json:"uid"`
		} `json:"nodeRef"`
		ObservedGeneration int    `json:"observedGeneration"`
		Phase              string `json:"phase"`
	} `json:"status"`
}

type Verrazzano struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			KubectlKubernetesIoLastAppliedConfiguration string `json:"kubectl.kubernetes.io/last-applied-configuration"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		ResourceVersion   string    `json:"resourceVersion"`
		Uid               string    `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		Components struct {
		} `json:"components"`
		EnvironmentName string `json:"environmentName"`
		Profile         string `json:"profile"`
		Security        struct {
		} `json:"security"`
	} `json:"spec"`
	Status struct {
		Available  string `json:"available"`
		Components struct {
			Argocd struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"argocd"`
			CertManager struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"cert-manager"`
			CertManagerWebhookOci struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"cert-manager-webhook-oci"`
			ClusterApi struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"cluster-api"`
			ClusterIssuer struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"cluster-issuer"`
			CoherenceOperator struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"coherence-operator"`
			ExternalDns struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"external-dns"`
			FluentOperator struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"fluent-operator"`
			FluentbitOpensearchOutput struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"fluentbit-opensearch-output"`
			Fluentd struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"fluentd"`
			Grafana struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"grafana"`
			IngressController struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"ingress-controller"`
			Istio struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"istio"`
			JaegerOperator struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"jaeger-operator"`
			Keycloak struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"keycloak"`
			KialiServer struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"kiali-server"`
			KubeStateMetrics struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"kube-state-metrics"`
			Mysql struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"mysql"`
			MysqlOperator struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"mysql-operator"`
			OamKubernetesRuntime struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"oam-kubernetes-runtime"`
			Opensearch struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"opensearch"`
			OpensearchDashboards struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"opensearch-dashboards"`
			PrometheusAdapter struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"prometheus-adapter"`
			PrometheusNodeExporter struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"prometheus-node-exporter"`
			PrometheusOperator struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"prometheus-operator"`
			PrometheusPushgateway struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"prometheus-pushgateway"`
			Rancher struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"rancher"`
			RancherBackup struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"rancher-backup"`
			Thanos struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"thanos"`
			Velero struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"velero"`
			Verrazzano struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"verrazzano"`
			VerrazzanoApplicationOperator struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"verrazzano-application-operator"`
			VerrazzanoAuthproxy struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"verrazzano-authproxy"`
			VerrazzanoClusterAgent struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"verrazzano-cluster-agent"`
			VerrazzanoClusterOperator struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"verrazzano-cluster-operator"`
			VerrazzanoConsole struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"verrazzano-console"`
			VerrazzanoGrafanaDashboards struct {
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"verrazzano-grafana-dashboards"`
			VerrazzanoMonitoringOperator struct {
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"verrazzano-monitoring-operator"`
			VerrazzanoNetworkPolicies struct {
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
			} `json:"verrazzano-network-policies"`
			WeblogicOperator struct {
				Available  string `json:"available"`
				Conditions []struct {
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Message            string    `json:"message"`
					Status             string    `json:"status"`
					Type               string    `json:"type"`
				} `json:"conditions"`
				LastReconciledGeneration int    `json:"lastReconciledGeneration"`
				Name                     string `json:"name"`
				State                    string `json:"state"`
				Version                  string `json:"version"`
			} `json:"weblogic-operator"`
		} `json:"components"`
		Conditions []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Message            string    `json:"message"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		Instance struct {
			ConsoleUrl              string `json:"consoleUrl"`
			GrafanaUrl              string `json:"grafanaUrl"`
			KeyCloakUrl             string `json:"keyCloakUrl"`
			KialiUrl                string `json:"kialiUrl"`
			OpenSearchDashboardsUrl string `json:"openSearchDashboardsUrl"`
			OpenSearchUrl           string `json:"openSearchUrl"`
			PrometheusUrl           string `json:"prometheusUrl"`
			RancherUrl              string `json:"rancherUrl"`
		} `json:"instance"`
		State   string `json:"state"`
		Version string `json:"version"`
	} `json:"status"`
}
