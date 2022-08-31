// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/capi"
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const nameFlag = "--" + constants.ClusterNameFlagName
const typeFlag = "--" + constants.ClusterTypeFlagName
const kubePathFlag = "--" + constants.KubeconfigPathFlagName

func TestClusterArgs(t *testing.T) {
	tempKubeconfig := path.Join(os.TempDir(), fmt.Sprintf("somefile%d", time.Now().UnixNano()))
	defer func() { os.Remove(tempKubeconfig) }()
	tests := []struct {
		name       string
		subcommand string
		args       []string
		expectErr  bool
	}{
		// capi.NoClusterType is used in all testing so that we don't trigger an actual cluster create/delete
		{"cluster create with default name", createSubCommandName, []string{typeFlag, capi.NoClusterType}, false},
		{"cluster delete with default name", deleteSubCommandName, []string{typeFlag, capi.NoClusterType}, false},
		{"cluster get-kubeconfig with default name and path", getKubeconfigSubCommandName, []string{kubePathFlag, tempKubeconfig, typeFlag, capi.NoClusterType}, false},
		{"cluster with nonexistent subcommand", "nonexistent", []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(tt.subcommand, tt.args)
			if tt.expectErr {
				asserts.Error(t, err)
				return
			}
			asserts.NoError(t, err)
		})
	}
}

func TestClusterCreateOptions(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{"cluster create with custom name and type", []string{nameFlag, "mycluster", typeFlag, capi.NoClusterType}, false},
		{"cluster create with custom name, type and image", []string{nameFlag, "mycluster", typeFlag, capi.NoClusterType, "--image", "somerepo.io/someimage"}, false},
		{"cluster create with unsupported type", []string{typeFlag, "unknown"}, true},
		{"cluster create with unknown flag", []string{"--unknown", "value"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(createSubCommandName, tt.args)
			if tt.expectErr {
				asserts.Error(t, err)
				return
			}
			asserts.NoError(t, err)
		})
	}
}

func TestClusterCreateHelp(t *testing.T) {
	output, _, err := runCommand(createSubCommandName, []string{"-h"})
	asserts.Nil(t, err)
	asserts.Contains(t, output, nameFlag)
	asserts.Contains(t, output, constants.ClusterNameFlagHelp)

	// The image and type flags should be hidden in help
	asserts.NotContains(t, output, constants.ClusterImageFlagName)
	asserts.NotContains(t, output, constants.ClusterImageFlagHelp)
	asserts.NotContains(t, output, typeFlag)
	asserts.NotContains(t, output, constants.ClusterTypeFlagHelp)

}

func TestClusterDeleteOptions(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{"cluster delete with default name", []string{typeFlag, capi.NoClusterType}, false},
		{"cluster delete with custom name", []string{nameFlag, "randomcluster", typeFlag, capi.NoClusterType}, false},
		{"cluster delete with unknown flag", []string{"--someflag", "randomcluster"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(deleteSubCommandName, tt.args)
			if tt.expectErr {
				asserts.Error(t, err)
				return
			}
			asserts.NoError(t, err)
		})
	}
}

func TestClusterDeleteHelp(t *testing.T) {
	output, _, err := runCommand(deleteSubCommandName, []string{"-h"})
	asserts.Nil(t, err)
	asserts.Contains(t, output, nameFlag)
	asserts.Contains(t, output, constants.ClusterNameFlagHelp)

	// The type flag should be hidden in delete help
	asserts.NotContains(t, output, typeFlag)
	asserts.NotContains(t, output, constants.ClusterTypeFlagHelp)
}

var testEmptyKubeconfigData = `
apiVersion: v1
kind: ""
clusters:
users:
contexts:
`

func TestClusterGetKubeconfigOptions(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		kubeconfigPath   string
		kubeconfigExists bool
		expectErr        bool
	}{
		{"cluster get-kubeconfig with default name", []string{typeFlag, capi.NoClusterType}, "", false, false},
		{"cluster get-kubeconfig with custom name", []string{nameFlag, "randomcluster", typeFlag, capi.NoClusterType}, "", false, false},
		{"cluster get-kubeconfig with custom kubeconfig path", []string{nameFlag, "randomcluster", typeFlag, capi.NoClusterType}, "somekubeconfig", false, false},
		{"cluster get-kubeconfig with existing kubeconfig", []string{nameFlag, "randomcluster", typeFlag, capi.NoClusterType}, "", true, false},
		{"cluster get-kubeconfig with unknown flag", []string{"--someflag", "randomcluster"}, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var kubeFile *os.File
			kubePath := ""
			var err error
			if tt.kubeconfigPath != "" {
				kubePath = tt.kubeconfigPath
			}
			if tt.kubeconfigExists {
				// create a tempfile for the kubeconfig so that it is an existing file
				kubeFile, err = os2.CreateTempFile(kubePath, []byte(testEmptyKubeconfigData))
				asserts.NoError(t, err)
				kubePath = kubeFile.Name()
			} else {
				// generate a random file name (which the test assumes does not exist)
				kubePath = path.Join(os.TempDir(), fmt.Sprintf("testgetkubeconfig_%d", time.Now().UnixNano()))
			}
			if kubePath != "" {
				tt.args = append(tt.args, kubePathFlag, kubePath)
				// ensure that if the file was created, we delete it
				defer func() { os.Remove(kubePath) }()
			}
			_, _, err = runCommand(getKubeconfigSubCommandName, tt.args)
			if tt.expectErr {
				asserts.Error(t, err)
				return
			}
			asserts.NoError(t, err)
		})
	}
}

func TestClusterGetKubeconfigHelp(t *testing.T) {
	output, _, err := runCommand(getKubeconfigSubCommandName, []string{"-h"})
	asserts.Nil(t, err)
	asserts.Contains(t, output, nameFlag)
	asserts.Contains(t, output, constants.ClusterNameFlagHelp)
	asserts.Contains(t, output, kubePathFlag)
	asserts.Contains(t, output, constants.KubeconfigPathFlagHelp)

	asserts.NotContains(t, output, typeFlag)
	asserts.NotContains(t, output, constants.ClusterTypeFlagHelp)
}

func runCommand(subCommand string, args []string) (string, string, error) {
	// Send stdout and stderr to byte buffers
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdCluster(rc)
	subCommandArr := []string{subCommand}
	cmd.SetArgs(append(subCommandArr, args...))
	err := cmd.Execute()
	return buf.String(), errBuf.String(), err
}
