// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const verrazzanoNamespace string = "verrazzano-system"
const VmiESURL = "http://verrazzano-authproxy-elasticsearch:8775"
const VmiESLegacySecret = "verrazzano"               //nolint:gosec //#gosec G101
const VmiESInternalSecret = "verrazzano-es-internal" //nolint:gosec //#gosec G101

func getFluentdDaemonset() (*appv1.DaemonSet, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ds, err := clientset.AppsV1().DaemonSets(verrazzanoNamespace).Get(context.TODO(), globalconst.FluentdDaemonSetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return ds, nil
}

// AssertFluentdURLAndSecret makes sure that expectedURL and expectedSecret are the same as
// what is configured in the fluentd daemonset.
// empty string expectedURL means skip comparing URL. Verrazzano prior to 1.1 could configure URL differently.
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
			if env.Name == "OPENSEARCH_URL" {
				urlFound = env.Value
			}
			if env.Name == "OPENSEARCH_USER" {
				usernameSecretFound = env.ValueFrom.SecretKeyRef.Name
			}
			if env.Name == "OPENSEARCH_PASSWORD" {
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
	return (expectedURL == "" || urlFound == expectedURL) && usernameSecretFound == expectedSecret && passwordSecretFound == expectedSecret && volumeSecretFound == expectedSecret
}
