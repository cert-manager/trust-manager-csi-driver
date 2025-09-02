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

package scheme

import (
	trustv1alpha1 "github.com/cert-manager/trust-manager/pkg/apis/trust/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetes "k8s.io/client-go/kubernetes/scheme"

	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata"
	metadatav1alpha1 "github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata/v1alpha1"
)

func New() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = metadata.AddToScheme(scheme)
	_ = metadatav1alpha1.AddToScheme(scheme)
	_ = trustv1alpha1.AddToScheme(scheme)
	_ = kubernetes.AddToScheme(scheme)
	return scheme
}
