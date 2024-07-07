package server

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type IdentityServer struct {
	Name    string
	Version string
}

func (i *IdentityServer) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	if i.Name == "" {
		return nil, status.Error(codes.Unavailable, "driver name not configured")
	}

	if i.Version == "" {
		return nil, status.Error(codes.Unavailable, "driver is missing version")
	}

	return &csi.GetPluginInfoResponse{
		Name:          i.Name,
		VendorVersion: i.Version,
	}, nil
}

func (i *IdentityServer) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{}, nil
}

func (i *IdentityServer) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{Ready: wrapperspb.Bool(true)}, nil
}
