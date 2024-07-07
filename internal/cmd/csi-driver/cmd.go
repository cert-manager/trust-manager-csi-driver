package csidriver

import (
	"fmt"

	"github.com/cert-manager/trust-manager-csi-driver/internal/cmd/csi-driver/options"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver"
	"github.com/cert-manager/trust-manager-csi-driver/internal/scheme"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	trustv1alpha1 "github.com/cert-manager/trust-manager/pkg/apis/trust/v1alpha1"
)

const (
	helpOutput = "A CSI driver to mount trust bundles into pods."
)

// NewCommand returns an new command instance of approver-policy.
func NewCommand() *cobra.Command {
	opts := new(options.Options)

	cmd := &cobra.Command{
		Use:   "trust-manager-csi-driver",
		Short: helpOutput,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Complete()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			log.Log = opts.Logr.WithName("apiutil")
			log := opts.Logr.WithName("main")
			mlog := opts.Logr.WithName("controller-manager")
			ctrl.SetLogger(mlog)

			mustHaveBundleLabelKeyRequirement, err := labels.NewRequirement(trustv1alpha1.BundleLabelKey, selection.Exists, nil)
			if err != nil {
				return fmt.Errorf("invalid label selector: %w", err)
			}

			trustBundleLabelSelector := labels.NewSelector().Add(*mustHaveBundleLabelKeyRequirement)

			mgr, err := ctrl.NewManager(opts.RestConfig, ctrl.Options{
				Scheme: scheme.New(),
				Cache: cache.Options{
					ByObject: map[client.Object]cache.ByObject{
						&corev1.Secret{}: {
							Label: trustBundleLabelSelector,
						},
						&corev1.ConfigMap{}: {
							Label: trustBundleLabelSelector,
						},
					},
				},
				ReadinessEndpointName:  "/readyz",
				HealthProbeBindAddress: opts.ReadyzAddress,
				Metrics: server.Options{
					BindAddress: opts.MetricsAddress,
				},
				Logger: mlog,
			})

			if err != nil {
				return fmt.Errorf("unable to create controller manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
				return fmt.Errorf("unable to add readyz check: %w", err)
			}

			if err := driver.Setup(ctx, mgr, &opts.CSI); err != nil {
				return fmt.Errorf("unable to setup csi driver: %w", err)
			}

			log.Info("starting csi-driver...")
			return mgr.Start(ctx)
		},
	}

	opts.AddFlags(cmd)

	return cmd
}
