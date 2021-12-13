package template

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getSecret returns a secret using the client provided in the template processor
func getSecret(name string, tp *Processor) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := tp.client.Get(context.TODO(), client.ObjectKey{Namespace: tp.namespace, Name: name}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// getConfigMap returns a configmap using the client provided in the template processor
func getConfigMap(name string, tp *Processor) (*v1.ConfigMap, error) {
	cm := &v1.ConfigMap{}
	err := tp.client.Get(context.TODO(), client.ObjectKey{Namespace: tp.namespace, Name: name}, cm)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
