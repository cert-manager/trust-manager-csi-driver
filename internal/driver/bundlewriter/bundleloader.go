/*
Copyright 2024 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bundlewriter

import (
	"context"
	"fmt"

	trustv1alpha1 "github.com/cert-manager/trust-manager/pkg/apis/trust/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BundleLoader is used to load the CA bundle
type BundleLoader interface {
	// Load will load a CA bundle given a trust-manager bundle name
	Load(ctx context.Context, namespace, name string) ([]byte, error)
}

// NewBundleLoader creates a new BundleLoader with the given Kubernetes client
func NewBundleLoader(client client.Client) BundleLoader {
	return bundleLoader{client: client}
}

type bundleLoader struct {
	client client.Client
}

func (l bundleLoader) Load(ctx context.Context, namespace, name string) ([]byte, error) {
	// Load the bundle object
	var bundle trustv1alpha1.Bundle
	if err := l.client.Get(ctx, client.ObjectKey{Name: name}, &bundle); err != nil {
		return nil, err
	}

	// Trust bundles can target both secrets and configmaps, we need to be able
	// to load from either.
	switch {
	case bundle.Spec.Target.ConfigMap != nil:
		return l.loadFromConfigMap(ctx, namespace, name, bundle.Spec.Target.ConfigMap.Key)
	case bundle.Spec.Target.Secret != nil:
		return l.loadFromSecret(ctx, namespace, name, bundle.Spec.Target.Secret.Key)
	default:
		return nil, fmt.Errorf("bundle has no target specified")
	}
}

func (l bundleLoader) loadFromSecret(ctx context.Context, namespace, name, key string) ([]byte, error) {
	log.FromContext(ctx).Info("loading bundle from secret", "secret_namespace", namespace, "secret_name", name, "secret_key", key)

	var secret corev1.Secret
	if err := l.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &secret); err != nil {
		return nil, err
	}

	if data, exists := secret.Data[key]; exists {
		return data, nil
	}

	return nil, fmt.Errorf("key %q does not exist in secret %s/%s", key, namespace, name)
}

func (l bundleLoader) loadFromConfigMap(ctx context.Context, namespace, name, key string) ([]byte, error) {
	log.FromContext(ctx).Info("loading bundle from secret", "configmap_namespace", namespace, "configmap_name", name, "configmap_key", key)

	var configmap corev1.ConfigMap
	if err := l.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &configmap); err != nil {
		return nil, err
	}

	if data, exists := configmap.BinaryData[key]; exists {
		return data, nil
	}

	if data, exists := configmap.Data[key]; exists {
		return []byte(data), nil
	}

	return nil, fmt.Errorf("key %q does not exist in configmap %s/%s", key, namespace, name)
}
