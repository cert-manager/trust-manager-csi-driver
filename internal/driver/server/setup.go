package server

import (
	"context"
	"net"
	"strings"

	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/bundlewriter"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/state"
	"github.com/cert-manager/trust-manager-csi-driver/internal/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var grpcMetrics = grpcPrometheus.NewServerMetrics()

func init() {
	metrics.Registry.MustRegister(grpcMetrics)
}

func Setup(mgr ctrl.Manager, config *config.Config, state *state.State, bw bundlewriter.BundleWriter) error {
	return mgr.Add(
		manager.RunnableFunc(func(ctx context.Context) error {
			// Ensure we don't leak any goroutines by canceling the context on function
			// return
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			// Create listener for the server
			listener, err := net.Listen(parseEndpoint(config.GRPCEndpoint))
			if err != nil {
				return err
			}

			// Get the logger from the context
			logger := log.FromContext(ctx)

			// Create server interceptors
			unaryInterceptor := grpc.ChainUnaryInterceptor(
				// Prometheus metric for GRPC endpoints
				grpcMetrics.UnaryServerInterceptor(),
				// Inject logger into context
				func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
					// Build the logger for the request and inject it into the
					// context
					logger := logger.WithValues("method", info.FullMethod, "request_id", uuid.NewUUID(), "request", protosanitizer.StripSecrets(req))
					ctx = log.IntoContext(ctx, logger)

					// Call the handler, if there is an error we log it
					logger.V(2).Info("starting request")
					resp, err = handler(ctx, req)
					if err != nil {
						logger.Error(err, "failed processing request")
					} else {
						logger.V(2).Info("request completed", "response", protosanitizer.StripSecrets(resp))
					}

					return resp, err
				},
			)

			// Create a new GRPC server
			server := grpc.NewServer(unaryInterceptor)

			// Register all services on the GRPC server
			csi.RegisterNodeServer(server, &NodeServer{Config: config, State: state, BundleWriter: bw})
			csi.RegisterIdentityServer(server, &IdentityServer{Name: config.DriverName, Version: version.AppVersion})

			// Initialize prometheus metrics. This MUST be called after all services are
			// registered on the server.
			grpcMetrics.InitializeMetrics(server)

			// When the golang context is canceled, shut down the GRPC server
			go func() {
				<-ctx.Done()
				server.GracefulStop()
			}()

			// Serve requests
			return server.Serve(listener)
		}))
}

func parseEndpoint(endpoint string) (proto, addr string) {
	parts := strings.SplitN(endpoint, "://", 2)
	if len(parts) == 1 {
		return "tcp", endpoint
	}

	return parts[0], parts[1]
}
