package verify_analyze_tool

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

// PatchPod This Method implements the patching of bad image & its revival
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

// RunVzAnalyze utility method to run vz analyze and deliver its report
func RunVzAnalyze() (string, error) {
	cmd := exec.Command("./vz", "analyze")
	if goRepoPath := os.Getenv("GO_REPO_PATH"); goRepoPath != "" {
		cmd.Dir = goRepoPath
	}
	out, err := cmd.Output()
	return string(out), err
}

// SetDepResources utility function to set deployment pod's resources (cpu/ memory)
func SetDepResources(dep, ns, req string) (string, error) {
	args := []string{"set", "resources", "deploy/" + dep, "--requests=" + req, "-n", ns}
	out, err := exec.Command("kubectl", args...).Output()
	return string(out), err
}

// VerifyIssue utility method to verify issue into vz analyze report
func VerifyIssue(out, issueType string) bool {
	return strings.Contains(out, issueType)
}
