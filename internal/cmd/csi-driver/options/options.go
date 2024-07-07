package options

import (
	"flag"
	"fmt"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2/textlogger"
)

// Options are the main options for the approver-policy. Populated via
// processing command line flags.
type Options struct {
	// logConfig contains the logger config, including verbosity
	logConfig *textlogger.Config

	// kubeConfigFlags is used for generating a Kubernetes rest config via CLI
	// flags.
	kubeConfigFlags *genericclioptions.ConfigFlags

	// MetricsAddress is the TCP address for exposing HTTP Prometheus metrics
	// which will be served on the HTTP path '/metrics'. The value "0" will
	// disable exposing metrics.
	MetricsAddress string

	// ReadyzAddress is the TCP address for exposing the HTTP readiness probe
	// which will be served on the HTTP path '/readyz'.
	ReadyzAddress string

	// RestConfig is the shared base rest config to connect to the Kubernetes
	// API.
	RestConfig *rest.Config

	// Logr is the shared base logger.
	Logr logr.Logger

	// CSI config
	CSI config.Config
}

func New() *Options {
	return new(Options)
}

func (o *Options) Complete() error {
	log := textlogger.NewLogger(o.logConfig)
	o.Logr = log

	var err error
	o.RestConfig, err = o.kubeConfigFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes rest config: %s", err)
	}

	return nil
}

func (o *Options) AddFlags(cmd *cobra.Command) {
	var nfs cliflag.NamedFlagSets

	o.addAppFlags(nfs.FlagSet("App"))
	o.kubeConfigFlags = genericclioptions.NewConfigFlags(true)
	o.kubeConfigFlags.AddFlags(nfs.FlagSet("Kubernetes"))

	usageFmt := "Usage:\n  %s\n"
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, 0)
		return nil
	})

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), nfs, 0)
	})

	fs := cmd.Flags()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}
}

func (o *Options) addAppFlags(fs *pflag.FlagSet) {
	o.addLogFlags(fs)

	fs.StringVar(&o.MetricsAddress, "metrics-bind-address", ":9402",
		`TCP address for exposing HTTP Prometheus metrics which will be served on the HTTP path '/metrics'. The value "0" will
	 disable exposing metrics.`)

	fs.StringVar(&o.ReadyzAddress, "readiness-probe-bind-address", ":6060",
		"TCP address for exposing the HTTP readiness probe which will be served on the HTTP path '/readyz'.")

	fs.StringVar(&o.CSI.GRPCEndpoint, "endpoint", "unix://plugin/csi.sock",
		"Endpoint for exposing the CSI GRPC API.")

	fs.StringVar(&o.CSI.NodeID, "node-id", "",
		"ID of the Kubernetes node the pod is running on.")

	fs.StringVar(&o.CSI.DriverName, "driver-name", "trust-manager-csi-driver",
		"Name of the CSI driver.")

	fs.StringVar(&o.CSI.DataDir, "data-root", ":6060",
		"Directory the CSI driver uses to sync bundles into")
}

func (o *Options) addLogFlags(fs *pflag.FlagSet) {
	// Create a FlagSet, we create a new one so we can rewrite the flags
	logFs := pflag.NewFlagSet("", pflag.ContinueOnError)
	logGoFs := flag.NewFlagSet("", flag.ContinueOnError)

	// Add the flags to the logFS flagset
	o.logConfig = textlogger.NewConfig()
	o.logConfig.AddFlags(logGoFs)
	logFs.AddGoFlagSet(logGoFs)

	// Walk over the log flags, merging onto the real flagset
	logFs.VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		// Translate the "v" flag to "log-level"
		case "v":
			flag.Name = "log-level"
			flag.Usage = "Log level (1-5)."
			fs.AddFlag(flag)
		}
	})
}
