// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utility

import (
	"context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	waitTimeout = 10 * time.Second
)

const (
	ImagePullNotFound      string = "ImagePullNotFound"
	ImagePullBackOff       string = "ImagePullBackOff"
	PodProblemsNotReported string = "PodProblemsNotReported"
	PendingPods            string = "PendingPods"
	InsufficientMemory     string = "InsufficientMemory"
	VzSystemNS             string = "verrazzano-system"
	DeploymentToBePatched  string = "verrazzano-console"
)

type Action struct {
	Patch  string
	Revive string
}

var ReportAnalysis = make(map[string]Action)

// PatchImage patches a deployment's image and feeds cluster analysis report
// patching includes both injection of an issue and its revival
func PatchImage(client *kubernetes.Clientset, deploymentName, issueType, patchImage string) error {
	deploymentClient := client.AppsV1().Deployments(VzSystemNS)
	result, getErr := deploymentClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	for i, container := range result.Spec.Template.Spec.Containers {
		if container.Name == deploymentName {
			image := result.Spec.Template.Spec.Containers[i].Image
			// PATCHING
			if issueType == ImagePullNotFound {
				patchImage = image + patchImage
			}
			result.Spec.Template.Spec.Containers[i].Image = patchImage
			_, updateErr := deploymentClient.Update(context.TODO(), result, v1.UpdateOptions{})
			if updateErr != nil {
				return updateErr
			}
			time.Sleep(waitTimeout)
			out1, err := RunVzAnalyze()
			if err != nil {
				return err
			}
			time.Sleep(time.Second * 20)
			result, getErr = deploymentClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			// REVIVING
			result.Spec.Template.Spec.Containers[i].Image = image
			_, updateErr = deploymentClient.Update(context.TODO(), result, v1.UpdateOptions{})
			if updateErr != nil {
				return updateErr
			}
			time.Sleep(waitTimeout)
			out2, err := RunVzAnalyze()
			if err != nil {
				return err
			}
			ReportAnalysis[issueType] = Action{out1, out2}
			break
		}
	}
	return nil
}

// PatchPod patches a deployment's pod and feeds cluster analysis report
// patching includes both injection of an issue and its revival
func PatchPod(issueType string, resourceReq []string) error {
	out := make([]string, 2)
	for i := 0; i < len(resourceReq); i++ {
		_, err := SetDepResources(DeploymentToBePatched, VzSystemNS, resourceReq[i])
		if err != nil {
			return err
		}
		time.Sleep(waitTimeout)
		out[i], err = RunVzAnalyze()
		if err != nil {
			return err
		}
		if i == 0 {
			time.Sleep(time.Second * 20)
		}
	}
	ReportAnalysis[issueType] = Action{out[0], out[1]}
	return nil
}

// RunVzAnalyze runs and deliver cluster analysis report
func RunVzAnalyze() (string, error) {
	cmd := exec.Command("./vz", "analyze")
	if goRepoPath := os.Getenv("GO_REPO_PATH"); goRepoPath != "" {
		cmd.Dir = goRepoPath
	}
	out, err := cmd.Output()
	return string(out), err
}

// SetDepResources sets pod's resources i.e (cpu/ memory)
func SetDepResources(dep, ns, req string) (string, error) {
	args := []string{"set", "resources", "deploy/" + dep, "--requests=" + req, "-n", ns}
	out, err := exec.Command("kubectl", args...).Output()
	return string(out), err
}

// VerifyIssue verifies issue against cluster analysis report
func VerifyIssue(out, issueType string) bool {
	return strings.Contains(out, issueType)
}
