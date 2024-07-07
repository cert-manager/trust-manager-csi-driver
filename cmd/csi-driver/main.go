package main

import (
	"os"

	csidriver "github.com/cert-manager/trust-manager-csi-driver/internal/cmd/csi-driver"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	if err := csidriver.NewCommand().ExecuteContext(signals.SetupSignalHandler()); err != nil {
		os.Exit(1)
	}
}
