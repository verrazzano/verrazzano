// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const verrazzanoNamespace string = "verrazzano-system"
const fluentdDaemonsetName string = "fluentd"
const VmiESURL = "http://vmi-system-es-ingest-oidc:8775"
const VmiESSecret = "verrazzano"

func getFluentdDaemonset() (*appv1.DaemonSet, error) {
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

func AssertFluentdURLAndSecret(expectedURL, expectedSecret string) bool {
	urlFound := ""
	usernameSecretFound := ""
	passwordSecretFound := ""
	volumeSecretFound := ""
	fluentdDaemonset, err := getFluentdDaemonset()
	if err != nil {
		Log(Info, "Can not find fluentdDaemonset")
		return false
	}
	if fluentdDaemonset == nil {
		Log(Info, "Can not find fluentdDaemonset")
		return false
	}
	containers := fluentdDaemonset.Spec.Template.Spec.Containers
	if len(containers) > 0 {
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
	} else {
		Log(Info, "Can not find containers in fluentdDaemonset")
		return false
	}
	if len(fluentdDaemonset.Spec.Template.Spec.Volumes) > 0 {
		for _, vol := range fluentdDaemonset.Spec.Template.Spec.Volumes {
			if vol.Name == "secret-volume" {
				volumeSecretFound = vol.Secret.SecretName
			}
		}
	} else {
		Log(Info, "Can not find volumes in fluentdDaemonset")
		return false
	}
	if urlFound != expectedURL {
		Log(Info, fmt.Sprintf("ES URL in fluentdDaemonset %s doesn't match expected %s", urlFound, expectedURL))
	}
	if usernameSecretFound != expectedSecret {
		Log(Info, fmt.Sprintf("ES user secret in fluentdDaemonset %s doesn't match expected %s", usernameSecretFound, expectedSecret))
	}
	if passwordSecretFound != expectedSecret {
		Log(Info, fmt.Sprintf("ES password secret in fluentdDaemonset %s doesn't match expected %s", passwordSecretFound, expectedSecret))
	}
	if volumeSecretFound != expectedSecret {
		Log(Info, fmt.Sprintf("ES volume secret in fluentdDaemonset %s doesn't match expected %s", volumeSecretFound, expectedSecret))
	}
	return urlFound == expectedURL && usernameSecretFound == expectedSecret && passwordSecretFound == expectedSecret && volumeSecretFound == expectedSecret
}
