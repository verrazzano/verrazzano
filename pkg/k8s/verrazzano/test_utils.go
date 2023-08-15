// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// getTestingContextAndClient returns the Context and the Client used for these unit tests.
// v1beta1 is loaded into the client's scheme.
func getTestingContextAndClient() (context.Context, client.Client, error) {
	ctx := context.TODO()

	scheme := runtime.NewScheme()
	if err := v1beta1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	return ctx, client, nil
}

// createTestVZ creates a v1beta1 VZ resource through the fake client.
// The expected v1alpha1 version of that VZ resource is returned.
func createTestVZ(ctx context.Context, client client.Client) (*v1alpha1.Verrazzano, error) {
	// create a v1beta1 Verrazzano through the K8s client
	var vzStoredV1Beta1 *v1beta1.Verrazzano
	vzStoredV1Beta1 = loadV1Beta1()
	if err := client.Create(ctx, vzStoredV1Beta1); err != nil {
		return nil, err
	}

	// the expected VZ resource returned should be v1alpha1
	var vzExpected *v1alpha1.Verrazzano
	vzExpected = loadV1Alpha1()
	return vzExpected, nil
}

// loadV1Alpha1 returns a pointer to a v1alpha1 Verrazzano struct.
// The returned Verrazzano is equivalent to the one returned by loadV1Beta1
// except for the API version.
func loadV1Alpha1() (*v1alpha1.Verrazzano) {
	return &v1alpha1.Verrazzano{
		TypeMeta: metav1.TypeMeta{
			Kind: "verrazzano",
			APIVersion: "install.verrazzano.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano",
		},
		Spec: v1alpha1.VerrazzanoSpec{
			Profile: "prod",
		},
	}
}

// loadV1Beta1 returns a pointer to a v1alpha1 Verrazzano struct.
// The returned Verrazzano is equivalent to the one returned by loadV1Beta1
// except for the API version.
func loadV1Beta1() (*v1beta1.Verrazzano) {
	return &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{
			Kind: "verrazzano",
			APIVersion: "install.verrazzano.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano",
		},
		Spec: v1beta1.VerrazzanoSpec{
			Profile: "prod",
		},
	}
}


func loadTestCase(version string) ([]byte, error) {
	path := path.Join("./testdata", fmt.Sprintf("%s.yaml", version))
	fmt.Printf("path = %s\n", path)
	return os.ReadFile(path)
}
