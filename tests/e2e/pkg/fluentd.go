// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const verrazzanoNamespace string = "verrazzano-system"
const fluentdDaemonsetName string = "fluentd"

func GetFluentdDaemonset() (*appv1.DaemonSet, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ds, err := clientset.AppsV1().DaemonSets(verrazzanoNamespace).Get(context.TODO(), fluentdDaemonsetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func AssertFluentdURLAndSecret(fluentdDaemonset *appv1.DaemonSet, expectedURL, expectSecret string) {
	urlFound := ""
	usernameSecretFound := ""
	passwordSecretFound := ""
	secretVolumeFound := ""
	containers := fluentdDaemonset.Spec.Template.Spec.Containers
	gomega.Expect(len(containers)).To(gomega.Equal(1))
	for _, env := range containers[0].Env {
		if env.Name == "ELASTICSEARCH_URL" {
			urlFound = env.Value
		}
		if env.Name == "ELASTICSEARCH_USER" {
			usernameSecretFound = env.ValueFrom.SecretKeyRef.Name
		}
		if env.Name == "ELASTICSEARCH_PASSWORD" {
			passwordSecretFound = env.ValueFrom.SecretKeyRef.Name
		}
	}
	for _, vol := range fluentdDaemonset.Spec.Template.Spec.Volumes {
		if vol.Name == "secret-volume" {
			secretVolumeFound = vol.Secret.SecretName
		}
	}
	gomega.Expect(urlFound).To(gomega.Equal(expectedURL))
	gomega.Expect(usernameSecretFound).To(gomega.Equal(expectSecret))
	gomega.Expect(passwordSecretFound).To(gomega.Equal(expectSecret))
	gomega.Expect(secretVolumeFound).To(gomega.Equal(expectSecret))
}
