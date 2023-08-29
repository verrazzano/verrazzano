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
		Topology struct {
			Class        string `json:"class"`
			ControlPlane struct {
				Metadata struct {
				} `json:"metadata"`
				Replicas int `json:"replicas"`
			} `json:"controlPlane"`
			Variables []struct {
				Name  string      `json:"name"`
				Value interface{} `json:"value"`
			} `json:"variables"`
			Version string `json:"version"`
			Workers struct {
				MachineDeployments []struct {
					Class    string `json:"class"`
					Metadata struct {
					} `json:"metadata"`
					Name     string `json:"name"`
					Replicas int    `json:"replicas"`
				} `json:"machineDeployments"`
			} `json:"workers"`
		} `json:"topology"`
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
		VerrazzanoPlatformOperator struct {
			Enabled bool `json:"enabled"`
		} `json:"verrazzanoPlatformOperator"`
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
			Reason             string    `json:"reason"`
			Severity           string    `json:"severity"`
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
	APIVersion string `json:"apiVersion"`
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
		UID               string    `json:"uid"`
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
		Conditions []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Message            string    `json:"message"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		Instance struct {
			ConsoleURL              string `json:"consoleUrl"`
			GrafanaURL              string `json:"grafanaUrl"`
			KeyCloakURL             string `json:"keyCloakUrl"`
			KialiURL                string `json:"kialiUrl"`
			OpenSearchDashboardsURL string `json:"openSearchDashboardsUrl"`
			OpenSearchURL           string `json:"openSearchUrl"`
			PrometheusURL           string `json:"prometheusUrl"`
			RancherURL              string `json:"rancherUrl"`
		} `json:"instance"`
		State   string `json:"state"`
		Version string `json:"version"`
	} `json:"status"`
}
