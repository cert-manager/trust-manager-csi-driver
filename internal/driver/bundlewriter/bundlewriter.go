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
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"path"
	"strings"

	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata"
	x509util "github.com/cert-manager/trust-manager-csi-driver/internal/utils/x509"
	volumeutil "github.com/cert-manager/trust-manager-csi-driver/third_party/k8s.io/kubernetes/pkg/volume/util"
)

// BundleWriter is used to write a bundle to a directory
type BundleWriter struct {
	FileWriter   FileWriter
	BundleLoader BundleLoader
}

func NewBundleWriter(loader BundleLoader, writer FileWriter) BundleWriter {
	return BundleWriter{
		FileWriter:   writer,
		BundleLoader: loader,
	}
}

// Sync will update the target directory with the latest bundle contents
func (s BundleWriter) Sync(ctx context.Context, meta metadata.Metadata, target string) error {
	// Load the bundle, this should return a slice containing PEM bundles
	bundle, err := s.BundleLoader.Load(ctx, meta.PodNamespace, meta.Bundle)
	if err != nil {
		return err
	}

	// Build payload for the file writer
	payload := map[string]volumeutil.FileProjection{}
	for _, output := range meta.Outputs {
		var err error

		switch output.Format {
		case metadata.OutputFormatConcatenatedFile:
			err = s.addConcatenatedFileToPayload(bundle, output, payload)
		case metadata.OutputFormatOpenSSLRehash:
			err = s.addRehashFilesToPayload(bundle, output, payload)
		}

		if err != nil {
			return err
		}
	}

	// Write the files to disk
	if err := s.FileWriter.Write(ctx, target, payload); err != nil {
		return err
	}

	return nil
}

func (s BundleWriter) addConcatenatedFileToPayload(bundle []byte, output metadata.Output, payload map[string]volumeutil.FileProjection) error {
	buffer := new(bytes.Buffer)

	// Instead of using the bundle directly we mimic other bundle formats by
	// adding comments containing the subject info.
	//
	// This also serves as a way to filter the bundle, keeping only certificates
	// and validating each certificate is valid.
	err := x509util.ForEachCertInBundle(bundle, func(cert *x509.Certificate, pem []byte) error {
		// Errors are ignored here since it is pretty much impossible for a
		// bytes.Buffer to error on Write.
		_, _ = fmt.Fprintf(buffer, "\n# %s\n", cert.Subject)
		_, _ = buffer.Write(pem)
		return nil
	})

	// Any error returned will be an error parsing a certificate
	if err != nil {
		return err
	}

	// Add the constructed bundle (with the new comments) to the payload, we
	// also trim spaces and ensure we end on a trailing new line.
	fpath := strings.TrimLeft(output.Path, "/")
	payload[fpath] = volumeutil.FileProjection{
		Data:    append(bytes.TrimSpace(buffer.Bytes()), '\n'),
		Mode:    0440,
		FsUser:  output.UID,
		FsGroup: output.GID,
	}

	return nil
}

func (s BundleWriter) addRehashFilesToPayload(bundle []byte, output metadata.Output, payload map[string]volumeutil.FileProjection) error {
	count := map[string]int{}
	return x509util.ForEachCertInBundle(bundle, func(cert *x509.Certificate, pem []byte) error {
		// Hash the subject
		hash, err := x509util.CertificateSubjectHash(cert)
		if err != nil {
			return err
		}

		// Build the filename and path, the filename is in the format
		// "<hash>.<count>", the count is used to handle hash collisions which
		// may happen as the hash is a truncated sha1.
		fname := fmt.Sprintf("%s.%d", hash, count[hash])
		fpath := strings.TrimLeft(path.Join(output.Path, fname), "/")

		// Increment the count
		count[hash]++

		// Add to the payload
		payload[fpath] = volumeutil.FileProjection{
			Data:    pem,
			Mode:    0440,
			FsUser:  output.UID,
			FsGroup: output.GID,
		}

		return nil
	})
}
