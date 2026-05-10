package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	// IdempotencyKeyVersionV1 is the initial stable spec for SDK-level idempotency keys.
	IdempotencyKeyVersionV1 = "v1"

	maxIdempotencyPartLength = 128
)

var (
	errIdempotencyTenantRequired        = errors.New("execution idempotency: tenant is required")
	errIdempotencyStrategyRequired      = errors.New("execution idempotency: strategy is required")
	errIdempotencyClientOrderIDRequired = errors.New("execution idempotency: client_order_id is required")
	errIdempotencyPartTooLong           = errors.New("execution idempotency: field exceeds max length")
	errIdempotencyCanonicalInvalid      = errors.New("execution idempotency: invalid canonical format")
)

// IdempotencyKeyInput defines the minimal fields for key construction.
type IdempotencyKeyInput struct {
	Tenant        string
	Strategy      string
	ClientOrderID string
}

// IdempotencyKey represents canonical + hashed key outputs.
type IdempotencyKey struct {
	Version   string `json:"version"`
	Canonical string `json:"canonical"`
	Digest    string `json:"digest"`
	Value     string `json:"value"`
}

// BuildIdempotencyKey creates a deterministic key from tenant + strategy + client_order_id.
//
// Canonical format:
// tenant=<tenant>;strategy=<strategy>;client_order_id=<clientOrderID>
//
// Final value format:
// idem:<version>:<sha256(canonical)>
func BuildIdempotencyKey(input IdempotencyKeyInput) (IdempotencyKey, error) {
	tenant := normalizeScopePart(input.Tenant)
	if tenant == "" {
		return IdempotencyKey{}, errIdempotencyTenantRequired
	}
	strategy := normalizeScopePart(input.Strategy)
	if strategy == "" {
		return IdempotencyKey{}, errIdempotencyStrategyRequired
	}
	clientOrderID := strings.TrimSpace(input.ClientOrderID)
	if clientOrderID == "" {
		return IdempotencyKey{}, errIdempotencyClientOrderIDRequired
	}

	if len(tenant) > maxIdempotencyPartLength || len(strategy) > maxIdempotencyPartLength || len(clientOrderID) > maxIdempotencyPartLength {
		return IdempotencyKey{}, errIdempotencyPartTooLong
	}

	canonical := fmt.Sprintf("tenant=%s;strategy=%s;client_order_id=%s", tenant, strategy, clientOrderID)
	sum := sha256.Sum256([]byte(canonical))
	digest := hex.EncodeToString(sum[:])

	return IdempotencyKey{
		Version:   IdempotencyKeyVersionV1,
		Canonical: canonical,
		Digest:    digest,
		Value:     fmt.Sprintf("idem:%s:%s", IdempotencyKeyVersionV1, digest),
	}, nil
}

// IdempotencyParts are parsed from canonical representation.
type IdempotencyParts struct {
	Tenant        string
	Strategy      string
	ClientOrderID string
}

// ParseIdempotencyCanonical parses canonical form into typed parts.
func ParseIdempotencyCanonical(canonical string) (IdempotencyParts, error) {
	s := strings.TrimSpace(canonical)
	if s == "" {
		return IdempotencyParts{}, errIdempotencyCanonicalInvalid
	}

	parts := strings.Split(s, ";")
	if len(parts) != 3 {
		return IdempotencyParts{}, errIdempotencyCanonicalInvalid
	}

	lookup := make(map[string]string, 3)
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			return IdempotencyParts{}, errIdempotencyCanonicalInvalid
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" || v == "" {
			return IdempotencyParts{}, errIdempotencyCanonicalInvalid
		}
		lookup[k] = v
	}

	out := IdempotencyParts{
		Tenant:        lookup["tenant"],
		Strategy:      lookup["strategy"],
		ClientOrderID: lookup["client_order_id"],
	}
	if out.Tenant == "" || out.Strategy == "" || out.ClientOrderID == "" {
		return IdempotencyParts{}, errIdempotencyCanonicalInvalid
	}
	return out, nil
}

func normalizeScopePart(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
