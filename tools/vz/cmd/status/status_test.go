// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStatusCmd(t *testing.T) {
	name := "verrazzano"
	namespace := "test"

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Status: vzapi.VerrazzanoStatus{
				Version:            "1.2.3",
				VerrazzanoInstance: nil,
				Conditions:         nil,
				State:              "",
				Components:         nil,
			},
		}).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: os.Stderr})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, expect the Verrazzano resource to be found
	statusCmd.SetArgs([]string{fmt.Sprintf("--%s", nameFlag), name, fmt.Sprintf("--%s", namespaceFlag), namespace})
	err := statusCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	assert.True(t, strings.Contains(result, "Version 1.2.3"))
}
