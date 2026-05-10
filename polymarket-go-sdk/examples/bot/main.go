package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	polymarket "github.com/GoPolymarket/polymarket-go-sdk"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	clobtypes "github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"

	"github.com/ethereum/go-ethereum/common"
)

// maxDemoMarkets caps public order-book fetches so a large /markets page does not burn the whole timeout.
const maxDemoMarkets = 10

func main() {
	fmt.Println("=== Polymarket Trading Bot Example ===")

	ctx := context.Background()

	pk := os.Getenv("POLYMARKET_PK")
	if pk == "" {
		fmt.Println("SKIPPED: POLYMARKET_PK not set")
		return
	}

	chainID := int64(137)
	if raw := strings.TrimSpace(os.Getenv("POLYMARKET_CHAIN_ID")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			chainID = v
		}
	}

	signer, err := auth.NewPrivateKeySigner(pk, chainID)
	if err != nil {
		log.Fatalf("Failed to create signer: %v", err)
	}
	fmt.Printf("Signer: %s\n", signer.Address().Hex())

	opts := []polymarket.Option{polymarket.WithUseServerTime(true)}
	if p := strings.TrimSpace(os.Getenv("POLYMARKET_PROXY_URL")); p != "" {
		opts = append(opts, polymarket.WithProxyURL(p))
		log.Printf("Using HTTP proxy: %s", p)
	}
	root := polymarket.NewClient(opts...)

	gammaProxy := gammaProxyWallet(ctx, root, signer)
	apiKey, err := resolveL2Credentials(ctx, root, signer)
	if err != nil {
		log.Fatalf("L2 API credentials: %v", err)
	}
	fmt.Printf("L2 API key ready (prefix: %.12s…)\n", apiKey.Key)

	runBot(ctx, root, signer, apiKey, gammaProxy)
}

func gammaProxyWallet(ctx context.Context, root *polymarket.Client, signer auth.Signer) common.Address {
	sub, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if root.Gamma == nil {
		log.Printf("Gamma: client unavailable (no proxyWallet for funder)")
		return common.Address{}
	}
	prof, err := root.Gamma.PublicProfile(sub, &gamma.PublicProfileRequest{Address: signer.Address().Hex()})
	if err != nil {
		log.Printf("Gamma public-profile: %v", err)
		return common.Address{}
	}
	w := strings.TrimSpace(prof.ProxyWallet)
	if w == "" || !common.IsHexAddress(w) {
		log.Printf("Gamma: no proxyWallet (EOA-only or new account)")
		return common.Address{}
	}
	return common.HexToAddress(w)
}

func resolveL2Credentials(ctx context.Context, root *polymarket.Client, signer auth.Signer) (*auth.APIKey, error) {
	key := strings.TrimSpace(os.Getenv("POLYMARKET_API_KEY"))
	secret := strings.TrimSpace(os.Getenv("POLYMARKET_API_SECRET"))
	pass := strings.TrimSpace(os.Getenv("POLYMARKET_API_PASSPHRASE"))
	if key != "" && secret != "" && pass != "" {
		return &auth.APIKey{Key: key, Secret: secret, Passphrase: pass}, nil
	}

	clob := root.CLOB.WithAuth(signer, nil)
	if raw := strings.TrimSpace(os.Getenv("POLYMARKET_AUTH_NONCE")); raw != "" {
		if nonce, err := strconv.ParseInt(raw, 10, 64); err == nil {
			clob = clob.WithAuthNonce(nonce)
		}
	}

	sub, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	resp, err := clob.CreateOrDeriveAPIKey(sub)
	if err != nil {
		return nil, fmt.Errorf("CreateOrDeriveAPIKey: %w", err)
	}
	return &auth.APIKey{
		Key:        resp.APIKey,
		Secret:     resp.Secret,
		Passphrase: resp.Passphrase,
	}, nil
}

func parseSignatureTypeFromEnv() *auth.SignatureType {
	raw := strings.TrimSpace(os.Getenv("POLYMARKET_SIGNATURE_TYPE"))
	if raw == "" {
		return nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 || v > 3 {
		log.Printf("POLYMARKET_SIGNATURE_TYPE=%q ignored (use 0–3)", raw)
		return nil
	}
	st := auth.SignatureType(v)
	return &st
}

func balanceSignatureType(explicit *auth.SignatureType, gammaProxy common.Address) *auth.SignatureType {
	if explicit != nil {
		return explicit
	}
	if gammaProxy != (common.Address{}) {
		st := auth.SignaturePoly1271
		return &st
	}
	return nil
}

func runBot(ctx context.Context, root *polymarket.Client, signer auth.Signer, apiKey *auth.APIKey, gammaProxy common.Address) {
	publicClob := root.CLOB

	fmt.Println("\n--- Bot Scanning Markets ---")
	scanCtx, cancelScan := context.WithTimeout(ctx, 60*time.Second)
	defer cancelScan()

	markets, err := publicClob.Markets(scanCtx, &clobtypes.MarketsRequest{
		Active: ptrBool(true),
		Limit:  maxDemoMarkets,
	})
	if err != nil {
		log.Printf("Failed to get markets: %v", err)
		return
	}

	fmt.Printf("Found %d active markets (showing up to %d order books)\n", len(markets.Data), maxDemoMarkets)

	n := 0
	for _, market := range markets.Data {
		if n >= maxDemoMarkets {
			break
		}
		if len(market.Tokens) == 0 {
			continue
		}

		tokenID := market.Tokens[0].TokenID
		book, err := publicClob.OrderBook(scanCtx, &clobtypes.BookRequest{
			TokenID: tokenID,
		})
		if err != nil {
			continue
		}

		if len(book.Asks) == 0 || len(book.Bids) == 0 {
			continue
		}

		bestAsk := book.Asks[0].Price
		bestBid := book.Bids[0].Price
		fmt.Printf("%d. %s\n   Ask: %s, Bid: %s\n", n+1, market.Question, bestAsk, bestBid)
		n++
	}

	fmt.Println("\n--- Bot Checking Account ---")
	acctCtx, cancelAcct := context.WithTimeout(ctx, 45*time.Second)
	defer cancelAcct()

	explicitSig := parseSignatureTypeFromEnv()
	sig := balanceSignatureType(explicitSig, gammaProxy)
	funderHex := ""
	if gammaProxy != (common.Address{}) {
		funderHex = gammaProxy.Hex()
	}

	bal, err := root.ClobCollateralBalance(acctCtx, signer, apiKey, sig, funderHex)
	if err != nil {
		log.Printf("Failed to get collateral balance: %v", err)
		return
	}
	fmt.Printf("CLOB collateral (USDC) balance: %s\n", bal.Balance)

	authClob := root.CLOB.WithAuth(signer, apiKey)
	if sig != nil {
		authClob = authClob.WithSignatureType(*sig)
	}
	orders, err := authClob.Orders(acctCtx, &clobtypes.OrdersRequest{Limit: 5})
	if err != nil {
		log.Printf("Failed to get orders: %v", err)
		return
	}
	fmt.Printf("Open orders: %d\n", len(orders.Data))

	fmt.Println("\n--- Bot Complete ---")
}

func ptrBool(b bool) *bool {
	return &b
}
