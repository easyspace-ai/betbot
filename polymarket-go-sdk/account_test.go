package polymarket

import (
	"context"
	"testing"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
)

func TestClient_ClobCollateralBalance_validation(t *testing.T) {
	ctx := context.Background()
	key := &auth.APIKey{Key: "k", Secret: "s", Passphrase: "p"}

	t.Run("nil client", func(t *testing.T) {
		var c *Client
		_, err := c.ClobCollateralBalance(ctx, nil, key, nil, "")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil clob", func(t *testing.T) {
		c := NewClient(WithConfig(invalidStreamingConfig()), WithCLOB(nil))
		_, err := c.ClobCollateralBalance(ctx, nil, key, nil, "")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil signer", func(t *testing.T) {
		c := NewClient(WithConfig(invalidStreamingConfig()))
		_, err := c.ClobCollateralBalance(ctx, nil, key, nil, "")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil api key", func(t *testing.T) {
		c := NewClient(WithConfig(invalidStreamingConfig()))
		s, _ := auth.NewPrivateKeySigner("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318", 137)
		_, err := c.ClobCollateralBalance(ctx, s, nil, nil, "")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestClient_ClobCollateralBalance_invalidFunder(t *testing.T) {
	ctx := context.Background()
	c := NewClient(WithConfig(invalidStreamingConfig()))
	s, _ := auth.NewPrivateKeySigner("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318", 137)
	key := &auth.APIKey{Key: "k", Secret: "s", Passphrase: "p"}
	_, err := c.ClobCollateralBalance(ctx, s, key, nil, "not-an-address")
	if err == nil {
		t.Fatal("expected error for invalid funder")
	}
}

func TestClient_ClobConditionalBalance_validation(t *testing.T) {
	ctx := context.Background()
	s, _ := auth.NewPrivateKeySigner("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318", 137)
	key := &auth.APIKey{Key: "k", Secret: "s", Passphrase: "p"}

	c := NewClient(WithConfig(invalidStreamingConfig()))
	_, err := c.ClobConditionalBalance(ctx, s, key, "", nil, "")
	if err == nil {
		t.Fatal("expected error for empty tokenID")
	}
}
