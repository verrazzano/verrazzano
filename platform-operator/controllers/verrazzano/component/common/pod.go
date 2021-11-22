package common

import (
	"bytes"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
)

var NewSPDYExecutor = remotecommand.NewSPDYExecutor
var FakeStdOut = ""

//ExecPod runs a remote command a pod, returning the stdout and stderr of the command.
func ExecPod(cfg *rest.Config, restClient rest.Interface, pod *v1.Pod, command []string) (string, string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	request := restClient.
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: command,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	executor, err := NewSPDYExecutor(cfg, "POST", request.URL())
	if err != nil {
		return "", "", err
	}
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return "", "", errors.New(fmt.Sprintf("error running command %s on %v/%v: %v", command, pod.Namespace, pod.Name, err))
	}

	return stdout.String(), stderr.String(), nil
}

func FakeNewSPDYExecutor(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	return &fakeExecutor{method: method, url: url}, nil
}

type fakeExecutor struct {
	method string
	stdout string
	url    *url.URL
}

func (f *fakeExecutor) Stream(options remotecommand.StreamOptions) error {
	if options.Stdout != nil {
		buf := new(bytes.Buffer)
		buf.WriteString(FakeStdOut)
		if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}
