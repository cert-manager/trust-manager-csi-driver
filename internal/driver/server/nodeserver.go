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

package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/bundlewriter"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/state"
)

type NodeServer struct {
	Config       *config.Config
	State        *state.State
	BundleWriter bundlewriter.BundleWriter

	once    sync.Once
	mounter mount.Interface

	csi.UnimplementedNodeServer
}

func (n *NodeServer) setup() {
	n.mounter = mount.New("")
}

func (n *NodeServer) NodeGetCapabilities(ctx context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			// We specify the VOLUME_MOUNT_GROUP capability sot the group will
			// be passed as part of the mount request. This is important because
			// we are constantly updating the files, so need to ensure files are
			// written with the correct group.
			//
			// See: https://kubernetes-csi.github.io/docs/support-fsgroup.html#delegate-fsgroup-to-csi-driver
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_VOLUME_MOUNT_GROUP,
					},
				},
			},
		},
	}, nil
}

func (n *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (_ *csi.NodePublishVolumeResponse, err error) {
	n.once.Do(n.setup)

	logger := log.FromContext(ctx).WithValues("volume_id", req.GetVolumeId(), "target_path", req.GetTargetPath())
	logger.Info("starting volume publish")

	// If the method fails for any reason we want to roll back the changes,
	// including:
	// - Removing from the state so the controller does not try and sync it
	// - Unmounting the bind volume
	// - Removing the root directory for the volume
	//
	// This is essentially what happens inside NodeUnpublishVolume but without
	// error checking.
	defer func() {
		if err != nil {
			_ = n.State.StopSync(req.GetVolumeId())
			_ = n.mounter.Unmount(req.GetTargetPath())
			_ = os.RemoveAll(n.Config.RootPathForVolume(req.GetVolumeId()))
		}
	}()

	// Ephemeral volumes are when the volume is defined directly in the Pod spec
	// instead of in a PVC. This is the only supported method since a PVC makes
	// no sense for out use case.
	if req.GetVolumeContext()["csi.storage.k8s.io/ephemeral"] != "true" {
		return nil, fmt.Errorf("only ephemeral volume types are supported")
	}

	// We don't want the directory to be writable, we need full control over the
	// files
	if !req.GetReadonly() {
		return nil, status.Error(codes.InvalidArgument, "pod.spec.volumes[].csi.readOnly must be set to 'true'")
	}

	// trust-manager replicates the ConfigMap/Secret into the Pods namespace, so
	// we need the Pods namespace to look up the ConfigMap/Secret
	namespace := req.GetVolumeContext()["csi.storage.k8s.io/pod.namespace"]
	if namespace == "" {
		return nil, fmt.Errorf("namespace is not set in volume context")
	}

	// trust-manager replicates the ConfigMap/Secret into the Pods namespace, so
	// we need the Pods namespace to look up the ConfigMap/Secret
	bundle := req.GetVolumeContext()["trust.cert-manager.io/bundle"]
	if namespace == "" {
		return nil, fmt.Errorf("bundle is not set in volume context")
	}

	// Since we specify the VOLUME_MOUNT_GROUP capability the group is passed to
	// us as part of the mount request. This is important because we are
	// constantly updating the files, so need to ensure files are written with
	// the correct group
	//
	// See: https://kubernetes-csi.github.io/docs/support-fsgroup.html#delegate-fsgroup-to-csi-driver
	var gid *int64
	if mount := req.GetVolumeCapability().GetMount(); mount != nil && mount.GetVolumeMountGroup() != "" {
		parsedGid, err := strconv.ParseInt(mount.GetVolumeMountGroup(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse volume_mount_group")
		}

		gid = &parsedGid
	}

	files, err := splitList(req.GetVolumeContext()["trust.cert-manager.io/concatenated-files"])
	if err != nil {
		return nil, fmt.Errorf("could not parse concatenated-files: %w", err)
	}

	hashes, err := splitList(req.GetVolumeContext()["trust.cert-manager.io/openssl-rehash"])
	if err != nil {
		return nil, fmt.Errorf("could not parse openssl-rehash: %w", err)
	}

	// Build the metadata object, this needs to contain all the information to
	// reconcile this mount.
	meta := metadata.Metadata{
		VolumeID:     req.GetVolumeId(),
		PodNamespace: namespace,
		Bundle:       bundle,
	}

	for _, p := range files {
		meta.Outputs = append(meta.Outputs, metadata.Output{
			Format: metadata.OutputFormatConcatenatedFile,
			// We use path.Join to clean any leading "../" to prevent path
			// traversal attacks
			Path: path.Join("/", p),
			GID:  gid,
		})
	}

	for _, p := range hashes {
		meta.Outputs = append(meta.Outputs, metadata.Output{
			Format: metadata.OutputFormatOpenSSLRehash,
			// We use path.Join to clean any leading "../" to prevent path
			// traversal attacks
			Path: path.Join("/", p),
			GID:  gid,
		})
	}

	if len(meta.Outputs) == 0 {
		return nil, fmt.Errorf("no outputs specified")
	}

	// Create the volume root/data directories, the data directory is what is
	// bind mounted to req.TargetPath
	logger.Info("creating volume root directory")
	if err := os.MkdirAll(n.Config.DataPathForVolume(req.GetVolumeId()), 0440); err != nil {
		return nil, fmt.Errorf("failed to create volume directory: %w", err)
	}

	// First attempt a sync, we want the data in place before the Pod starts, so
	// we sync the data before adding to state
	logger.Info("performing initial volume sync")
	if err := n.BundleWriter.Sync(ctx, meta, n.Config.DataPathForVolume(req.GetVolumeId())); err != nil {
		return nil, fmt.Errorf("failed perform initial volume sync: %w", err)
	}

	// Create bind mount from our data directory to req.TargetPath
	isMnt, err := n.mounter.IsMountPoint(req.GetTargetPath())

	if os.IsNotExist(err) {
		err = os.MkdirAll(req.GetTargetPath(), 0440)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check of path is mount volume: %w", err)
	}

	if !isMnt {
		logger.Info("creating bind mount")
		if err := n.mounter.Mount(n.Config.DataPathForVolume(req.GetVolumeId()), req.GetTargetPath(), "", []string{"bind", "ro"}); err != nil {
			return nil, err
		}
	}

	// Add to the state so the controller will reconcile changes
	logger.Info("tracking changes for volume")
	if err := n.State.Track(meta); err != nil {
		return nil, fmt.Errorf("failed to add volume to state: %w", err)
	}

	logger.Info("volume has been published")
	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	n.once.Do(n.setup)

	logger := log.FromContext(ctx).WithValues("volume_id", req.GetVolumeId(), "target_path", req.GetTargetPath())
	logger.Info("starting volume unpublish")

	// Remove the volume from the state, this will stop the controller syncing
	// the volume while we clean up.
	logger.Info("stopping sync for volume")
	if err := n.State.StopSync(req.GetVolumeId()); err != nil {
		return &csi.NodeUnpublishVolumeResponse{}, err
	}

	// Check if the target path is a mount point
	isMnt, err := n.mounter.IsMountPoint(req.GetTargetPath())
	if err != nil {
		return nil, err
	}

	// Clean up the bind mount
	if isMnt {
		logger.Info("unmounting volume")
		if err := n.mounter.Unmount(req.GetTargetPath()); err != nil {
			return nil, err
		}
	}

	// Delete the root directory for the volume, the data directory is a subdir
	// of this so gets handled by this call. The metadata file is also in this
	// directory cleaning this up.
	logger.Info("cleaning up volume")
	if err := os.RemoveAll(n.Config.RootPathForVolume(req.GetVolumeId())); err != nil && !os.IsNotExist(err) {
		return &csi.NodeUnpublishVolumeResponse{}, err
	}

	logger.Info("volume has been unpublished")
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeStageVolume not implemented")
}

func (n *NodeServer) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeUnstageVolume not implemented")
}

func (n *NodeServer) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeGetVolumeStats not implemented")
}

func (n *NodeServer) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeExpandVolume not implemented")
}

func (n *NodeServer) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: n.Config.NodeID,
	}, nil
}

func splitList(s string) ([]string, error) {
	cr := csv.NewReader(strings.NewReader(s))
	cr.TrimLeadingSpace = true
	return cr.Read()
}
