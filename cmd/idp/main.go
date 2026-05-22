// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Command infra-idp is the identity provider: it issues ES256 JWTs to clients
// via the OAuth2 client-credentials grant and publishes its public key.
package main

import (
	"crypto/ecdsa"
	"flag"
	"log"
	"os"
	"time"

	"github.com/goabonga/infrastructure/internal/auth"
	"github.com/goabonga/infrastructure/internal/idp"
	"github.com/goabonga/infrastructure/internal/meta"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("infra-idp: %v", err)
	}
}

func run() error {
	addr := flag.String("addr", envOr("GOA_IDP_ADDR", ":8081"), "listen address")
	issuerURL := flag.String("issuer", envOr("GOA_IDP_ISSUER", "http://localhost:8081"), "issuer URL")
	ttl := flag.Duration("ttl", time.Hour, "token lifetime")
	flag.Parse()

	key, err := loadOrGenerateKey()
	if err != nil {
		return err
	}

	clients, err := auth.ParseTokens(os.Getenv("GOA_IDP_CLIENTS"))
	if err != nil {
		return err
	}

	server := idp.NewServer(idp.NewIssuer(key, *issuerURL, *ttl), clients, &key.PublicKey, *issuerURL)

	log.Printf("%s listening on %s (issuer %s)", meta.Line("infra-idp", Version), *addr, *issuerURL)
	pub, err := idp.MarshalPublicKeyPEM(&key.PublicKey)
	if err != nil {
		return err
	}
	log.Printf("infra-idp: verification public key:\n%s", pub)

	return server.ListenAndServe(*addr)
}

// loadOrGenerateKey reads the signing key from GOA_IDP_KEY (PEM) or generates an
// ephemeral one, warning that tokens will not survive a restart.
func loadOrGenerateKey() (*ecdsa.PrivateKey, error) {
	if raw := os.Getenv("GOA_IDP_KEY"); raw != "" {
		return idp.ParsePrivateKeyPEM([]byte(raw))
	}
	log.Print("infra-idp: GOA_IDP_KEY unset; generating an ephemeral signing key")
	return idp.GenerateKey()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
