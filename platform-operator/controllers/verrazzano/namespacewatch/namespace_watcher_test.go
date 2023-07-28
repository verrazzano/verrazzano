// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespacewatch

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

var period = time.Duration(10) * time.Second
var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
}

func TestStart(t *testing.T) {

}

func TestMoveSystemNamespacesToRancherSystemProject(t *testing.T) {
	namespace1 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano-system",
			Labels: map[string]string{
				constants.VerrazzanoManagedKey: "verrazzano-system",
			},
		},
	}
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(namespace1).Build()
	//fakeCtx := spi.NewFakeContext(client, nil, nil, false)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(testScheme)
	namespaceWatcher = NewNamespaceWatcher(client, period)
	namespaceWatcher.MoveSystemNamespacesToRancherSystemProject(dynamicClient)

}
