// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path"
	controllerruntime "sigs.k8s.io/controller-runtime"
	crtpkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

const (
	sep       = "---"
	specField = "spec"
)

type (
	YAMLApplier struct {
		client  crtpkg.Client
		objects []unstructured.Unstructured
	}

	action func(obj *unstructured.Unstructured) error
)

func NewYAMLApplier(client crtpkg.Client) *YAMLApplier {
	return &YAMLApplier{
		client:  client,
		objects: []unstructured.Unstructured{},
	}
}

//Objects is the list of objects created using the ApplyX methods
func (y *YAMLApplier) Objects() []unstructured.Unstructured {
	return y.objects
}

//ApplyD applies all YAML files in a directory to Kubernetes
func (y *YAMLApplier) ApplyD(directory string) error {
	files, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	filteredFiles := filterYamlExt(files)
	if len(filteredFiles) < 1 {
		return fmt.Errorf("no files passed to apply: %s", directory)
	}
	for _, file := range filteredFiles {
		filePath := path.Join(directory, file.Name())
		if err = y.ApplyF(filePath); err != nil {
			return err
		}
	}

	return nil
}

//ApplyF applies a file spec to Kubernetes
func (y *YAMLApplier) ApplyF(filePath string) error {
	return y.doFileAction(filePath, y.applyAction)
}

//ApplyFT applies a file template spec (go text.template) to Kubernetes
func (y *YAMLApplier) ApplyFT(filePath string, args map[string]interface{}) error {
	return y.doTemplatedFileAction(filePath, y.applyAction, args)
}

// ApplyFTDefaultConfig calls ApplyFT with rest client from the default config
func (y *YAMLApplier) ApplyFTDefaultConfig(filePath string, args map[string]interface{}) error {
	config, err := GetKubeConfig()
	if err != nil {
		return err
	}
	client, err := crtpkg.New(config, crtpkg.Options{})
	if err != nil {
		return err
	}
	y.client = client
	return y.ApplyFT(filePath, args)
}

//DeleteF deletes a file spec from Kubernetes
func (y *YAMLApplier) DeleteF(filePath string) error {
	return y.doFileAction(filePath, y.deleteAction)
}

//DeleteFT deletes a file template spec (go text.template) to Kubernetes
func (y *YAMLApplier) DeleteFT(filePath string, args map[string]interface{}) error {
	return y.doTemplatedFileAction(filePath, y.deleteAction, args)
}

// DeleteFTDefaultConfig calls deleteFT with rest client from the default config
func (y *YAMLApplier) DeleteFTDefaultConfig(filePath string, args map[string]interface{}) error {
	config, err := GetKubeConfig()
	if err != nil {
		return err
	}
	client, err := crtpkg.New(config, crtpkg.Options{})
	if err != nil {
		return err
	}
	y.client = client
	return y.DeleteFT(filePath, args)
}

//applyAction creates a merge patch of the object with the server object
func (y *YAMLApplier) applyAction(obj *unstructured.Unstructured) error {
	clientSpec, _, err := unstructured.NestedFieldCopy(obj.Object, specField)
	if err != nil {
		return err
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), y.client, obj, func() error {
		serverSpec, _, err := unstructured.NestedFieldCopy(obj.Object, specField)
		if err != nil {
			return err
		}
		merge(serverSpec.(map[string]interface{}), clientSpec.(map[string]interface{}))
		return unstructured.SetNestedField(obj.Object, serverSpec, specField)
	}); err != nil {
		return err
	}
	y.objects = append(y.objects, *obj)
	return nil
}

//deleteAction deletes the object from the server
func (y *YAMLApplier) deleteAction(obj *unstructured.Unstructured) error {
	if err := y.client.Delete(context.TODO(), obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

//doFileAction runs the action against a file
func (y *YAMLApplier) doFileAction(filePath string, f action) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return y.doAction(bufio.NewReader(file), f)

}

//doTemplatedFileAction runs the action against a template file
func (y *YAMLApplier) doTemplatedFileAction(filePath string, f action, args map[string]interface{}) error {
	templateName := path.Base(filePath)
	tmpl, err := template.New(templateName).
		Option("missingkey=error"). // Treat any missing keys as errors
		ParseFiles(filePath)
	if err != nil {
		return err
	}
	buffer := &bytes.Buffer{}
	if err = tmpl.Execute(buffer, args); err != nil {
		return err
	}
	return y.doAction(bufio.NewReader(buffer), f)
}

//doAction executes the action on a YAML reader
func (y *YAMLApplier) doAction(reader *bufio.Reader, f action) error {
	objs, err := y.unmarshall(reader)
	if err != nil {
		return err
	}

	for i := range objs {
		if err := f(&objs[i]); err != nil {
			return err
		}
	}
	return nil
}

//unmarshall a reader containing YAML to a list of unstructured objects
func (y *YAMLApplier) unmarshall(reader *bufio.Reader) ([]unstructured.Unstructured, error) {
	buffer := bytes.Buffer{}
	objs := []unstructured.Unstructured{}

	flushBuffer := func() error {
		if buffer.Len() < 1 {
			return nil
		}
		obj := unstructured.Unstructured{Object: map[string]interface{}{}}
		yamlBytes := buffer.Bytes()
		if err := yaml.Unmarshal(yamlBytes, &obj); err != nil {
			return err
		}
		if len(obj.Object) > 0 {
			objs = append(objs, obj)
		}
		buffer.Reset()
		return nil
	}

	for {
		// Read the file line by line
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// if EOF, flush the buffer and return the objs
			if err == io.EOF {
				flushErr := flushBuffer()
				return objs, flushErr
			}
			return objs, err
		}
		lineStr := string(line)
		// Flush buffer at document break
		if strings.TrimSpace(lineStr) == sep {
			if err = flushBuffer(); err != nil {
				return objs, err
			}
		} else {
			// Save line to buffer
			if _, err := buffer.Write(line); err != nil {
				return objs, err
			}
		}
	}
}

//merge keys from m2 into m1, overwriting existing keys of m1.
func merge(m1, m2 map[string]interface{}) {
	for k, v := range m2 {
		m1[k] = v
	}
}

//DeleteAll deletes all objects created by the applier
//If you are using a YAMLApplier in a temporary context, please use defer y.DeleteAll()
//to clean up resources when you are done.
func (y *YAMLApplier) DeleteAll() error {
	for i := range y.objects {
		if err := y.client.Delete(context.TODO(), &y.objects[i]); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	y.objects = []unstructured.Unstructured{}
	return nil
}

//isYamlExt checks if a file has a YAML extension.
func isYamlExt(fileName string) bool {
	ext := path.Ext(fileName)
	return ext == ".yml" || ext == ".yaml"
}

func filterYamlExt(files []os.DirEntry) []os.DirEntry {
	res := []os.DirEntry{}
	for _, file := range files {
		if !file.IsDir() && isYamlExt(file.Name()) {
			res = append(res, file)
		}
	}

	return res
}
