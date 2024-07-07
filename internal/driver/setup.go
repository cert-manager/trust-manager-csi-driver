package driver

import (
	"context"
	"fmt"

	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata"
	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata/v1alpha1"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/bundlewriter"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/controller"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/server"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/state"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Setup(ctx context.Context, mgr ctrl.Manager, config *config.Config) error {
	metadataEncoder, err := state.NewVersionedObjectEncoder[metadata.Metadata, v1alpha1.Metadata](mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("could not create object encoder for volume metadata: %w", err)
	}

	state, err := state.InitializeState(ctx, config, metadataEncoder)
	if err != nil {
		return fmt.Errorf("could not initialize state: %w", err)
	}

	bundleWriter := bundlewriter.NewBundleWriter(
		bundlewriter.NewBundleLoader(mgr.GetClient()),
		bundlewriter.NewAtomicFileWriter(),
	)

	if err := server.Setup(mgr, config, state, bundleWriter); err != nil {
		return fmt.Errorf("could not setup grpc server: %w", err)
	}

	if err := controller.Setup(mgr, config, state, bundleWriter); err != nil {
		return fmt.Errorf("could not setup controller: %w", err)
	}

	return nil
}
