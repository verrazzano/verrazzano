// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"bytes"
	"io/ioutil"
	"os"
	"text/template"

	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kind "sigs.k8s.io/kind/pkg/cmd"
)

type kindBootstrapProviderImpl struct{}

func (k *kindBootstrapProviderImpl) CreateCluster(config ClusterConfig) error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))

	bootstrapConfig, err := parseKindBoostrapConfig(config)
	if err != nil {
		return err
	}
	//fmt.Println(fmt.Sprintf("%s", bootstrapConfig))
	return provider.Create(config.GetClusterName(), kindcluster.CreateWithRawConfig(bootstrapConfig))
}

func (k *kindBootstrapProviderImpl) DestroyCluster(config ClusterConfig) error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	kubeconfig, err := k.GetKubeconfig(config)
	if err != nil {
		return err
	}
	kubePath, err := saveKubeconfigToFile(kubeconfig)
	if err != nil {
		return err
	}
	defer func() {
		os.Remove(kubePath)
	}()
	return provider.Delete(config.GetClusterName(), kubePath)
}

func saveKubeconfigToFile(kubeconfigContents string) (string, error) {
	// Create a temp file that contains the kubeconfig
	tmpFile, err := ioutil.TempFile("", "kubeconfig")
	if err != nil {
		return "", err
	}
	kubeConfigBytes := bytes.Buffer{}
	kubeConfigBytes.WriteString(kubeconfigContents)
	err = ioutil.WriteFile(tmpFile.Name(), kubeConfigBytes.Bytes(), 0600)
	return tmpFile.Name(), err
}

func (k *kindBootstrapProviderImpl) GetKubeconfig(config ClusterConfig) (string, error) {
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return "", nil
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	return provider.KubeConfig(config.GetClusterName(), false)
}

func parseKindBoostrapConfig(config ClusterConfig) ([]byte, error) {
	kindBoostrapConfig := getDefaultBoostrapKindConfig(config.GetType())
	data := templateData{BootstrapNodeImage: config.GetContainerImage()}
	var b bytes.Buffer
	t, err := template.New("boostrapConfig").Parse(kindBoostrapConfig)
	if err != nil {
		return []byte{}, err
	}
	if err := t.Execute(&b, &data); err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

func getDefaultBoostrapImage(clusterType string) string {
	bootstrapImageOverride, envOverrideFound := os.LookupEnv(BootstrapImageEnvVar)
	if envOverrideFound {
		return bootstrapImageOverride
	}
	switch clusterType {
	case KindClusterType:
		return defaultKindBootstrapNodeImage
	case OCNEClusterType:
		return defaultCNEBootstrapNodeImage
	default:
		return ""
	}
}

func getDefaultBoostrapKindConfig(clusterType string) string {
	switch clusterType {
	case KindClusterType:
		return defaultKindBootstrapConfig
	}
	return defaultCNEBootstrapConfig
}

func createKubeConfigFile(clcm ClusterLifeCycleManager) (*os.File, error) {
	kcFile, err := ioutil.TempFile(os.TempDir(), "kubeconfig-"+clcm.GetConfig().GetClusterName())
	if err != nil {
		return nil, err
	}
	config, err := clcm.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	if _, err := kcFile.Write([]byte(config)); err != nil {
		return nil, err
	}
	return kcFile, nil
}
