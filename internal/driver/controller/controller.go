package controller

import (
	"context"

	trustapi "github.com/cert-manager/trust-manager/pkg/apis/trust/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/bundlewriter"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/state"
)

type Reconciler struct {
	Config       *config.Config
	State        *state.State
	Client       client.Client
	BundleWriter bundlewriter.BundleWriter
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Sync the volume, collect any errors into a slice.
	errs := []error{}
	for _, meta := range r.State.GetMetadataForBundle(req.Name) {
		ctx := log.IntoContext(ctx,
			log.FromContext(ctx).
				WithValues(
					"volume_id", meta.VolumeID,
				),
		)

		if err := r.BundleWriter.Sync(ctx, meta, r.Config.DataPathForVolume(meta.VolumeID)); err != nil {
			errs = append(errs, err)
		}
	}

	// Return the error aggregate
	return ctrl.Result{}, errors.NewAggregate(errs)
}

func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.bundleForSecretOrConfigMap)).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.bundleForSecretOrConfigMap)).
		Named("bundle").
		Complete(r)
}

func (r *Reconciler) bundleForSecretOrConfigMap(ctx context.Context, obj client.Object) []reconcile.Request {
	labels := obj.GetLabels()
	if bundle, exists := labels[trustapi.BundleLabelKey]; exists {
		return []reconcile.Request{{NamespacedName: client.ObjectKey{Name: bundle}}}
	}

	return nil
}
