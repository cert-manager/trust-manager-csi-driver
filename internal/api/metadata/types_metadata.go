package metadata

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Metadata contains the stored metadata for a given volume mount, it is
// versioned to ensure an upgrade will always be able to load metadata
type Metadata struct {
	metav1.TypeMeta

	// VolumeID is the ID passed to the CSI driver in the NodePublish request.
	VolumeID string
	// PodNamespace is the namespace of the pod being mounted into
	PodNamespace string
	// Bundle is the trust bundle to mount
	Bundle string
	// Outputs defines the output formats
	Outputs []Output
}

func (m Metadata) GetName() string {
	return m.VolumeID
}

// Output defines an output for a given CSI trust bundle mount
type Output struct {
	// Format to write the certificate bundle
	Format OutputFormat
	// Owner of the files
	UID, GID *int64
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
