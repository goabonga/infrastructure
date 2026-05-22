// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package secret_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/secret"
	"github.com/goabonga/infrastructure/internal/state"
)

func newService(t *testing.T) *secret.Service {
	t.Helper()
	key, _ := crypto.GenerateKey()
	kek, err := crypto.NewKEK(key)
	if err != nil {
		t.Fatalf("kek: %v", err)
	}
	reg := registry.New[resource.SecretSpec, resource.SecretStatus](state.NewFileStore(t.TempDir()), resource.KindSecret)
	return secret.NewService(reg, kek)
}

func TestSecretPutRedactsAndReveals(t *testing.T) {
	t.Parallel()

	svc := newService(t)
	out, err := svc.Put("sec-1", "db-password", "s3cr3t")
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if out.Spec.Data != "" || out.Status.Ciphertext != nil {
		t.Fatalf("put response leaks secret material: %+v", out)
	}
	if !out.Status.IsReady() || out.Metadata.Generation != 1 {
		t.Fatalf("unexpected meta/status: %+v %+v", out.Metadata, out.Status)
	}

	got, err := svc.Get("sec-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Spec.Data != "" || got.Status.Ciphertext != nil {
		t.Fatal("get response leaks secret material")
	}

	plaintext, err := svc.Reveal("sec-1")
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if plaintext != "s3cr3t" {
		t.Fatalf("reveal = %q, want s3cr3t", plaintext)
	}
}

func TestSecretNotStoredInClear(t *testing.T) {
	t.Parallel()

	key, _ := crypto.GenerateKey()
	kek, _ := crypto.NewKEK(key)
	store := state.NewFileStore(t.TempDir())
	reg := registry.New[resource.SecretSpec, resource.SecretStatus](store, resource.KindSecret)
	svc := secret.NewService(reg, kek)

	if _, err := svc.Put("sec-1", "", "p@ssw0rd-plaintext"); err != nil {
		t.Fatalf("put: %v", err)
	}
	// Read the raw stored bytes: the plaintext must not appear.
	raw, err := store.Get("secret/sec-1")
	if err != nil {
		t.Fatalf("raw get: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("nothing stored")
	}
	if bytes.Contains(raw, []byte("p@ssw0rd-plaintext")) {
		t.Fatal("plaintext stored at rest")
	}
}

func TestSecretPutBumpsGeneration(t *testing.T) {
	t.Parallel()

	svc := newService(t)
	if _, err := svc.Put("sec-1", "n", "v1"); err != nil {
		t.Fatalf("put v1: %v", err)
	}
	out, err := svc.Put("sec-1", "n", "v2")
	if err != nil {
		t.Fatalf("put v2: %v", err)
	}
	if out.Metadata.Generation != 2 {
		t.Fatalf("generation = %d, want 2", out.Metadata.Generation)
	}
	if rev, _ := svc.Reveal("sec-1"); rev != "v2" {
		t.Fatalf("reveal = %q, want v2", rev)
	}
}

func TestSecretValidationAndDelete(t *testing.T) {
	t.Parallel()

	svc := newService(t)
	if _, err := svc.Put("", "n", "v"); err == nil {
		t.Fatal("expected error for empty uid")
	}
	if _, err := svc.Put("sec-1", "n", ""); err == nil {
		t.Fatal("expected error for empty data")
	}

	if _, err := svc.Put("sec-1", "n", "v"); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := svc.Delete("sec-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get("sec-1"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
