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

	volumeutil "github.com/cert-manager/trust-manager-csi-driver/third_party/k8s.io/kubernetes/pkg/volume/util"
)

// FileProjection contains file data
type FileProjection = volumeutil.FileProjection

// FileWriter is used to write files to a given directory, this will wipe
// existing files and replace them with the given payload.
type FileWriter interface {
	Write(ctx context.Context, target string, payload map[string]FileProjection) error
}

// NewAtomicFileWriter returns a FileWriter that will replace the contents of a
// directory in an atomic manner.
func NewAtomicFileWriter() FileWriter {
	return atomicWriter{}
}

type atomicWriter struct{}

func (w atomicWriter) Write(ctx context.Context, target string, payload map[string]FileProjection) error {
	atomicWriter, err := volumeutil.NewAtomicWriter(target, "trust-manager-csi-driver")
	if err != nil {
		return err
	}

	return atomicWriter.Write(payload, nil)
}
