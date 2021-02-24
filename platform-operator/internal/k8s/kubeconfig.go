package k8s

// KubeConfig represents a kubeconfig object used to connect using a token
type KubeConfig struct {
	Clusters       []kcCluster `json:"clusters"`
	Users          []kcUser    `json:"users"`
	Contexts       []kcContext `json:"contexts"`
	CurrentContext string      `json:"current-context"`
}
type kcCluster struct {
	Name    string        `json:"name"`
	Cluster kcClusterData `json:"cluster"`
}
type kcClusterData struct {
	Server   string `json:"server"`
	CertAuth string `json:"certificate-authority-data"`
}
type kcUser struct {
	Name string     `json:"name"`
	User kcUserData `json:"user"`
}
type kcUserData struct {
	Token string `json:"token"`
}
type kcContext struct {
	Name    string        `json:"name"`
	Context kcContextData `json:"context"`
}
type kcContextData struct {
	User    string `json:"user"`
	Cluster string `json:"cluster"`
}

type KubeconfigBuilder struct {
	ClusterName string
	Server      string
	CertAuth    string
	UserName    string
	UserToken   string
	ContextName string
}

// New creates a KubeConfig object using the fields of the builder
func (b *KubeconfigBuilder) New() KubeConfig {
	return KubeConfig{
		Clusters: []kcCluster{{
			Name: b.ClusterName,
			Cluster: kcClusterData{
				Server:   b.Server,
				CertAuth: b.CertAuth,
			},
		}},
		Users: []kcUser{{
			Name: b.UserName,
			User: kcUserData{
				Token: b.UserToken,
			},
		}},
		Contexts: []kcContext{{
			Name: b.ContextName,
			Context: kcContextData{
				User:    b.UserName,
				Cluster: b.ClusterName,
			},
		}},
		CurrentContext: b.ContextName,
	}
}
