// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package template

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
)

// Processor contains the references required to populate a golang template
type Processor struct {
	template string
	client   client.Client
}

// NewProcessor creates a new template processor that can read values from kubernetes resources and provided structs
func NewProcessor(client client.Client, template string) *Processor {
	return &Processor{
		client:   client,
		template: template,
	}
}

// get fetches a resources from the processor's client and returns the resulting map
func (p *Processor) get(apiversion string, kind string, namespace string, name string) (map[string]interface{}, error) {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiversion)
	u.SetKind(kind)
	err := p.client.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, u)
	if err != nil {
		return nil, err
	}
	return u.Object, nil
}

// configmap populates template values from a kubernetes configmap
func (p *Processor) configmap(namespace string, name string, key string) (string, error) {
	cm := &k8score.ConfigMap{}
	err := p.client.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, cm)
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
func (p *Processor) secret(namespace string, name string, key string) (string, error) {
	secret := &k8score.Secret{}
	err := p.client.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, secret)
	if err != nil {
		return "", err
	}
	value, ok := secret.Data[key]
	if !ok {
		return "", errors.New("missing value for key " + key)
	}

	return string(value), nil
}

// Process leverages kubernetes and the provided inputs to populate a template
func (p *Processor) Process(inputs map[string]interface{}) (string, error) {
	h := sha256.New()
	h.Write([]byte(p.template))
	n := base64.URLEncoding.EncodeToString(h.Sum(nil))
	t, err := template.New(n).
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"get":       p.get,
			"configmap": p.configmap,
			"secret":    p.secret}).
		Parse(p.template)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	var v bytes.Buffer
	err = t.Execute(&v, inputs)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return v.String(), nil
}
