// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"strings"
)

// PrintFlags contains all the flags used for output options
type PrintFlags struct {
	JSONYamlPrintFlags *genericclioptions.JSONYamlPrintFlags
	TemplateFlags      *genericclioptions.KubeTemplatePrintFlags

	OutputFormat *string
}

// NewGetPrintFlags returns a new PrintFlags struct for use
func NewGetPrintFlags() *PrintFlags {
	outputFormat := ""
	return &PrintFlags{
		OutputFormat:       &outputFormat,
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		TemplateFlags:      genericclioptions.NewKubeTemplatePrintFlags(),
	}
}

// AllowedFormats is a list of all the allowed formats for output options
func (f *PrintFlags) AllowedFormats() []string {
	formats := f.JSONYamlPrintFlags.AllowedFormats()
	formats = append(formats, f.TemplateFlags.AllowedFormats()...)
	return formats
}

// AddFlags binds all the required flags for output to the cmd
func (f *PrintFlags) AddFlags(cmd *cobra.Command) {
	f.JSONYamlPrintFlags.AddFlags(cmd)
	f.TemplateFlags.AddFlags(cmd)

	if f.OutputFormat != nil {
		cmd.Flags().StringVarP(f.OutputFormat, "output", "o", *f.OutputFormat, fmt.Sprintf("Output format. One of: %s", strings.Join(f.AllowedFormats(), "|")))
	}
}

// ToPrinter returns an appropriate printer based on the output format
// or returns an error if there is none
func (f *PrintFlags) ToPrinter() (printers.ResourcePrinter, error) {
	outputFormat := ""

	if f.OutputFormat != nil {
		outputFormat = *f.OutputFormat
	}

	if f.TemplateFlags.TemplateArgument != nil && len(*f.TemplateFlags.TemplateArgument) > 0 {
		outputFormat = "go-template"
	}

	if p, err := f.TemplateFlags.ToPrinter(outputFormat); !genericclioptions.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	if p, err := f.JSONYamlPrintFlags.ToPrinter(outputFormat); !genericclioptions.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	return nil, genericclioptions.NoCompatiblePrinterError{OutputFormat: &outputFormat, AllowedFormats: f.AllowedFormats()}
}
