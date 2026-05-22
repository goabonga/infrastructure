// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package idp

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer signs ES256 JWTs for authenticated subjects.
type Issuer struct {
	key    *ecdsa.PrivateKey
	issuer string
	ttl    time.Duration
}

// NewIssuer returns an Issuer signing with key, stamping the issuer claim and a
// ttl-bounded expiry.
func NewIssuer(key *ecdsa.PrivateKey, issuer string, ttl time.Duration) *Issuer {
	return &Issuer{key: key, issuer: issuer, ttl: ttl}
}

// TTL is the lifetime of issued tokens.
func (i *Issuer) TTL() time.Duration {
	return i.ttl
}

// Issue signs a token for subject.
func (i *Issuer) Issue(subject string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    i.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(i.key)
	if err != nil {
		return "", fmt.Errorf("idp: sign token: %w", err)
	}
	return token, nil
}
