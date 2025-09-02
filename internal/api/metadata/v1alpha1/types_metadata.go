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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Metadata contains the stored metadata for a given volume mount, it is
// versioned to ensure an upgrade will always be able to load metadata
type Metadata struct {
	metav1.TypeMeta `json:",inline"`

	// VolumeID is the ID passed to the CSI driver in the NodePublish request.
	VolumeID string `json:"volumeID"`
	// PodNamespace is the namespace of the pod being mounted into
	PodNamespace string `json:"podNamespace"`
	// Bundle is the trust bundle to mount
	Bundle string `json:"bundle"`
	// Outputs defines the output formats
	Outputs []Output `json:"outputs"`
}

// Output defines an output for a given CSI trust bundle mount
type Output struct {
	// Format to write the certificate bundle
	Format OutputFormat `json:"format"`

	// Owner of the files
	UID *int64 `json:"uid,omitempty"`
	GID *int64 `json:"gid,omitempty"`

	// Path to the file or directory.
	// For outputs that produce a single file this must be a path to the file,
	// outputs that produce multiple files this will be the path to the
	// directory
	Path string
}

// OutputFormat defines the format to write the certificate bundle
type OutputFormat string

const (
	// Output files in the OpenSSL rehash format, see
	// https://manpages.ubuntu.com/manpages/noble/en/man1/c_rehash.1ssl.html
	OutputFormatOpenSSLRehash = "OpenSSLRehash"
	// Output a single concatenated file
	OutputFormatConcatenatedFile = "ConcatenatedFile"
)
