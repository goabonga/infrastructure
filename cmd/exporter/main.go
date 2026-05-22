// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Command infra-exporter exposes the declarative state as Prometheus metrics.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/meta"
	"github.com/goabonga/infrastructure/internal/metrics"
	"github.com/goabonga/infrastructure/internal/state"
)

func main() {
	addr := flag.String("addr", envOr("GOA_EXPORTER_ADDR", ":9100"), "listen address")
	stateDir := flag.String("state-dir", envOr("GOA_STATE_DIR", "./state"), "state directory")
	stateDSN := flag.String("state-dsn", envOr("GOA_STATE_DSN", ""), "PostgreSQL DSN (enables the HA backend)")
	flag.Parse()

	store, err := state.Open(*stateDir, *stateDSN)
	if err != nil {
		log.Fatalf("infra-exporter: open state: %v", err)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(metrics.NewCollector(store,
		resource.KindVPC,
		resource.KindACLPolicy,
		resource.KindSecret,
		resource.KindSSLCA,
	))

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("%s listening on %s (state dir %s)", meta.Line("infra-exporter", Version), *addr, *stateDir)
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("infra-exporter: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
