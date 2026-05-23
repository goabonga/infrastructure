// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/handler"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/secret"
	"github.com/goabonga/infrastructure/internal/state"
)

func newSecretMux(t *testing.T) *http.ServeMux {
	t.Helper()
	key, _ := crypto.GenerateKey()
	kek, err := crypto.NewKEK(key)
	if err != nil {
		t.Fatalf("kek: %v", err)
	}
	store := state.NewFileStore(t.TempDir())
	reg := registry.New[resource.SecretSpec, resource.SecretStatus](store, resource.KindSecret)
	versions := registry.New[resource.SecretVersionSpec, resource.SecretVersionStatus](store, resource.KindSecretVersion)
	mux := http.NewServeMux()
	handler.NewSecretHandler(secret.NewService(reg, versions, kek)).Register(mux, "/api/v1")
	return mux
}

func TestSecretHandlerLifecycle(t *testing.T) {
	t.Parallel()

	mux := newSecretMux(t)

	// Create.
	body := resource.Secret{Metadata: resource.ObjectMeta{Name: "db"}, Spec: resource.SecretSpec{Data: "s3cr3t"}}
	rec := do(t, mux, http.MethodPut, "/api/v1/secret/sec-1", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, body %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "s3cr3t") || strings.Contains(rec.Body.String(), "ciphertext") {
		t.Fatalf("put response leaks secret material: %s", rec.Body)
	}

	// Get is redacted.
	rec = do(t, mux, http.MethodGet, "/api/v1/secret/sec-1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "s3cr3t") {
		t.Fatalf("get leaks plaintext: %s", rec.Body)
	}

	// Reveal returns plaintext.
	rec = do(t, mux, http.MethodGet, "/api/v1/secret/sec-1/reveal", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("reveal status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "s3cr3t") {
		t.Fatalf("reveal missing plaintext: %s", rec.Body)
	}

	// Delete then 404.
	rec = do(t, mux, http.MethodDelete, "/api/v1/secret/sec-1", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}
	rec = do(t, mux, http.MethodGet, "/api/v1/secret/sec-1", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d", rec.Code)
	}
}

func TestSecretHandlerListRedacts(t *testing.T) {
	t.Parallel()

	mux := newSecretMux(t)
	for _, uid := range []string{"sec-1", "sec-2"} {
		body := resource.Secret{Spec: resource.SecretSpec{Data: "plaintext-" + uid}}
		if rec := do(t, mux, http.MethodPut, "/api/v1/secret/"+uid, body); rec.Code != http.StatusOK {
			t.Fatalf("put %s: %d", uid, rec.Code)
		}
	}

	rec := do(t, mux, http.MethodGet, "/api/v1/secret", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "plaintext-") || strings.Contains(rec.Body.String(), "ciphertext") {
		t.Fatalf("list leaks secret material: %s", rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "sec-1") || !strings.Contains(rec.Body.String(), "sec-2") {
		t.Fatalf("list missing entries: %s", rec.Body)
	}
}

func TestSecretHandlerRejectsEmptyData(t *testing.T) {
	t.Parallel()

	mux := newSecretMux(t)
	rec := do(t, mux, http.MethodPut, "/api/v1/secret/sec-1", resource.Secret{Spec: resource.SecretSpec{}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty-data status = %d, want 400", rec.Code)
	}
}
