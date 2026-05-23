// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdStore implements Store on etcd v3, a highly available multi-instance
// backend. Keys map directly to etcd keys; CompareAndSwap is a single etcd
// transaction, so it is atomic across instances without an external lock.
type EtcdStore struct {
	client *clientv3.Client
}

// NewEtcdStore connects to the given etcd endpoints (host:port). The client
// dials lazily, so this returns without a live connection.
func NewEtcdStore(endpoints []string) (*EtcdStore, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("state: connect etcd: %w", err)
	}
	return &EtcdStore{client: client}, nil
}

// Get returns the value at key, or ErrNotFound.
func (s *EtcdStore) Get(key string) ([]byte, error) {
	resp, err := s.client.Get(context.Background(), key)
	if err != nil {
		return nil, fmt.Errorf("state: etcd get %q: %w", key, err)
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}
	return resp.Kvs[0].Value, nil
}

// Put stores value at key.
func (s *EtcdStore) Put(key string, value []byte) error {
	if _, err := s.client.Put(context.Background(), key, string(value)); err != nil {
		return fmt.Errorf("state: etcd put %q: %w", key, err)
	}
	return nil
}

// Delete removes key. A missing key is not an error.
func (s *EtcdStore) Delete(key string) error {
	if _, err := s.client.Delete(context.Background(), key); err != nil {
		return fmt.Errorf("state: etcd delete %q: %w", key, err)
	}
	return nil
}

// List returns the key-value pairs directly under prefix (one segment deeper),
// matching the file store's single-level semantics.
func (s *EtcdStore) List(prefix string) ([]KeyValue, error) {
	scan := prefix + "/"
	resp, err := s.client.Get(context.Background(), scan,
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return nil, fmt.Errorf("state: etcd list %q: %w", prefix, err)
	}
	var kvs []KeyValue
	for _, kv := range resp.Kvs {
		// Keep only direct children: the remainder after prefix+"/" must not
		// contain a further separator.
		if strings.Contains(string(kv.Key)[len(scan):], "/") {
			continue
		}
		kvs = append(kvs, KeyValue{Key: string(kv.Key), Value: kv.Value})
	}
	return kvs, nil
}

// CompareAndSwap atomically replaces the value at key with newValue only if the
// current value equals oldValue (nil oldValue means "expect absent").
func (s *EtcdStore) CompareAndSwap(key string, oldValue, newValue []byte) (bool, error) {
	var cmp clientv3.Cmp
	if oldValue == nil {
		// Expect absent: a key that has never been created has revision 0.
		cmp = clientv3.Compare(clientv3.CreateRevision(key), "=", 0)
	} else {
		cmp = clientv3.Compare(clientv3.Value(key), "=", string(oldValue))
	}
	resp, err := s.client.Txn(context.Background()).
		If(cmp).
		Then(clientv3.OpPut(key, string(newValue))).
		Commit()
	if err != nil {
		return false, fmt.Errorf("state: etcd cas %q: %w", key, err)
	}
	return resp.Succeeded, nil
}

// Close releases the etcd client.
func (s *EtcdStore) Close() error {
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("state: etcd close: %w", err)
	}
	return nil
}
