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

package driver

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata"
	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata/v1alpha1"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/bundlewriter"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/controller"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/server"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/state"
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
