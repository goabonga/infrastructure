// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package resource

import "fmt"

// KindSecret is the resource kind for secrets.
const KindSecret = "secret"

// SecretSpec is the desired state of a secret. Data carries the plaintext on
// write only; it is never persisted or returned in clear.
type SecretSpec struct {
	Data string `json:"data,omitempty"`
}

// Validate reports whether the spec is well-formed for a write.
func (s SecretSpec) Validate() error {
	if s.Data == "" {
		return fmt.Errorf("secret: data is required")
	}
	return nil
}

// SecretStatus is the observed state of a secret. Ciphertext is the envelope
// (encrypted) representation stored at rest.
type SecretStatus struct {
	StatusBase
	Ciphertext []byte `json:"ciphertext,omitempty"`
}

// Secret is an encrypted-at-rest secret resource.
type Secret = Resource[SecretSpec, SecretStatus]
