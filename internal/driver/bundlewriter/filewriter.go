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
	atomicWriter, err := volumeutil.NewAtomicWriter(target)
	if err != nil {
		return err
	}

	return atomicWriter.Write(ctx, payload, nil)
}
