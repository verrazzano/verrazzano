// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	"io"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sYaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"os"
	"os/exec"
	"strings"
	"time"
)

var decUnstructured = k8sYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

type BashCommand struct {
	Timeout     time.Duration `json:"timeout"`
	CommandArgs []string      `json:"cmdArgs"`
}

type RunnerResponse struct {
	StandardOut  bytes.Buffer `json:"stdout"`
	StandardErr  bytes.Buffer `json:"stderr"`
	CommandError error        `json:"error"`
}

func Runner(bcmd *BashCommand, log *zap.SugaredLogger) *RunnerResponse {
	var stdoutBuf, stderrBuf bytes.Buffer
	var bashCommandResponse RunnerResponse
	bashCommand := exec.Command(bcmd.CommandArgs[0], bcmd.CommandArgs[1:]...) //nolint:gosec
	bashCommand.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	bashCommand.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	log.Infof("Executing command '%v'", bashCommand.String())
	err := bashCommand.Start()
	if err != nil {
		log.Errorf("Cmd '%v' execution failed due to '%v'", bashCommand.String(), zap.Error(err))
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	}
	done := make(chan error, 1)
	go func() {
		done <- bashCommand.Wait()
	}()
	select {
	case <-time.After(bcmd.Timeout):
		if err = bashCommand.Process.Kill(); err != nil {
			log.Errorf("Failed to kill cmd '%v' due to '%v'", bashCommand.String(), zap.Error(err))
			bashCommandResponse.CommandError = err
			return &bashCommandResponse
		}
		log.Errorf("Cmd '%v' timeout expired", bashCommand.String())
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	case err = <-done:
		if err != nil {
			log.Errorf("Cmd '%v' execution failed due to '%v'", bashCommand.String(), zap.Error(err))
			bashCommandResponse.StandardErr = stderrBuf
			bashCommandResponse.CommandError = err
			return &bashCommandResponse
		}
		log.Debugf("Command '%s' execution successful", bashCommand.String())
		bashCommandResponse.StandardOut = stdoutBuf
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	}
}

func GetEsURL(log *zap.SugaredLogger) (string, error) {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "vz")
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "jsonpath={.items[].status.instance.elasticUrl}")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs

	bashResponse := Runner(&kcmd, log)
	if bashResponse.CommandError != nil {
		return "", bashResponse.CommandError
	}
	return bashResponse.StandardOut.String(), nil
}

func GetVZPasswd(log *zap.SugaredLogger) (string, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return "", err
	}

	secret, err := clientset.CoreV1().Secrets(constants.VerrazzanoSystemNamespace).Get(context.TODO(), "verrazzano", metav1.GetOptions{})
	if err != nil {
		log.Infof("Error creating secret ", zap.Error(err))
		return "", err
	}
	return string(secret.Data["password"]), nil
}

func dynamicSSA(ctx context.Context, deploymentYAML string, log *zap.SugaredLogger) error {

	kubeconfig, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Error getting kubeconfig, error: %v", err)
		return err
	}

	// Prepare a RESTMapper to find GVR followed by creating the dynamic client
	dc, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	dynamicClient, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}

	// Convert to unstructured since this will be used for CRDS
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode([]byte(deploymentYAML), nil, obj)
	if err != nil {
		return err
	}
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// Create a dynamic REST interface
	var dynamicRest dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dynamicRest = dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dynamicRest = dynamicClient.Resource(mapping.Resource)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// Apply the Yaml
	_, err = dynamicRest.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "backup-controller",
	})
	return err
}

func retryAndCheckShellCommandResponse(retryLimit int, bcmd *BashCommand, operation, objectName string, log *zap.SugaredLogger) error {
	retryCount := 0
	for {
		if retryCount >= retryLimit {
			return fmt.Errorf("retry count execeeded while checking progress for %s '%s'", operation, objectName)
		}
		bashResponse := Runner(bcmd, log)
		if bashResponse.CommandError != nil {
			return bashResponse.CommandError
		}
		response := strings.TrimSpace(strings.Trim(bashResponse.StandardOut.String(), "\n"))
		switch response {
		case "InProgress":
			log.Infof("%s '%s' is in progress. Check back after 20 seconds", strings.ToTitle(operation), objectName)
			time.Sleep(20 * time.Second)
		case "Completed":
			log.Infof("%s '%s' completed successfully", strings.ToTitle(operation), objectName)
			return nil
		default:
			return fmt.Errorf("%s failed. State = '%s'", strings.ToTitle(operation), response)
		}
		retryCount = retryCount + 1
	}

}

func veleroObjectDelete(objectType, objectname, nameSpaceName string, log *zap.SugaredLogger) error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, nameSpaceName)
	cmdArgs = append(cmdArgs, "delete")

	switch objectType {
	case "backup":
		cmdArgs = append(cmdArgs, "backup.velero.io")
	case "restore":
		cmdArgs = append(cmdArgs, "restore.velero.io")
	case "storage":
		cmdArgs = append(cmdArgs, "backupstoragelocation.velero.io")
	}
	cmdArgs = append(cmdArgs, objectname)

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	bashResponse := Runner(&kcmd, log)
	if bashResponse.CommandError != nil {
		return bashResponse.CommandError
	}
	return nil
}

func displayHookLogs(log *zap.SugaredLogger) error {
	log.Infof("Retrieving verrazzano hook logs ...")
	var cmdArgs []string
	logFileCmd := "kubectl exec -it -n verrazzano-system  vmi-system-es-master-0 -- ls -alt --time=ctime /tmp/ | grep verrazzano | cut -d ' ' -f9 | head -1"
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, logFileCmd)

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	bashResponse := Runner(&kcmd, log)
	logFileName := strings.TrimSpace(strings.Trim(bashResponse.StandardOut.String(), "\n"))

	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Failed to get config with error: %v", err)
		return err
	}

	podSpec, err := clientset.CoreV1().Pods(constants.VerrazzanoSystemNamespace).Get(context.TODO(), "vmi-system-es-master-0", metav1.GetOptions{})
	if err != nil {
		return err
	}

	var execCmd []string
	execCmd = append(execCmd, "cat")
	execCmd = append(execCmd, fmt.Sprintf("/tmp/%s", logFileName))
	stdout, _, err := k8sutil.ExecPod(clientset, config, podSpec, "es-master", execCmd)
	if err != nil {
		return err
	}
	log.Infof(stdout)
	return nil
}
