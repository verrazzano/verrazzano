// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	v1alpha12 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var targetNamespace string
var listenPort int32
var image string

func init() {
	helidonCreateCmd.Flags().StringVarP(&targetNamespace, "namespace", "n", "default", "Namespace to create Helidon application in")
	helidonCreateCmd.Flags().Int32VarP(&listenPort, "listenport", "l", 8080, "Helidon application's listen port")
	helidonCreateCmd.Flags().StringVarP(&image, "image", "i", "", "Docker image for the application")
	helidonCmd.AddCommand(helidonCreateCmd)
}

var helidonCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an Helidon application",
	Long:  "Create an Helidon application",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := createHelidonApplication(args); err != nil {
			return err
		}
		return nil
	},
}

func createHelidonApplication(args []string) error {
	name := args[0]

	// validate data
	if len(image) == 0 {
		return errors.New("you must specify the Docker image name")
	}

	// (the rest are validated or defaulted by cobra)

	//// connect to the server
	//config := pkg.GetKubeConfig()
	//oamclientset, err := v1alpha1.NewForConfig(config)
	//if err != nil {
	//	fmt.Print("could not get the OAM/Helidon clientset")
	//}

	// create the Helidon workload resource
	app := v1alpha12.VerrazzanoHelidonWorkload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: targetNamespace,
		},
		Spec: v1alpha12.VerrazzanoHelidonWorkloadSpec{
			DeploymentTemplate: v1alpha12.DeploymentTemplate{
				Metadata: metav1.ObjectMeta{
					Name:        name,
					Labels:      nil,
					Annotations: nil,
				},

				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: listenPort,
								},
							},
						},
					},
				},
			},
		},
	}

	fmt.Printf("app: %+v\n", app)

	// create the OAM component file

	return nil
}
