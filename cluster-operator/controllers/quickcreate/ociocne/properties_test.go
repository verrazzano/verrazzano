// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"text/template"
)

func TestT(t *testing.T) {
	p := Properties{
		OCNEOCIQuickCreate: &vmcv1alpha1.OCNEOCIQuickCreate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "frobber",
				Namespace: "Foo!",
			},
			Spec: vmcv1alpha1.OCIOCNEClusterSpec{
				CommonClusterSpec: vmcv1alpha1.CommonClusterSpec{
					Kubernetes: vmcv1alpha1.Kubernetes{
						Version: "123",
					},
				},
			},
		},
	}
	text := "Name: {{.Spec.Kubernetes.Version}}"
	tmpl, err := template.New("foo").
		Option("missingkey=error"). // Treat any missing keys as errors
		Parse(text)
	assert.NoError(t, err)
	buffer := &bytes.Buffer{}
	err = tmpl.Execute(buffer, p)
	assert.NoError(t, err)
	b := buffer.Bytes()
	fmt.Println(b)
}
