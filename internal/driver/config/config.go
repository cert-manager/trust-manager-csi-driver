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
