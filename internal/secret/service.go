// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Package secret stores secrets encrypted at rest. Plaintext is encrypted with
// the KEK on write and never persisted or returned in clear; a deliberate
// Reveal call is the only way to recover it.
package secret

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/registry"
)

// Registry is the typed store for secrets.
type Registry = registry.Registry[resource.SecretSpec, resource.SecretStatus]

// Service encrypts secrets with a KEK and stores them in a registry.
type Service struct {
	reg *Registry
	kek *crypto.KEK
	now func() time.Time
}

// NewService returns a secret Service backed by reg and kek.
func NewService(reg *Registry, kek *crypto.KEK) *Service {
	return &Service{reg: reg, kek: kek, now: time.Now}
}

// Put encrypts plaintext and stores the secret under uid, returning the
// redacted resource (no plaintext, no ciphertext).
func (s *Service) Put(uid, name, plaintext string) (*resource.Secret, error) {
	if uid == "" {
		return nil, fmt.Errorf("secret: uid is required")
	}
	if plaintext == "" {
		return nil, fmt.Errorf("secret: data is required")
	}
	env, err := s.kek.Encrypt([]byte(plaintext))
	if err != nil {
		return nil, err
	}
	blob, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("secret: marshal envelope: %w", err)
	}

	sec := &resource.Secret{Metadata: resource.ObjectMeta{UID: uid, Name: name}}
	if existing, err := s.reg.Get(uid); err == nil {
		sec.Metadata.CreatedAt = existing.Metadata.CreatedAt
		sec.Metadata.Generation = existing.Metadata.Generation + 1
	} else {
		sec.Metadata.CreatedAt = s.now()
		sec.Metadata.Generation = 1
	}
	sec.Status.Ciphertext = blob
	sec.Status.MarkReconciled(sec.Metadata.Generation)
	sec.Status.SetPhase(resource.PhaseReady, "Encrypted", "stored encrypted at rest")

	if err := s.reg.Put(sec); err != nil {
		return nil, err
	}
	return redact(sec), nil
}

// Get returns the redacted secret (metadata and status, no secret material).
func (s *Service) Get(uid string) (*resource.Secret, error) {
	sec, err := s.reg.Get(uid)
	if err != nil {
		return nil, err
	}
	return redact(sec), nil
}

// List returns every secret, redacted.
func (s *Service) List() ([]resource.Secret, error) {
	secs, err := s.reg.List()
	if err != nil {
		return nil, err
	}
	for i := range secs {
		secs[i] = *redact(&secs[i])
	}
	return secs, nil
}

// Reveal decrypts and returns the plaintext for uid.
func (s *Service) Reveal(uid string) (string, error) {
	sec, err := s.reg.Get(uid)
	if err != nil {
		return "", err
	}
	var env crypto.Envelope
	if err := json.Unmarshal(sec.Status.Ciphertext, &env); err != nil {
		return "", fmt.Errorf("secret: unmarshal envelope: %w", err)
	}
	plaintext, err := s.kek.Decrypt(&env)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// Delete removes the secret.
func (s *Service) Delete(uid string) error {
	return s.reg.Delete(uid)
}

// redact returns a copy with all secret material stripped.
func redact(sec *resource.Secret) *resource.Secret {
	out := *sec
	out.Spec.Data = ""
	out.Status.Ciphertext = nil
	return &out
}
