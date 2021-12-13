package template

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
)

// Processor contains the references required to populate a golang template
type Processor struct {
	templateFile string
	namespace    string
	client       client.Client
}

// NewProcessor creates a new template processor that can read values from kubernetes resources and provided structs
func NewProcessor(namespace string, client client.Client, templateFile string) *Processor {
	return &Processor{
		namespace:    namespace,
		client:       client,
		templateFile: templateFile,
	}
}

// configmap populates template values from a kubernetes configmap
func (p *Processor) configmap(name string, key string) (string, error) {
	cm, err := getConfigMap(name, p)
	if err != nil {
		return "", err
	}
	value, ok := cm.Data[key]
	if !ok {
		return "", errors.New("missing value for key " + key)
	}

	return value, nil
}

// secret populates template values from a kubernetes secret
func (p *Processor) secret(name string, key string) (string, error) {
	secret, err := getSecret(name, p)
	if err != nil {
		return "", err
	}
	value, ok := secret.Data[key]
	if !ok {
		return "", errors.New("missing value for key " + key)
	}

	return string(value), nil
}

// processTemplate leverages kubernetes and the provided inputs to populate a template
func (p *Processor) processTemplate(inputs map[string]interface{}) (string, error) {
	t := template.New(path.Base(p.templateFile))
	t.Funcs(template.FuncMap{
		"configmap": p.configmap,
		"secret":    p.secret,
	})
	t, err := t.ParseFiles(p.templateFile)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	var value bytes.Buffer
	err = t.Execute(&value, inputs)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return value.String(), nil
}
