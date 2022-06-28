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
	v1 "k8s.io/api/core/v1"
	"os"
	"path"
	"strings"
	"time"
)

const (
	deploymentName = "velero"

	veleroBin                 = "velero"
	installCli                = "install"
	volSnapshotEnableCli      = "--use-volume-snapshots=false"
	pluginImageCli            = "--plugins"
	veleroImageCli            = "--image"
	resticCli                 = "--use-restic"
	nosecret                  = "--no-secret"
	noDefaultBackup           = "--no-default-backup-location"
	veleroPodCPURequestCli    = "--velero-pod-cpu-request="
	veleroPodCPULimitCli      = "--velero-pod-cpu-limit="
	veleroPodMemoryRequestCli = "--velero-pod-mem-request="
	veleroPodMemoryLimitCli   = "--velero-pod-mem-limit="
	resticPodCPURequestCli    = "--restic-pod-cpu-request="
	resticPodCPULimitCli      = "--restic-pod-cpu-limit="
	resticPodMemoryRequestCli = "--restic-pod-mem-request="
	resticPodMemoryLimitCli   = "--restic-pod-mem-limit="
)

var subcomponentNames = []string{
	"velero",
	"velero-plugin-for-aws",
	"velero-restic-restore-helper",
}

func getDefaultValues(resourceCategory, resourceType string) string {
	switch resourceCategory {
	case "limits":
		switch resourceType {
		case "cpu":
			return "200m"
		case "memory":
			return "128Mi"
		}
	case "requests":
		switch resourceType {
		case "cpu":
			return "200m"
		case "memory":
			return "128Mi"
		}
	}
	return ""
}

func getValueFromResourceReqs(rq *v1.ResourceRequirements, resourceCategory, resourceType string) string {
	switch strings.ToLower(resourceCategory) {
	case "limits":
		if rq.Limits != nil {
			switch resourceType {
			case "cpu":
				if rq.Limits.Cpu() != nil {
					return rq.Limits.Cpu().String()
				}
				return getDefaultValues(resourceCategory, resourceType)

			case "memory":
				if rq.Limits.Memory() != nil {
					return rq.Limits.Memory().String()
				}
				return getDefaultValues(resourceCategory, resourceType)
			}
		}
		return getDefaultValues(resourceCategory, resourceType)
	case "requests":
		if rq.Requests != nil {
			switch resourceType {
			case "cpu":
				if rq.Requests.Cpu() != nil {
					return rq.Requests.Cpu().String()
				}
				return getDefaultValues(resourceCategory, resourceType)
			case "memory":
				if rq.Requests.Memory() != nil {
					return rq.Requests.Memory().String()
				}
				return getDefaultValues(resourceCategory, resourceType)
			}
		}
		return getDefaultValues(resourceCategory, resourceType)
	}
	return ""
}

func getCRValue(ctx spi.ComponentContext, resourceName, resourceCategory, resourceType string) string {
	switch resourceName {
	case "velero":
		if ctx.EffectiveCR().Spec.Components.Velero.Kubernetes != nil {
			if ctx.EffectiveCR().Spec.Components.Velero.Kubernetes.Resources != nil {
				rq := ctx.EffectiveCR().Spec.Components.Velero.Kubernetes.Resources
				return getValueFromResourceReqs(rq, resourceCategory, resourceType)
			}
			ctx.Log().Infof("calling default for velero for %v type %v", resourceCategory, resourceType)
			return getDefaultValues(resourceCategory, resourceType)
		}
		ctx.Log().Infof("calling default for velero for %v type %v", resourceCategory, resourceType)
		return getDefaultValues(resourceCategory, resourceType)

	case "restic":
		if ctx.EffectiveCR().Spec.Components.Velero.Restic != nil {
			if ctx.EffectiveCR().Spec.Components.Velero.Restic.Kubernetes != nil {
				if ctx.EffectiveCR().Spec.Components.Velero.Restic.Kubernetes.Resources != nil {
					rq := ctx.EffectiveCR().Spec.Components.Velero.Restic.Kubernetes.Resources
					return getValueFromResourceReqs(rq, resourceCategory, resourceType)
				}
				ctx.Log().Infof("calling default for restic for %v type %v", resourceCategory, resourceType)
				return getDefaultValues(resourceCategory, resourceType)
			}
			ctx.Log().Infof("calling default for restic for %v type %v", resourceCategory, resourceType)
			return getDefaultValues(resourceCategory, resourceType)
		}
		ctx.Log().Infof("calling default for restic for %v type %v", resourceCategory, resourceType)
		return getDefaultValues(resourceCategory, resourceType)

	}
	return ""
}

func componentInstall(ctx spi.ComponentContext) error {

	args, err := buildInstallArgs()
	if err != nil {
		ctx.Log().Errorf("Unable to build installargs %v", zap.Error(err))
		return err
	}
	var vcmd bashCommand
	var veleroInstallResponse *runnerResponse
	vcmd.Timeout = time.Second * 600
	veleroCPURequestCmd := fmt.Sprintf("%s%s", veleroPodCPURequestCli, getCRValue(ctx, "velero", "requests", "cpu"))
	veleroCPULimitCmd := fmt.Sprintf("%s%s", veleroPodCPULimitCli, getCRValue(ctx, "velero", "limits", "cpu"))
	veleroMemRequestCmd := fmt.Sprintf("%s%s", veleroPodMemoryRequestCli, getCRValue(ctx, "velero", "requests", "memory"))
	veleroMemLimitCmd := fmt.Sprintf("%s%s", veleroPodMemoryLimitCli, getCRValue(ctx, "velero", "limits", "memory"))

	resticCPURequestCmd := fmt.Sprintf("%s%s", resticPodCPURequestCli, getCRValue(ctx, "restic", "requests", "cpu"))
	resticCPULimitCmd := fmt.Sprintf("%s%s", resticPodCPULimitCli, getCRValue(ctx, "restic", "limits", "cpu"))
	resticMemRequestCmd := fmt.Sprintf("%s%s", resticPodMemoryRequestCli, getCRValue(ctx, "restic", "requests", "memory"))
	resticMemLimitCmd := fmt.Sprintf("%s%s", resticPodMemoryLimitCli, getCRValue(ctx, "restic", "limits", "memory"))

	var bcmd []string
	bcmd = append(bcmd, veleroBin, installCli)
	bcmd = append(bcmd, veleroImageCli, args.VeleroImage)
	bcmd = append(bcmd, pluginImageCli, args.VeleroPluginForAwsImage)
	bcmd = append(bcmd, resticCli, volSnapshotEnableCli, nosecret, noDefaultBackup)
	bcmd = append(bcmd, veleroCPURequestCmd, veleroCPULimitCmd, veleroMemRequestCmd, veleroMemLimitCmd)
	bcmd = append(bcmd, resticCPURequestCmd, resticCPULimitCmd, resticMemRequestCmd, resticMemLimitCmd)
	vcmd.CommandArgs = bcmd

	if os.Getenv("DEV_TEST") != "True" {
		veleroInstallResponse = genericRunner(&vcmd, ctx.Log())
		if veleroInstallResponse.CommandError != nil {
			return ctx.Log().ErrorfNewErr("Failed to install Velero Operator: %v", veleroInstallResponse.CommandError)
		}
		ctx.Log().Infof("%v", veleroInstallResponse.StandardOut.String())
	}

	// Create configmap for Velero Restic helper
	resticHelperYamlArgs := make(map[string]interface{})
	resticHelperYamlArgs["veleroNamespace"] = ComponentNamespace
	resticHelperYamlArgs["veleroRunnerImage"] = args.VeleroResticRestoreHelperImage

	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), ComponentNamespace)
	if err := yamlApplier.ApplyFT(path.Join(config.GetThirdPartyManifestsDir(), resticConfigmapFile), resticHelperYamlArgs); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create configmap for velero restic helper: %v", err)
	}

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
