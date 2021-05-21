// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"bytes"
	"errors"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"text/template"
)

var (
	targetNamespace string
	listenPort      int32
	image           string
	version         string
	description     string
	path            string
)

// this template is used to create the YAML we need to apply on the server
const compTmpl = `
apiVersion: core.oam.dev/v1alpha2
kind: Component
metadata:
  name: {{.Name}}-component
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
const appTmpl = `
apiVersion: core.oam.dev/v1alpha2
kind: ApplicationConfiguration
metadata:
  name: {{.Name}}-appconf
  namespace: {{.Namespace}}
  annotations:
    version: {{.Version}}
    description: "{{.Description}}"
spec:
  components:
    - componentName: {{.Name}}-component
      traits:
        - trait:
            apiVersion: oam.verrazzano.io/v1alpha1
            kind: MetricsTrait
            spec:
                scraper: verrazzano-system/vmi-system-prometheus-0
        - trait:
            apiVersion: oam.verrazzano.io/v1alpha1
            kind: IngressTrait
            metadata:
              name: {{.Name}}-ingress
            spec:
              rules:
                - paths:
                    - path: "{{.Path}}"
                      pathType: Prefix
`

// this struct holds the data needed to populate the template
type templateData struct {
	Name        string
	Namespace   string
	Image       string
	ListenPort  int32
	Version     string
	Description string
	Path        string
}

type HelidonCreateOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args []string
	genericclioptions.IOStreams
}

func NewHelidonCreateOptions(streams genericclioptions.IOStreams) *HelidonCreateOptions {
	return &HelidonCreateOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams: streams,
	}
}

func NewCmdAppHelidonCreate(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewHelidonCreateOptions(streams)
	cmd := &cobra.Command{
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
	o.configFlags.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&targetNamespace, "targetnamespace", "m", "default", "Namespace to create Helidon application in")
	cmd.Flags().Int32VarP(&listenPort, "listenport", "l", 8080, "Helidon application's listen port")
	cmd.Flags().StringVarP(&image, "image", "i", "", "Docker image for the application")
	cmd.Flags().StringVarP(&version, "version", "v", "v1.0.0", "Version of the application")
	cmd.Flags().StringVarP(&description, "description", "d", "An Helidon application", "Description of the application")
	cmd.Flags().StringVarP(&path, "path", "p", "/", "Path to the application endpoint")
	return cmd
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
		Name:        name,
		Namespace:   targetNamespace,
		Image:       image,
		ListenPort:  listenPort,
		Version:     version,
		Path:        path,
		Description: description,
	}

	// --- Create the OAM Component ---

	// use template to get populate template with data
	var b bytes.Buffer
	t, err := template.New("comp").Parse(compTmpl)
	if err != nil {
		return err
	}

	err = t.Execute(&b, &data)
	if err != nil {
		return err
	}

	// apply the resulting data (yaml) on the server
	err = helpers.ServerSideApply(pkg.GetKubeConfig(), b.String())
	if err != nil {
		return err
	}

	// --- create the OAM ApplicationConfiguration ---
	var b2 bytes.Buffer
	t2, err := template.New("appconfig").Parse(appTmpl)
	if err != nil {
		return err
	}

	err = t2.Execute(&b2, &data)
	if err != nil {
		return err
	}

	err = helpers.ServerSideApply(pkg.GetKubeConfig(), b2.String())
	return err
}
