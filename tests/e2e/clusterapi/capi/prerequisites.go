// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"encoding/base64"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func (c CAPITestImpl) ProcessOCIPrivateKeysBase64(file, key string, log *zap.SugaredLogger) error {
	_, err := os.Stat(file)
	if err != nil {
		log.Errorf("file '%s' not found", file)
		return err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		log.Errorf("failed reading file contents: %v", err)
		return err
	}

	return os.Setenv(key, base64.StdEncoding.EncodeToString(data))
}

func (c CAPITestImpl) ProcessOCISSHKeys(file, key string, log *zap.SugaredLogger) error {
	_, err := os.Stat(file)
	if err != nil {
		log.Errorf("file '%s' not found", file)
		return err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		log.Errorf("failed reading file contents: %v", err)
		return err
	}

	return os.Setenv(key, string(data))
}

func (c CAPITestImpl) ProcessOCIPrivateKeysSingleLine(file, key string, log *zap.SugaredLogger) error {
	_, err := os.Stat(file)
	if err != nil {
		log.Errorf("file '%s' not found", file)
		return err
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), "testkey")
	if err != nil {
		log.Errorf("Failed to create temporary file : %v", zap.Error(err))
		return err
	}

	var cmdArgs []string
	var bcmd helpers.BashCommand
	ocicmd := "awk 'NF {sub(/\\r/, \"\"); printf \"%s\\\\n\",$0;}' " + file + "> " + tmpFile.Name()
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	bcmd.CommandArgs = cmdArgs
	keydata := helpers.Runner(&bcmd, log)
	if keydata.CommandError != nil {
		return keydata.CommandError
	}

	bdata, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		log.Errorf("failed reading file contents: %v", err)
		return err
	}

	return os.Setenv(key, string(bdata))
}

func (c CAPITestImpl) CreateNamespace(namespace string, log *zap.SugaredLogger) error {
	log.Infof("creating namespace '%s'", namespace)
	k8s, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}

	nsObj := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = k8s.CoreV1().Namespaces().Create(context.TODO(), nsObj, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("failed to create namespace %v", zap.Error(err))
		return err
	}
	return nil

}

func (c CAPITestImpl) SetImageID(key string, log *zap.SugaredLogger) error {
	oci, err := NewClient(GetOCIConfigurationProvider(log))
	if err != nil {
		log.Error("Unable to create OCI client %v", zap.Error(err))
		return err
	}
	id, err := oci.GetImageIDByName(context.TODO(), OCICompartmentID, OracleLinuxDisplayName, OperatingSystem, OperatingSystemVersion, log)
	if err != nil {
		log.Error("Unable to fetch image id %v", zap.Error(err))
		return err
	}
	return os.Setenv(key, id)
}
