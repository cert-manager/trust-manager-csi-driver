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

package config

import "path"

const (
	MetadataFileName = "metadata"
)

// Config is the config for the CSI driver
type Config struct {
	NodeID       string
	DataDir      string
	GRPCEndpoint string
	DriverName   string
}

func (c Config) MetadataPathForVolume(id string) string {
	return path.Join(c.RootPathForVolume(id), MetadataFileName)
}

func (c Config) DataPathForVolume(id string) string {
	return path.Join(c.RootPathForVolume(id), "data")
}

func (c Config) RootPathForVolume(id string) string {
	return path.Join(c.TmpFSPath(), id)
}

func (c Config) TmpFSPath() string {
	return path.Join(c.DataDir, "tmpfs")
}
