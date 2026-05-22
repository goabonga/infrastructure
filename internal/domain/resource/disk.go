// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package resource

import "fmt"

// Disk resource kinds.
const (
	KindDisk     = "disk"
	KindDiskFile = "disk_file"
)

// DiskSpec is the desired state of a persistent disk. Setting KMSKeyID encrypts
// it at rest via dm-crypt with that key.
type DiskSpec struct {
	Name     string `json:"name,omitempty"`
	SizeMB   int    `json:"sizeMb"`
	KMSKeyID string `json:"kmsKeyId,omitempty"`
}

// Validate reports whether the spec is well-formed.
func (s DiskSpec) Validate() error {
	if s.SizeMB <= 0 {
		return fmt.Errorf("disk: sizeMb must be positive")
	}
	return nil
}

// DiskStatus is the observed state of a disk.
type DiskStatus struct {
	StatusBase
	Encrypted bool   `json:"encrypted"`
	Path      string `json:"path,omitempty"`
}

// Disk is a persistent-disk resource.
type Disk = Resource[DiskSpec, DiskStatus]

// DiskFileSpec injects a file into a disk's filesystem.
type DiskFileSpec struct {
	DiskID  string `json:"diskId"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode,omitempty"`
}

// Validate reports whether the spec is well-formed.
func (s DiskFileSpec) Validate() error {
	if s.DiskID == "" {
		return fmt.Errorf("disk_file: diskId is required")
	}
	if s.Path == "" {
		return fmt.Errorf("disk_file: path is required")
	}
	return nil
}

// DiskFileStatus is the observed state of an injected file.
type DiskFileStatus struct {
	StatusBase
}

// DiskFile is an injected-file resource.
type DiskFile = Resource[DiskFileSpec, DiskFileStatus]
