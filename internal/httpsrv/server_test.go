// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package httpsrv_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/httpsrv"
	"github.com/goabonga/infrastructure/internal/state"
)

func TestServerHealthz(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(httpsrv.New(state.NewFileStore(t.TempDir())).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("get healthz: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d", resp.StatusCode)
	}
}

func TestServerWiresVPCRoutes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(httpsrv.New(state.NewFileStore(t.TempDir())).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/vpc")
	if err != nil {
		t.Fatalf("get vpc collection: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vpc list status = %d", resp.StatusCode)
	}
}

func TestServerSecretRoutesDisabledByDefault(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(httpsrv.New(state.NewFileStore(t.TempDir())).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/secret")
	if err != nil {
		t.Fatalf("get secret collection: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("secret route should be absent without a KEK, got %d", resp.StatusCode)
	}
}

func TestServerSecretRoutesEnabledWithKEK(t *testing.T) {
	t.Parallel()

	key, _ := crypto.GenerateKey()
	kek, err := crypto.NewKEK(key)
	if err != nil {
		t.Fatalf("kek: %v", err)
	}
	srv := httptest.NewServer(httpsrv.New(state.NewFileStore(t.TempDir()), httpsrv.WithSecretEncryption(kek)).Handler())
	defer srv.Close()

	put, err := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/secret/sec-1",
		bytes.NewBufferString(`{"spec":{"data":"top-secret"}}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(put)
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put secret status = %d", resp.StatusCode)
	}
}
