// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utility

import (
	"context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
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

var ReportAnalysis = make(map[string][]string)

// PatchImage patches a deployment's image and feeds cluster analysis report
// patching includes both injection of an issue and its revival
func PatchImage(client *kubernetes.Clientset, deploymentName, issueType, patchImage string) error {
	deploymentClient := client.AppsV1().Deployments(VzSystemNS)
	for i := 0; i < 2; i++ {
		if err := update(deploymentClient, deploymentName, issueType, patchImage); err != nil {
			return err
		}
		r, err := RunVzAnalyze()
		if err != nil {
			return err
		}
		ReportAnalysis[issueType] = append(ReportAnalysis[issueType], r)
		patchImage = ""
		time.Sleep(waitTimeout)
	}
	return nil
}

func update(deploymentClient kv1.DeploymentInterface, deploymentName, issueType, patchImage string) error {
	result, err := deploymentClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
	if err != nil {
		return err
	}
	for i, container := range result.Spec.Template.Spec.Containers {
		if container.Name == deploymentName {
			image := result.Spec.Template.Spec.Containers[i].Image
			if patchImage == "" {
				patchImage = image
			} else if issueType == ImagePullNotFound {
				patchImage = image + patchImage
			}
			result.Spec.Template.Spec.Containers[i].Image = patchImage
			_, err = deploymentClient.Update(context.TODO(), result, v1.UpdateOptions{})
			time.Sleep(time.Second * 20)
			break
		}
	}
	return err
}

// PatchPod patches a deployment's pod and feeds cluster analysis report
// patching includes both injection of an issue and its revival
func PatchPod(issueType string, resourceReq []string) error {
	for i := 0; i < len(resourceReq); i++ {
		_, err := SetDepResources(DeploymentToBePatched, VzSystemNS, resourceReq[i])
		if err != nil {
			return err
		}
		time.Sleep(waitTimeout)
		out, err := RunVzAnalyze()
		if err != nil {
			return err
		}
		ReportAnalysis[issueType] = append(ReportAnalysis[issueType], out)
		if i == 0 {
			time.Sleep(time.Second * 20)
		}
	}
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
