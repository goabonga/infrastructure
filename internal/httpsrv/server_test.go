// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package httpsrv_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
