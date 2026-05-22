// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package ssl_test

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/ssl"
	"github.com/goabonga/infrastructure/internal/state"
)

func newService(t *testing.T) (*ssl.Service, state.Store) {
	t.Helper()
	key, _ := crypto.GenerateKey()
	kek, err := crypto.NewKEK(key)
	if err != nil {
		t.Fatalf("kek: %v", err)
	}
	store := state.NewFileStore(t.TempDir())
	reg := registry.New[resource.SSLCASpec, resource.SSLCAStatus](store, resource.KindSSLCA)
	return ssl.NewService(reg, kek), store
}

func TestCreateCARedactsKeyButKeepsCert(t *testing.T) {
	t.Parallel()

	svc, store := newService(t)
	ca, err := svc.CreateCA("ca-1", "root", resource.SSLCASpec{CommonName: "infra root"})
	if err != nil {
		t.Fatalf("create ca: %v", err)
	}
	if len(ca.Status.CertPEM) == 0 {
		t.Fatal("cert PEM should be present")
	}
	if ca.Status.EncryptedKey != nil {
		t.Fatal("response should not expose the encrypted key")
	}
	if !ca.Status.IsReady() {
		t.Fatalf("phase = %q, want Ready", ca.Status.Phase)
	}

	// The private key must not be readable in clear on disk.
	raw, err := store.Get("ssl_ca/ca-1")
	if err != nil {
		t.Fatalf("raw get: %v", err)
	}
	if bytes.Contains(raw, []byte("EC PRIVATE KEY")) {
		t.Fatal("private key stored in clear")
	}
}

func TestIssueCertVerifiesAgainstCA(t *testing.T) {
	t.Parallel()

	svc, _ := newService(t)
	ca, err := svc.CreateCA("ca-1", "root", resource.SSLCASpec{CommonName: "infra root"})
	if err != nil {
		t.Fatalf("create ca: %v", err)
	}

	certPEM, keyPEM, err := svc.Issue("ca-1", "web", []string{"example.com"}, 30)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if block, _ := pem.Decode(keyPEM); block == nil {
		t.Fatal("invalid leaf key PEM")
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca.Status.CertPEM) {
		t.Fatal("add CA to pool")
	}
	block, _ := pem.Decode(certPEM)
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: "example.com"}); err != nil {
		t.Fatalf("leaf should verify against the CA: %v", err)
	}
}

func TestSSLValidationListDelete(t *testing.T) {
	t.Parallel()

	svc, _ := newService(t)
	if _, err := svc.CreateCA("", "n", resource.SSLCASpec{CommonName: "x"}); err == nil {
		t.Fatal("expected error for empty uid")
	}
	if _, err := svc.CreateCA("ca-1", "n", resource.SSLCASpec{}); err == nil {
		t.Fatal("expected error for empty common name")
	}
	if _, _, err := svc.Issue("ca-1", "", nil, 0); err == nil {
		t.Fatal("expected error for empty cert common name")
	}

	if _, err := svc.CreateCA("ca-1", "n", resource.SSLCASpec{CommonName: "root"}); err != nil {
		t.Fatalf("create ca: %v", err)
	}
	cas, err := svc.List()
	if err != nil || len(cas) != 1 {
		t.Fatalf("list = %d %v", len(cas), err)
	}
	if cas[0].Status.EncryptedKey != nil {
		t.Fatal("list should redact the key")
	}
	if err := svc.Delete("ca-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get("ca-1"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
