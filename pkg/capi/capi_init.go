// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	"go.uber.org/zap"
	"os"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

func initializeCAPI(clcm ClusterLifeCycleManager) error {
	config, err := createKubeConfigFile(clcm)
	if err != nil {
		return err
	}
	defer os2.RemoveTempFiles(zap.S(), config.Name())
	capiclient, err := capiInitFunc("")
	if err != nil {
		return err
	}
	_, err = capiclient.Init(clusterapi.InitOptions{
		Kubeconfig: clusterapi.Kubeconfig{
			Path: config.Name(),
		},
		InfrastructureProviders: clcm.GetConfig().CAPIProviders,
	})
	return err
}

func createKubeConfigFile(clcm ClusterLifeCycleManager) (*os.File, error) {
	kcFile, err := os.CreateTemp(os.TempDir(), "kubeconfig-"+clcm.GetConfig().ClusterName)
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
