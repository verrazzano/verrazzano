// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package templates

import (
	"bytes"
	"text/template"
)

// ApplyTemplate - apply the replacement parameters to the specified template content
func ApplyTemplate(templateContent string, params interface{}) (string, error) {

	// Parse the template file
	testTemplate, err := template.New("cli").Parse(templateContent)
	if err != nil {
		return "", err
	}

	// Apply the replacement parameters to the template
	var buf bytes.Buffer
	err = testTemplate.Execute(&buf, &params)
	if err != nil {
		return "", err
	}

	// Return the result
	return buf.String(), nil
}
