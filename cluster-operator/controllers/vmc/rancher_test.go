// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TestCreateOrUpdateSecretRancherProxy tests the create or update simulation through the Rancher proxy
// GIVEN a new secret for the managed cluster
//
//	WHEN the createOrUpdateSecretRancherProxy is called
//	THEN the managed cluster has a version of the new secret spec that is generated
func TestCreateOrUpdateSecretRancherProxy(t *testing.T) {
	a := asserts.New(t)

	savedRetry := defaultRetry
	defer func() {
		defaultRetry = savedRetry
	}()
	defaultRetry = wait.Backoff{
		Steps:    1,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()

	secret := corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-name",
		},
	}
	clusterID := "cluster-id"

	mocker := gomock.NewController(t)

	tests := []struct {
		name   string
		f      controllerutil.MutateFn
		mock   *mocks.MockRequestSender
		result controllerutil.OperationResult
	}{
		{
			name:   "test secret not found",
			f:      func() error { return nil },
			mock:   addNotFoundMock(mocks.NewMockRequestSender(mocker), &secret, clusterID),
			result: controllerutil.OperationResultCreated,
		},
		{
			name:   "test secret found",
			f:      func() error { return nil },
			mock:   addFoundMock(mocks.NewMockRequestSender(mocker), a, &secret, clusterID),
			result: controllerutil.OperationResultNone,
		},
		{
			name: "test secret mutated",
			f: func() error {
				secret.Data = make(map[string][]byte)
				secret.Data["test"] = []byte("newVal")
				return nil
			},
			mock:   addMutatedMock(mocks.NewMockRequestSender(mocker), a, &secret, clusterID),
			result: controllerutil.OperationResultUpdated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rancherHTTPClient = tt.mock
			result, err := createOrUpdateSecretRancherProxy(&secret, &RancherConfig{}, clusterID, tt.f, vzlog.DefaultLogger())
			a.Nil(err)
			a.Equal(tt.result, result)
		})
	}
}

func Test_makeClusterPayload(t *testing.T) {
	// a := asserts.New(t)

	tests := []struct {
		name        string
		clusterName string
		labels      map[string]string
	}{
		{
			name:        "test nil labels",
			clusterName: "nilLabelsCluster",
			labels:      nil,
		},
		{
			name:        "test empty labels map",
			clusterName: "nilLabelsCluster",
			labels:      map[string]string{},
		},
		{
			name:        "test one label",
			clusterName: "cluster1",
			labels: map[string]string{
				"label1": "val1",
			},
		},
		{
			name:        "test two labels",
			clusterName: "cluster2",
			labels: map[string]string{
				"label1":      "val1",
				"rancherLbl2": "val2",
			},
		},
		{
			name:        "test 3 labels",
			clusterName: "cluster3",
			labels: map[string]string{
				"label1":      "val1",
				"rancherLbl2": "val2",
				"rancherLbl3": "val3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := makeClusterPayload(tt.clusterName, tt.labels)
			fmt.Printf("%v\n", payload)
			payloadParsed, err := gabs.ParseJSON([]byte(payload))
			asserts.NoError(t, err)
			asserts.Equal(t, tt.clusterName, payloadParsed.Path("name").Data())
			asserts.Equal(t, "cluster", payloadParsed.Path("type").Data())
			labels := payloadParsed.Path("labels")
			if tt.labels == nil || len(tt.labels) == 0 {
				asserts.Nil(t, labels)
			}
			for key, val := range tt.labels {
				asserts.Equal(t, val, labels.Path(key).Data())
			}
		})
	}
}

func addNotFoundMock(httpMock *mocks.MockRequestSender, secret *corev1.Secret, clusterID string) *mocks.MockRequestSender {
	httpMock.EXPECT().Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURIMethod(http.MethodGet, getTestPath(secret, clusterID, false))).
		Return(&http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader([]byte("")))}, fmt.Errorf("not found")).AnyTimes()
	httpMock.EXPECT().Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURIMethod(http.MethodPost, getTestPath(secret, clusterID, true))).
		Return(&http.Response{StatusCode: 201, Body: io.NopCloser(bytes.NewReader([]byte("")))}, nil)
	return httpMock
}

func addFoundMock(httpMock *mocks.MockRequestSender, a *asserts.Assertions, secret *corev1.Secret, clusterID string) *mocks.MockRequestSender {
	secretData, err := json.Marshal(secret)
	a.NoError(err)
	httpMock.EXPECT().Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURIMethod(http.MethodGet, getTestPath(secret, clusterID, false))).
		Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(secretData))}, nil)
	return httpMock
}

func addMutatedMock(httpMock *mocks.MockRequestSender, a *asserts.Assertions, secret *corev1.Secret, clusterID string) *mocks.MockRequestSender {
	secretData, err := json.Marshal(secret)
	a.NoError(err)
	httpMock.EXPECT().Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURIMethod(http.MethodGet, getTestPath(secret, clusterID, false))).
		Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(secretData))}, nil)
	httpMock.EXPECT().Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURIMethod(http.MethodPut, getTestPath(secret, clusterID, false))).
		Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte("")))}, nil)
	return httpMock
}

func getTestPath(secret *corev1.Secret, clusterID string, create bool) string {
	if create {
		return k8sClustersPath + clusterID + fmt.Sprintf(secretCreateTemplate, secret.GetNamespace())
	}
	return k8sClustersPath + clusterID + fmt.Sprintf(secretPathTemplate, secret.GetNamespace(), secret.GetName())
}
