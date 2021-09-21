// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/util/openapi"
)

func Test_struct2Unmarshal(t *testing.T) {
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    unstructured.Unstructured
		wantErr bool
	}{
		{
			name: "volumeMountJSON",
			args: args{
				obj: &corev1.VolumeMount{
					MountPath: loggingMountPath,
					Name:      loggingVolume,
					SubPath:   loggingKey,
					ReadOnly:  true,
				},
			},
			want: unstructured.Unstructured{
				Object: map[string]interface{}{
					"mountPath": loggingMountPath,
					"name":      loggingVolume,
					"subPath":   loggingKey,
					"readOnly":  true,
				},
			},
			wantErr: false,
		},
		{
			name: "nilJSON",
			args: args{
				obj: nil,
			},
			want: unstructured.Unstructured{
				Object: nil,
			},
			wantErr: false,
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := struct2Unmarshal(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("struct2Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("struct2Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_locateField(t *testing.T) {
	type args struct {
		document   openapi.Resources
		res        *unstructured.Unstructured
		fieldPaths [][]string
	}

	server, err := openapiSchemaFakeServer(t)
	if err != nil {
		t.Fatalf("Could not create fake server from openapi, %v", err)
	}

	client := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})
	schema, err := client.OpenAPISchema()
	if err != nil {
		t.Fatalf("Could not create the schema for the discoveryClient, %v", err)
	}
	document, err := openapi.NewOpenAPIData(schema)
	if err != nil {
		t.Fatalf("Could not get document from given schema: %v", err)
	}

	resource1 := unstructured.Unstructured{}
	resource1.SetAPIVersion("core/v1")
	resource1.SetKind("namespace")

	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		{
			name: "test1",
			args: args{
				document:   document,
				res:        &resource1,
				fieldPaths: [][]string{{"fake.type.1"}},
			},
			want: true,
			want1: []string{"test"},
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateField(tt.args.document, tt.args.res, tt.args.fieldPaths)
			if got != tt.want {
				t.Errorf("locateField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_locateContainersField(t *testing.T) {
	type args struct {
		document openapi.Resources
		res      *unstructured.Unstructured
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateContainersField(tt.args.document, tt.args.res)
			if got != tt.want {
				t.Errorf("locateContainersField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateContainersField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_locateVolumesField(t *testing.T) {
	type args struct {
		document openapi.Resources
		res      *unstructured.Unstructured
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateVolumesField(tt.args.document, tt.args.res)
			if got != tt.want {
				t.Errorf("locateVolumesField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateVolumesField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_locateVolumeMountsField(t *testing.T) {
	type args struct {
		document openapi.Resources
		res      *unstructured.Unstructured
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateVolumeMountsField(tt.args.document, tt.args.res)
			if got != tt.want {
				t.Errorf("locateVolumeMountsField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateVolumeMountsField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}


func openapiSchemaFakeServer(t *testing.T) (*httptest.Server, error) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/openapi/v2" {
			errMsg := fmt.Sprintf("Unexpected url %v", req.URL)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(errMsg))
			t.Errorf("testing should fail as %s", errMsg)
			return
		}
		if req.Method != "GET" {
			errMsg := fmt.Sprintf("Unexpected method %v", req.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(errMsg))
			t.Errorf("testing should fail as %s", errMsg)
			return
		}
		decipherableFormat := req.Header.Get("Accept")
		if decipherableFormat != "application/com.github.proto-openapi.spec.v2@v1.0+protobuf" {
			errMsg := fmt.Sprintf("Unexpected accept mime type %v", decipherableFormat)
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte(errMsg))
			t.Errorf("testing should fail as %s", errMsg)
			return
		}

		mime.AddExtensionType(".pb-v1", "application/com.github.googleapis.gnostic.OpenAPIv2@68f4ded+protobuf")

		output, err := proto.Marshal(returnedOpenAPI())
		if err != nil {
			errMsg := fmt.Sprintf("Unexpected marshal error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			t.Errorf("testing should fail as %s", errMsg)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))

	return server, nil
}

func returnGVKYaml() string {
	return `
- group: core
  version: v1
  kind: namespace`
}

func returnedOpenAPI() *openapi_v2.Document {
	return &openapi_v2.Document{
		Definitions: &openapi_v2.Definitions{
			AdditionalProperties: []*openapi_v2.NamedSchema{
				{
					Name: "namespace",
					Value: &openapi_v2.Schema{
						Properties: &openapi_v2.Properties{
							AdditionalProperties: []*openapi_v2.NamedSchema{
								{
									Name: "metadata",
									Value: &openapi_v2.Schema{
										Properties: &openapi_v2.Properties{
											AdditionalProperties: []*openapi_v2.NamedSchema{
												{
													Name: "name",
													Value: &openapi_v2.Schema{
														Type: &openapi_v2.TypeItem{
															Value: []string{"string"},
														},
													},
												},
												{
													Name: "labels",
													Value: &openapi_v2.Schema{
														Properties: &openapi_v2.Properties{
															AdditionalProperties: []*openapi_v2.NamedSchema{
																{
																	Name: "name",
																	Value: &openapi_v2.Schema{
																		Type: &openapi_v2.TypeItem{
																			Value: []string{"string"},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						VendorExtension: []*openapi_v2.NamedAny{
							{
								Name: "x-kubernetes-group-version-kind",
								Value: &openapi_v2.Any{
									Yaml: returnGVKYaml(),
								},
							},
						},
					},
				},
			},
		},
	}
}