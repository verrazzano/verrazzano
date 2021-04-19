// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"bytes"
	"errors"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	pkg2 "github.com/verrazzano/verrazzano/tools/cli/vz/pkg"
	"text/template"
)

var targetNamespace string
var listenPort int32
var image string

// this template is used to create the YAML we need to apply on the server
const tmpl = `
apiVersion: core.oam.dev/v1alpha2
kind: Component
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  workload:
    apiVersion: oam.verrazzano.io/v1alpha1
    kind: VerrazzanoHelidonWorkload
    metadata:
      name: {{.Name}}
      labels:
        app: {{.Name}}
    spec:
      deploymentTemplate:
        metadata:
          name: {{.Name}}-deployment
        podSpec:
          containers:
            - name: {{.Name}}-container
              image: "{{.Image}}"
              ports:
                - containerPort: {{.ListenPort}}
                  name: http
`

// this struct holds the data needed to populate the template
type templateData struct {
	Name string
	Namespace string
	Image string
	ListenPort int32
}

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

	// put data into struct
	data := templateData{
		Name: name,
		Namespace: targetNamespace,
		Image: image,
		ListenPort: listenPort,
	}

	// use template to get populate template with data
	var b bytes.Buffer
	t, err:= template.New("comp").Parse(tmpl)
	if err != nil {
		return err
	}

	err = t.Execute(&b, &data)
	if err != nil {
		return err
	}

	// apply the resulting data (yaml) on the server
	err = pkg2.ServerSideApply(pkg.GetKubeConfig(), b.String())
	return err
}
