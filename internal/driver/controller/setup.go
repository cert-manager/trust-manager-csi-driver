package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/bundlewriter"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/state"
)

func Setup(mgr ctrl.Manager, config *config.Config, state *state.State, bw bundlewriter.BundleWriter) error {
	return (&Reconciler{
		Config:       config,
		Client:       mgr.GetClient(),
		BundleWriter: bw,
		State:        state,
	}).SetupWithManager(mgr)
}
