package capi

type ClusterConfigInfo struct {
	ClusterNameVal    string
	TypeVal           string
	ContainerImageVal string
}

var _ ClusterConfig = ClusterConfigInfo{}

func (r ClusterConfigInfo) ClusterName() string {
	return r.ClusterNameVal
}

func (r ClusterConfigInfo) Type() string {
	return r.TypeVal
}

func (r ClusterConfigInfo) ContainerImage() string {
	return r.ContainerImageVal
}
