// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Command infra-api runs the declarative control-plane API server.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/httpsrv"
	"github.com/goabonga/infrastructure/internal/meta"
	"github.com/goabonga/infrastructure/internal/state"
)

func main() {
	addr := flag.String("addr", envOr("GOA_API_ADDR", ":8080"), "listen address")
	stateDir := flag.String("state-dir", envOr("GOA_STATE_DIR", "./state"), "state directory")
	flag.Parse()

	store := state.NewFileStore(*stateDir)

	var opts []httpsrv.Option
	if raw := os.Getenv("GOA_KMS_KEY"); raw != "" {
		kek, err := crypto.NewKEKFromBase64(raw)
		if err != nil {
			log.Fatalf("infra-api: GOA_KMS_KEY: %v", err)
		}
		opts = append(opts, httpsrv.WithSecretEncryption(kek))
		log.Print("infra-api: secret encryption enabled")
	} else {
		log.Print("infra-api: GOA_KMS_KEY unset; secret routes disabled")
	}
	srv := httpsrv.New(store, opts...)

	log.Printf("%s listening on %s (state dir %s)", meta.Line("infra-api", Version), *addr, *stateDir)
	if err := srv.ListenAndServe(*addr); err != nil {
		log.Fatalf("infra-api: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
