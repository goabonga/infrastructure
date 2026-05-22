// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Package httpsrv wires the resource handlers onto an HTTP mux and runs the
// API server.
package httpsrv

import (
	"net/http"
	"time"

	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/handler"
	"github.com/goabonga/infrastructure/internal/registry"
	"github.com/goabonga/infrastructure/internal/secret"
	"github.com/goabonga/infrastructure/internal/state"
)

// APIBase is the path prefix for all resource routes.
const APIBase = "/api/v1"

// Server holds the routed HTTP handler for the control-plane API.
type Server struct {
	mux   *http.ServeMux
	store state.Store
	kek   *crypto.KEK
}

// Option configures a Server.
type Option func(*Server)

// WithSecretEncryption enables the secret resource, encrypting at rest with kek.
// Without it the API serves no secret routes.
func WithSecretEncryption(kek *crypto.KEK) Option {
	return func(s *Server) { s.kek = kek }
}

// New builds a Server backed by store with every resource handler registered.
func New(store state.Store, opts ...Option) *Server {
	s := &Server{mux: http.NewServeMux(), store: store}
	for _, o := range opts {
		o(s)
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	vpcs := registry.New[resource.VPCSpec, resource.VPCStatus](s.store, resource.KindVPC)
	handler.New(vpcs, resource.KindVPC).Register(s.mux, APIBase)

	if s.kek != nil {
		secrets := registry.New[resource.SecretSpec, resource.SecretStatus](s.store, resource.KindSecret)
		handler.NewSecretHandler(secret.NewService(secrets, s.kek)).Register(s.mux, APIBase)
	}

	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

// Handler returns the routed HTTP handler, useful for tests.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe runs the API server on addr until it errors.
func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}
