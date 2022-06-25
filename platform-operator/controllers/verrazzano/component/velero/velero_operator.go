// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	"path"
	"time"
)

const (
	deploymentName = "velero"

	veleroBin            = "velero"
	installCli           = "install"
	volSnapshotEnableCli = "--use-volume-snapshots=false"
	pluginImageCli       = "--plugins"
	veleroImageCli       = "--image"
	resticCli            = "--use-restic"
	nosecret             = "--no-secret"
	noDefaultBackup      = "--no-default-backup-location"
	veleroPodCPURequest  = "--velero-pod-cpu-request=500m"
	veleroPodCPULimit    = "--velero-pod-cpu-limit=1000m"
	veleroPodMemRequest  = "--velero-pod-mem-request=128Mi"
	veleroPodMemLimit    = "--velero-pod-mem-limit=512Mi"
	resticPodCPURequest  = "--restic-pod-cpu-request=500m"
	resticPodCPULimit    = "--restic-pod-cpu-limit=1000m"
	resticPodMemRequest  = "--restic-pod-mem-request=128Mi"
	resticPodMemLimit    = "--restic-pod-mem-limit=512Mi"
)

var subcomponentNames = []string{
	"velero",
	"velero-plugin-for-aws",
	"velero-restic-restore-helper",
}

func componentInstall(ctx spi.ComponentContext) error {
	args, err := buildInstallArgs()
	if err != nil {
		ctx.Log().Errorf("Unable to build installargs %v", zap.Error(err))
		return err
	}
	var vcmd BashCommand
	vcmd.Timeout = time.Second * 600

	var bcmd []string
	bcmd = append(bcmd, veleroBin, installCli)
	bcmd = append(bcmd, veleroImageCli, args.VeleroImage)
	bcmd = append(bcmd, pluginImageCli, args.VeleroPluginForAwsImage)
	bcmd = append(bcmd, resticCli, volSnapshotEnableCli, nosecret, noDefaultBackup)
	bcmd = append(bcmd, veleroPodCPURequest, veleroPodCPULimit, veleroPodMemRequest, veleroPodMemLimit)
	bcmd = append(bcmd, resticPodCPURequest, resticPodCPULimit, resticPodMemRequest, resticPodMemLimit)
	vcmd.CommandArgs = bcmd

	veleroInstallResponse := genericRunner(&vcmd, ctx.Log())
	if veleroInstallResponse.CommandError != nil {
		return ctx.Log().ErrorfNewErr("Failed to install Velero Operator: %v", veleroInstallResponse.CommandError)
	}

	// Create configmap for velero restic helper
	resticHelperYamlArgs := make(map[string]interface{})
	resticHelperYamlArgs["veleroNamespace"] = ComponentNamespace
	resticHelperYamlArgs["veleroRunnerImage"] = args.VeleroResticRestoreHelperImage

	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), ComponentNamespace)
	if err := yamlApplier.ApplyFT(path.Join(config.GetThirdPartyManifestsDir(), resticConfigmapFile), resticHelperYamlArgs); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create configmap for velero restic helper: %v", err)
	}

	ctx.Log().Infof("%v", veleroInstallResponse.StandardOut.String())
	return nil
}

func buildInstallArgs() (VeleroImage, error) {
	args := map[string]interface{}{
		"namespace": constants.VeleroNameSpace,
	}
	var vi VeleroImage
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return vi, err
	}
	for _, subcomponent := range subcomponentNames {
		if err := setImageOverride(args, bomFile, subcomponent); err != nil {
			return vi, err
		}
	}
	dbin, err := json.Marshal(args)
	if err != nil {
		return vi, err
	}

	err = json.Unmarshal(dbin, &vi)
	if err != nil {
		return vi, err
	}

	return vi, nil
}

func setImageOverride(args map[string]interface{}, bomFile bom.Bom, subcomponent string) error {
	images, err := bomFile.GetImageNameList(subcomponent)
	if err != nil {
		return err
	}
	if len(images) != 1 {
		return fmt.Errorf("expected 1 %s image, got %d", subcomponent, len(images))
	}

	args[subcomponent] = images[0]
	return nil
}

// isVeleroOperatorReady checks if the Velero operator deployment is ready
func isVeleroOperatorReady(context spi.ComponentContext) bool {
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix)
}
