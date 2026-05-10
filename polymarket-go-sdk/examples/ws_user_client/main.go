// User WebSocket demo: only POLYMARKET_PK is required. L2 API credentials are obtained
// via L1 (CreateOrDeriveAPIKey).
//
// By default this subscribes to user order/trade streams for all markets so events match
// whatever you trade on polymarket.com. Set POLYMARKET_MARKET (one condition id) or
// POLYMARKET_MARKETS (comma-separated) to filter.
//
// After L2 auth: prints CLOB USDC collateral. If Gamma reports a proxyWallet, the example
// sends signature_type=3 (POLY_1271) and funder=<proxy> on /balance-allowance to match many polymarket.com setups.
// Override with POLYMARKET_SIGNATURE_TYPE (0=EOA, 1=Proxy, 2=Gnosis Safe, 3=POLY1271).
// Optional POLYMARKET_TOKEN_ID logs conditional (outcome share) balance for that token.
//
// Wallet lines match Polymarket.com: proxy from Gamma public-profile, deposit from Bridge API.
// Set POLYMARKET_SHOW_DERIVED_WALLETS=1 to also print local CREATE2 hints (often ≠ site for MetaMask/Safe users).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	polymarket "github.com/GoPolymarket/polymarket-go-sdk"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/bridge"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	ctx := context.Background()

	pkHex := strings.TrimSpace(os.Getenv("POLYMARKET_PK"))
	if pkHex == "" {
		log.Fatal("POLYMARKET_PK is required (hex private key, 0x prefix optional)")
	}

	chainID := int64(137)
	if raw := strings.TrimSpace(os.Getenv("POLYMARKET_CHAIN_ID")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			chainID = v
		}
	}

	signer, err := auth.NewPrivateKeySigner(pkHex, chainID)
	if err != nil {
		log.Fatalf("signer: %v", err)
	}

	opts := []polymarket.Option{polymarket.WithUseServerTime(true)}
	if p := strings.TrimSpace(os.Getenv("POLYMARKET_PROXY_URL")); p != "" {
		opts = append(opts, polymarket.WithProxyURL(p))
		log.Printf("Using HTTP/WebSocket proxy: %s", p)
	}
	client := polymarket.NewClient(opts...)

	gammaProxy := logPolymarketWalletSummary(ctx, client, signer, chainID)

	apiKey, err := resolveL2Credentials(ctx, client, signer)
	if err != nil {
		log.Fatalf("L2 API credentials: %v", err)
	}
	log.Printf("L2 API key ready (key id prefix: %.12s…)", apiKey.Key)

	logClobAccountBalances(ctx, client, signer, apiKey, gammaProxy)

	userMarkets := parseUserMarketsFromEnv()
	if len(userMarkets) == 0 {
		log.Printf("Subscribing user streams for all markets")
	} else {
		log.Printf("Subscribing user streams for %d market(s): %s", len(userMarkets), strings.Join(userMarkets, ", "))
	}

	wsBase := strings.TrimSpace(os.Getenv("POLYMARKET_CLOB_WS_URL"))
	if wsBase == "" {
		wsBase = client.Config.BaseURLs.CLOBWS
	}
	if wsBase == "" {
		wsBase = ws.ProdBaseURL
	}

	wsClient, err := ws.NewClientWithConfig(wsBase, signer, apiKey, client.Config.CLOBWSConfig)
	if err != nil {
		log.Fatalf("WebSocket connect: %v", err)
	}
	defer wsClient.Close()

	orderCh, err := wsClient.SubscribeUserOrders(ctx, userMarkets)
	if err != nil {
		log.Fatalf("SubscribeUserOrders: %v", err)
	}
	tradeCh, err := wsClient.SubscribeUserTrades(ctx, userMarkets)
	if err != nil {
		log.Fatalf("SubscribeUserTrades: %v", err)
	}

	log.Println("Listening — place or cancel an order on polymarket.com, then watch events (Ctrl+C to exit).")

	go func() {
		for event := range orderCh {
			fmt.Printf("[order] %+v\n", event)
		}
	}()
	go func() {
		for event := range tradeCh {
			fmt.Printf("[trade] %+v\n", event)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	fmt.Println("Shutting down...")
}

// logPolymarketWalletSummary prints the same conceptual addresses as the website:
// EOA, proxy (positions / CLOB funder from Gamma), bridge deposit EVM (funding rail).
// It returns the Gamma proxyWallet when present (zero address otherwise) for use by CLOB balance queries.
func logPolymarketWalletSummary(ctx context.Context, root *polymarket.Client, signer auth.Signer, chainID int64) common.Address {
	eoa := signer.Address()
	log.Printf("EOA (signer): %s (chainId=%d)", eoa.Hex(), chainID)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var proxyAddr common.Address
	if root.Gamma != nil {
		prof, err := root.Gamma.PublicProfile(ctx, &gamma.PublicProfileRequest{Address: eoa.Hex()})
		if err != nil {
			log.Printf("Proxy wallet (Gamma public-profile): lookup failed: %v", err)
		} else if w := strings.TrimSpace(prof.ProxyWallet); w != "" && common.IsHexAddress(w) {
			proxyAddr = common.HexToAddress(w)
			log.Printf("Proxy wallet (trading / positions — same as Polymarket.com profile): %s", proxyAddr.Hex())
		} else {
			log.Printf("Proxy wallet: Gamma returned no proxyWallet (new account or never used on site). Trade once on polymarket.com or check address format.")
		}
	} else {
		log.Printf("Proxy wallet: Gamma client unavailable")
	}

	if root.Bridge != nil {
		logBridgeDepositEVM(ctx, root.Bridge, "EOA", eoa.Hex())
		if proxyAddr != (common.Address{}) && !strings.EqualFold(proxyAddr.Hex(), eoa.Hex()) {
			logBridgeDepositEVM(ctx, root.Bridge, "proxy wallet (Gamma)", proxyAddr.Hex())
		}
		log.Printf("Bridge deposit addresses are returned by the API per queried wallet — not computable from your private key on-chain. Match the row to what Polymarket.com shows for your login (often proxy-scoped for email/Magic).")
	} else {
		log.Printf("Bridge deposit: client unavailable")
	}

	if strings.TrimSpace(os.Getenv("POLYMARKET_SHOW_DERIVED_WALLETS")) == "1" {
		magic, errM := auth.DeriveProxyWalletForChain(eoa, chainID)
		safe, errS := auth.DeriveSafeWalletForChain(eoa, chainID)
		if errM == nil && errS == nil {
			log.Printf("DEBUG CREATE2 only (may not match site): Magic-factory proxy=%s | Gnosis-Safe=%s", magic.Hex(), safe.Hex())
		}
	}
	return proxyAddr
}

func logClobAccountBalances(ctx context.Context, root *polymarket.Client, signer auth.Signer, apiKey *auth.APIKey, gammaProxy common.Address) {
	subCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	explicitSig := parseSignatureTypeFromEnv()
	sig := balanceSignatureType(explicitSig, gammaProxy)
	funderHex := ""
	if gammaProxy != (common.Address{}) {
		funderHex = gammaProxy.Hex()
	}
	resp, err := root.ClobCollateralBalance(subCtx, signer, apiKey, sig, funderHex)
	if err != nil {
		log.Printf("CLOB USDC collateral: %v", err)
		return
	}
	desc := balanceSigDescription(explicitSig, gammaProxy)
	log.Printf("CLOB collateral (USDC) [%s]: balance=%s", desc, resp.Balance)
	if len(resp.Allowances) > 0 {
		log.Printf("CLOB allowances (spender -> allowance): %+v", resp.Allowances)
	}

	if tokenID := strings.TrimSpace(os.Getenv("POLYMARKET_TOKEN_ID")); tokenID != "" {
		cond, err2 := root.ClobConditionalBalance(subCtx, signer, apiKey, tokenID, sig, funderHex)
		if err2 != nil {
			log.Printf("CLOB conditional balance (token_id=%s): %v", tokenID, err2)
		} else {
			log.Printf("CLOB conditional (token_id=%s): balance=%s", tokenID, cond.Balance)
		}
	}
}

func parseSignatureTypeFromEnv() *auth.SignatureType {
	raw := strings.TrimSpace(os.Getenv("POLYMARKET_SIGNATURE_TYPE"))
	if raw == "" {
		return nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 || v > 3 {
		log.Printf("POLYMARKET_SIGNATURE_TYPE=%q ignored (use 0=EOA, 1=Proxy, 2=Gnosis Safe, 3=POLY1271)", raw)
		return nil
	}
	st := auth.SignatureType(v)
	return &st
}

// balanceSignatureType picks CLOB signature_type for balance: explicit env wins; otherwise
// use POLY1271 (3) when Gamma returned a proxyWallet (common for site trading + funder query), else EOA default.
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

func balanceSigDescription(explicit *auth.SignatureType, gammaProxy common.Address) string {
	if explicit != nil {
		return fmt.Sprintf("signature_type=%d (from POLYMARKET_SIGNATURE_TYPE)", int(*explicit))
	}
	if gammaProxy != (common.Address{}) {
		return fmt.Sprintf("signature_type=3 (POLY1271) + funder=%s (Gamma proxyWallet)", gammaProxy.Hex())
	}
	return "signature_type=0 (EOA default; no Gamma proxyWallet — set POLYMARKET_SIGNATURE_TYPE if needed)"
}

func logBridgeDepositEVM(ctx context.Context, bc bridge.Client, scopeLabel, queryAddress string) {
	resp, err := bc.DepositAddress(ctx, &bridge.DepositRequest{Address: queryAddress})
	if err != nil {
		log.Printf("Bridge deposit (query as %s): failed: %v", scopeLabel, err)
		return
	}
	evm := strings.TrimSpace(resp.Address.EVM)
	if evm == "" || !common.IsHexAddress(evm) {
		log.Printf("Bridge deposit (query as %s): no EVM address in response", scopeLabel)
		return
	}
	log.Printf("Bridge EVM deposit address (queried as %s): %s", scopeLabel, common.HexToAddress(evm).Hex())
}

// resolveL2Credentials prefers explicit env L2 triple if all set; otherwise CreateOrDeriveAPIKey (L1 signer).
// parseUserMarketsFromEnv returns nil for "subscribe all markets". POLYMARKET_MARKETS
// (comma-separated condition ids) takes precedence over a single POLYMARKET_MARKET.
func parseUserMarketsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("POLYMARKET_MARKETS"))
	if raw != "" {
		var out []string
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if single := strings.TrimSpace(os.Getenv("POLYMARKET_MARKET")); single != "" {
		return []string{single}
	}
	return nil
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

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	resp, err := clob.CreateOrDeriveAPIKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("CreateOrDeriveAPIKey: %w", err)
	}
	return &auth.APIKey{
		Key:        resp.APIKey,
		Secret:     resp.Secret,
		Passphrase: resp.Passphrase,
	}, nil
}

