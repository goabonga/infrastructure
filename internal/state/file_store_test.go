// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package state_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/goabonga/infrastructure/internal/state"
)

func TestFileStorePutGetRoundtrip(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())
	if err := fs.Put("vpcs/vpc-1", []byte("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := fs.Get("vpcs/vpc-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestFileStoreGetMissing(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())
	_, err := fs.Get("vpcs/none")
	if !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFileStoreOverwrite(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())
	if err := fs.Put("k", []byte("v1")); err != nil {
		t.Fatalf("put v1: %v", err)
	}
	if err := fs.Put("k", []byte("v2")); err != nil {
		t.Fatalf("put v2: %v", err)
	}
	got, err := fs.Get("k")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "v2" {
		t.Fatalf("got %q, want v2", got)
	}
}

func TestFileStoreDelete(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())
	if err := fs.Put("k", []byte("v")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := fs.Delete("k"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := fs.Get("k"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	// Deleting a missing key is not an error.
	if err := fs.Delete("k"); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}

func TestFileStoreList(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())

	// Listing a non-existent prefix yields nothing, not an error.
	kvs, err := fs.List("vpcs")
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(kvs) != 0 {
		t.Fatalf("expected no entries, got %d", len(kvs))
	}

	for _, uid := range []string{"vpc-1", "vpc-2", "vpc-3"} {
		if err := fs.Put("vpcs/"+uid, []byte(uid)); err != nil {
			t.Fatalf("put %s: %v", uid, err)
		}
	}
	// A nested prefix must not appear in the parent listing.
	if err := fs.Put("vpcs/sub/vpc-x", []byte("x")); err != nil {
		t.Fatalf("put nested: %v", err)
	}

	kvs, err = fs.List("vpcs")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	keys := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		keys = append(keys, kv.Key)
	}
	sort.Strings(keys)
	want := []string{"vpcs/vpc-1", "vpcs/vpc-2", "vpcs/vpc-3"}
	if len(keys) != len(want) {
		t.Fatalf("got keys %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("key[%d] = %q, want %q", i, keys[i], want[i])
		}
	}
}

func TestFileStoreCompareAndSwap(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())

	// Create only when absent (nil oldValue).
	ok, err := fs.CompareAndSwap("k", nil, []byte("v1"))
	if err != nil || !ok {
		t.Fatalf("create cas: ok=%v err=%v", ok, err)
	}

	// Stale oldValue is rejected.
	ok, err = fs.CompareAndSwap("k", []byte("stale"), []byte("v2"))
	if err != nil {
		t.Fatalf("mismatch cas err: %v", err)
	}
	if ok {
		t.Fatal("expected mismatch cas to fail")
	}

	// Correct oldValue swaps.
	ok, err = fs.CompareAndSwap("k", []byte("v1"), []byte("v2"))
	if err != nil || !ok {
		t.Fatalf("swap cas: ok=%v err=%v", ok, err)
	}
	got, _ := fs.Get("k")
	if string(got) != "v2" {
		t.Fatalf("got %q, want v2", got)
	}
}

func TestFileStoreRejectsEmptyKey(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())
	if err := fs.Put("", []byte("v")); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestFileStoreKeyCannotEscapeRoot(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := state.NewFileStore(base)

	if err := fs.Put("../escape", []byte("v")); err != nil {
		t.Fatalf("put traversal key: %v", err)
	}
	// The write must land inside base, not in its parent.
	if _, err := os.Stat(filepath.Join(base, "escape")); err != nil {
		t.Fatalf("expected file inside base: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(base), "escape")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("traversal key escaped the store root")
	}
}

func TestFileStoreClose(t *testing.T) {
	t.Parallel()

	fs := state.NewFileStore(t.TempDir())
	if err := fs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
