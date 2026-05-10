package auth

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDerivePolymarketDepositWallet_Golden(t *testing.T) {
	owner := common.HexToAddress("0x3689aA2cEb25087B1806e13ba9f44fEA081Dad98")
	want := strings.ToLower("0x6579c2eef112f5187bfa322e40b24b6317a43bec")

	got, err := DerivePolymarketDepositWallet(owner)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ToLower(got.Hex()) != want {
		t.Fatalf("got %s want %s", got.Hex(), want)
	}
}
