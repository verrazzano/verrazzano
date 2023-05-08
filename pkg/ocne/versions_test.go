// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocne

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

const testFileName = "testdata/kubernetes_versions.yaml"

func TestLoadMetadata(t *testing.T) {
	k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset().CoreV1(), nil
	}
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	err := CreateOCNEMetadataConfigMap(context.TODO(), testFileName)
	assert.NoError(t, err)
}
