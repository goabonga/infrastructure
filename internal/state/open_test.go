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

func TestOpenEtcdScheme(t *testing.T) {
	t.Parallel()

	// An "etcd://" DSN selects the etcd backend. The client dials lazily, so
	// Open succeeds without a reachable server.
	store, err := state.Open("", "etcd://127.0.0.1:2379,127.0.0.1:12379")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	if _, ok := store.(*state.EtcdStore); !ok {
		t.Fatalf("Open returned %T, want *state.EtcdStore", store)
	}
}
