// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package state_test

import (
	"testing"

	"github.com/goabonga/infrastructure/internal/state"
)

func TestOpenFileStore(t *testing.T) {
	t.Parallel()

	// With no DSN, Open returns a working file-backed store.
	store, err := state.Open(t.TempDir(), "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Put("k", []byte("v")); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := store.Get("k")
	if err != nil || string(got) != "v" {
		t.Fatalf("get = %q, %v", got, err)
	}
}
