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

package state

import (
	"context"
	"fmt"
	"os"
	"sync"

	"k8s.io/mount-utils"
	"k8s.io/utils/set"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata"
	"github.com/cert-manager/trust-manager-csi-driver/internal/driver/config"
)

// State contains the current state of the CSI implementation, it tracks the
// volumes currently being managed.
//
// This is its own type so both the Controller and CSI GRPC server have access
// to the state in a thread safe way.
type State struct {
	mu               sync.RWMutex
	volumeToMetadata map[string]metadata.Metadata
	bundleToVolumeID index
	metadataEncoder  ObjectEncoder[metadata.Metadata]
	config           *config.Config
}

// InitializeState will setup the persistent state required by the CSI
// implementation. It has two main jobs:
//
// 1. Ensure the tmpfs mount exists, creating if necessary
// 2. Loading config for existing volumes left over from a previous instance
func InitializeState(ctx context.Context, config *config.Config, metadataEncoder ObjectEncoder[metadata.Metadata]) (*State, error) {
	logger := log.FromContext(ctx)

	// Create empty state
	state := &State{
		volumeToMetadata: make(map[string]metadata.Metadata),
		bundleToVolumeID: index{},
		config:           config,
		metadataEncoder:  metadataEncoder,
	}

	// The volumes are stored in a tmpfs mount, this is used multiple times in
	// this function so we define a variable up here to keep it clean.
	tmpFSPath := config.TmpFSPath()

	// If the tmpfs mount does not exist, create it.
	mount := mount.New("")
	isMnt, err := mount.IsMountPoint(tmpFSPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		if err := os.MkdirAll(tmpFSPath, 0700); err != nil {
			return nil, err
		}
	}

	if isMnt {
		logger.Info("existing tmpsfs mount found", "path", tmpFSPath)
	} else {
		logger.Info("creating tmpsfs mount", "path", tmpFSPath)
		if err := mount.Mount("tmpfs", tmpFSPath, "tmpfs", []string{}); err != nil {
			return nil, fmt.Errorf("could not mount tmpfs: %w", err)
		}
	}

	// Each folder within the tmpfs path is a volume, with the folder name being
	// the volume id
	entries, err := os.ReadDir(tmpFSPath)
	if err != nil {
		return nil, fmt.Errorf("could not list volumes: %w", err)
	}

	// Attempt to load the metadata for each of the discovered folders
	for _, entry := range entries {
		// The volume id is always the folder name, the metadata path is the
		// json file within that dir containing the metadata for that volume
		volumeID := entry.Name()
		logger.Info("found existing volume", "volume_id", volumeID)

		// Read the metadata for the given volume
		metadataPath := config.MetadataPathForVolume(volumeID)
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("could not read metadata for volume %q: %w", volumeID, err)
		}

		// Decode the metadata file into the metadata object
		meta, err := metadataEncoder.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("could not decode metadata for volume %q: %w", volumeID, err)
		}

		// Insert loaded metadata into the state
		state.volumeToMetadata[volumeID] = meta
		state.bundleToVolumeID.Insert(meta.Bundle, volumeID)
	}

	return state, nil
}

// Track adds a volume to the state, meaning the controller can resume managing
// it if restarted
func (s *State) Track(meta metadata.Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encode the metadata for storage
	data, err := s.metadataEncoder.Encode(meta)
	if err != nil {
		return fmt.Errorf("could not encode metadata for volume %q: %w", meta.VolumeID, err)
	}

	// Create the metadata.json file
	metadataPath := s.config.MetadataPathForVolume(meta.VolumeID)
	//nolint:gosec // FIXME: Review required permissions
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("could not write metadata for volume %q: %w", meta.VolumeID, err)
	}

	// Add to internal map and index
	s.volumeToMetadata[meta.VolumeID] = meta
	s.bundleToVolumeID.Insert(meta.Bundle, meta.VolumeID)

	return nil
}

// StopSync removes a volume id from the state while keeping the file, this
// is used to make the controller stop syncing a volume while it is being
// cleaned up.
//
// This explicitly does not delete the persisted metadata file.
func (s *State) StopSync(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from internal map and index
	if meta, exists := s.volumeToMetadata[id]; exists {
		s.bundleToVolumeID.Delete(meta.Bundle, id)
	}

	return nil
}

// GetMetadataForBundle returns the metadata for a given bundle name, each
// metadata item relates to a single volume that may require a re-sync.
func (s *State) GetMetadataForBundle(name string) []metadata.Metadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.bundleToVolumeID[name]
	meta := make([]metadata.Metadata, 0, len(ids))
	for id := range ids {
		meta = append(meta, s.volumeToMetadata[id])
	}
	return meta
}

type index map[string]set.Set[string]

func (i index) Insert(k string, v string) {
	// Get set at given key
	existing, exists := i[k]
	if !exists {
		i[k] = set.New(v)
		return
	}

	// Insert into the underlying set
	existing.Insert(v)
}

func (i index) Delete(k string, v string) {
	// If there is nothing under the given key, there is nothing to delete
	existing, exists := i[k]
	if !exists {
		return
	}

	// Delete from the set
	existing.Delete(v)

	// If the set is now empty, we don't need it anymore and it can be cleaned
	// up
	if existing.Len() == 0 {
		delete(i, k)
	}
}
