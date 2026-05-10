package polymarket

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/ethereum/go-ethereum/common"
)

// ClobCollateralBalance returns CLOB collateral balance (USDC available for trading) using L2-authenticated REST.
//
// funderHex should be your Polymarket proxy wallet (Gamma proxyWallet) when funds sit there; it is sent as the
// CLOB "funder" query parameter together with signatureType (many site accounts use SignaturePoly1271 (3)).
//
// signatureType nil uses the CLOB default (EOA). Match the type your Polymarket account uses (0–3); the ws_user_client
// example defaults to 3 + funder when Gamma returns a proxyWallet.
func (c *Client) ClobCollateralBalance(ctx context.Context, signer auth.Signer, apiKey *auth.APIKey, signatureType *auth.SignatureType, funderHex string) (clobtypes.BalanceAllowanceResponse, error) {
	if c == nil {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("polymarket client is nil")
	}
	if c.CLOB == nil {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("clob client is not configured")
	}
	if signer == nil || apiKey == nil {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("signer and API key are required")
	}
	cl := c.CLOB.WithAuth(signer, apiKey)
	if signatureType != nil {
		cl = cl.WithSignatureType(*signatureType)
	}
	req := &clobtypes.BalanceAllowanceRequest{
		AssetType: clobtypes.AssetTypeCollateral,
	}
	if fh := strings.TrimSpace(funderHex); fh != "" {
		if !common.IsHexAddress(fh) {
			return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("invalid funder address: %s", fh)
		}
		req.Funder = common.HexToAddress(fh).Hex()
	}
	return cl.BalanceAllowance(ctx, req)
}

// ClobConditionalBalance returns the CLOB balance for a conditional (outcome) token.
// Pass funderHex when collateral/outcome balances are held under your Polymarket proxy (same as ClobCollateralBalance).
func (c *Client) ClobConditionalBalance(ctx context.Context, signer auth.Signer, apiKey *auth.APIKey, tokenID string, signatureType *auth.SignatureType, funderHex string) (clobtypes.BalanceAllowanceResponse, error) {
	if c == nil {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("polymarket client is nil")
	}
	if c.CLOB == nil {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("clob client is not configured")
	}
	if signer == nil || apiKey == nil {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("signer and API key are required")
	}
	tid := strings.TrimSpace(tokenID)
	if tid == "" {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("tokenID is required")
	}
	cl := c.CLOB.WithAuth(signer, apiKey)
	if signatureType != nil {
		cl = cl.WithSignatureType(*signatureType)
	}
	req := &clobtypes.BalanceAllowanceRequest{
		AssetType: clobtypes.AssetTypeConditional,
		TokenID:   tid,
	}
	if fh := strings.TrimSpace(funderHex); fh != "" {
		if !common.IsHexAddress(fh) {
			return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("invalid funder address: %s", fh)
		}
		req.Funder = common.HexToAddress(fh).Hex()
	}
	return cl.BalanceAllowance(ctx, req)
}
