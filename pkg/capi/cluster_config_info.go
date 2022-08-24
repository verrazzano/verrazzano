package capi

type ClusterConfigInfo struct {
	ClusterName    string
	Type           string
	ContainerImage string
}

var _ ClusterConfig = ClusterConfigInfo{}

func (r ClusterConfigInfo) GetClusterName() string {
	return r.ClusterName
}

func (r ClusterConfigInfo) GetType() string {
	return r.Type
}

func (r ClusterConfigInfo) GetContainerImage() string {
	return r.ContainerImage
}
