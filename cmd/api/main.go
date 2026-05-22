// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Command infra-api runs the declarative control-plane API server.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/goabonga/infrastructure/internal/auth"
	"github.com/goabonga/infrastructure/internal/crypto"
	"github.com/goabonga/infrastructure/internal/httpsrv"
	"github.com/goabonga/infrastructure/internal/meta"
	"github.com/goabonga/infrastructure/internal/state"
)

func main() {
	addr := flag.String("addr", envOr("GOA_API_ADDR", ":8080"), "listen address")
	stateDir := flag.String("state-dir", envOr("GOA_STATE_DIR", "./state"), "state directory")
	stateDSN := flag.String("state-dsn", envOr("GOA_STATE_DSN", ""), "PostgreSQL DSN (enables the HA backend)")
	flag.Parse()

	store, err := state.Open(*stateDir, *stateDSN)
	if err != nil {
		log.Fatalf("infra-api: open state: %v", err)
	}

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

	if authn := buildAuth(); authn != nil {
		opts = append(opts, httpsrv.WithAuth(authn))
	}

	srv := httpsrv.New(store, opts...)

	log.Printf("%s listening on %s (state dir %s)", meta.Line("infra-api", Version), *addr, *stateDir)
	if err := srv.ListenAndServe(*addr); err != nil {
		log.Fatalf("infra-api: %v", err)
	}
}

// buildAuth selects the authenticator from the environment: a JWT verifier when
// GOA_API_JWT_PUBKEY is set, otherwise static tokens from GOA_API_TOKENS,
// otherwise none (the API is open).
func buildAuth() auth.Authenticator {
	if pubPEM := os.Getenv("GOA_API_JWT_PUBKEY"); pubPEM != "" {
		pub, err := auth.ParseECPublicKeyPEM([]byte(pubPEM))
		if err != nil {
			log.Fatalf("infra-api: GOA_API_JWT_PUBKEY: %v", err)
		}
		issuer := os.Getenv("GOA_API_JWT_ISSUER")
		if issuer == "" {
			log.Fatal("infra-api: GOA_API_JWT_ISSUER is required with GOA_API_JWT_PUBKEY")
		}
		log.Print("infra-api: JWT authentication enabled")
		return auth.NewJWTAuthenticator(pub, issuer)
	}
	if spec := os.Getenv("GOA_API_TOKENS"); spec != "" {
		tokens, err := auth.ParseTokens(spec)
		if err != nil {
			log.Fatalf("infra-api: GOA_API_TOKENS: %v", err)
		}
		log.Print("infra-api: token authentication enabled")
		return auth.NewTokenAuthenticator(tokens)
	}
	log.Print("infra-api: no auth configured; API is unauthenticated")
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
