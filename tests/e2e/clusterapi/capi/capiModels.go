package capi

import "time"

type Cluster struct {
	ApiVersion string `json:"apiVersion"`
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
		Uid             string `json:"uid"`
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
			ApiVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			Namespace  string `json:"namespace"`
		} `json:"controlPlaneRef"`
		InfrastructureRef struct {
			ApiVersion string `json:"apiVersion"`
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
		CreationTimestamp          time.Time `json:"creationTimestamp"`
		DeletionGracePeriodSeconds int       `json:"deletionGracePeriodSeconds"`
		DeletionTimestamp          time.Time `json:"deletionTimestamp"`
		Finalizers                 []string  `json:"finalizers"`
		Generation                 int       `json:"generation"`
		Labels                     struct {
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
			Format            string `json:"format"`
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
			Reason             string    `json:"reason,omitempty"`
			Severity           string    `json:"severity,omitempty"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		ObservedGeneration  int    `json:"observedGeneration"`
		Replicas            int    `json:"replicas"`
		Selector            string `json:"selector"`
		UnavailableReplicas int    `json:"unavailableReplicas"`
		UpdatedReplicas     int    `json:"updatedReplicas"`
	} `json:"status"`
}

type Machine struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
		} `json:"annotations"`
		CreationTimestamp          time.Time `json:"creationTimestamp"`
		DeletionGracePeriodSeconds int       `json:"deletionGracePeriodSeconds"`
		DeletionTimestamp          time.Time `json:"deletionTimestamp"`
		Finalizers                 []string  `json:"finalizers"`
		Generation                 int       `json:"generation"`
		Labels                     struct {
			ClusterXK8SIoClusterName      string `json:"cluster.x-k8s.io/cluster-name"`
			ClusterXK8SIoControlPlane     string `json:"cluster.x-k8s.io/control-plane"`
			ClusterXK8SIoControlPlaneName string `json:"cluster.x-k8s.io/control-plane-name"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		OwnerReferences []struct {
			ApiVersion         string `json:"apiVersion"`
			BlockOwnerDeletion bool   `json:"blockOwnerDeletion"`
			Controller         bool   `json:"controller"`
			Kind               string `json:"kind"`
			Name               string `json:"name"`
			Uid                string `json:"uid"`
		} `json:"ownerReferences"`
		ResourceVersion string `json:"resourceVersion"`
		Uid             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		Bootstrap struct {
			ConfigRef struct {
				ApiVersion string `json:"apiVersion"`
				Kind       string `json:"kind"`
				Name       string `json:"name"`
				Namespace  string `json:"namespace"`
				Uid        string `json:"uid"`
			} `json:"configRef"`
			DataSecretName string `json:"dataSecretName"`
		} `json:"bootstrap"`
		ClusterName       string `json:"clusterName"`
		FailureDomain     string `json:"failureDomain"`
		InfrastructureRef struct {
			ApiVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			Namespace  string `json:"namespace"`
			Uid        string `json:"uid"`
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
		BootstrapReady bool `json:"bootstrapReady"`
		Conditions     []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Reason             string    `json:"reason,omitempty"`
			Severity           string    `json:"severity,omitempty"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		InfrastructureReady bool      `json:"infrastructureReady"`
		LastUpdated         time.Time `json:"lastUpdated"`
		ObservedGeneration  int       `json:"observedGeneration"`
		Phase               string    `json:"phase"`
	} `json:"status"`
}
