package auth

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Polymarket CLOB V2 EIP-1271 deposit wallet (CREATE2). Matches bot `derivePolymarketDepositWalletAddress`
// and `polymarket-clob-v2-go-exmaple` — NOT the Magic.link proxy from DeriveProxyWalletForChain.
var (
	polymarketDepositWalletFactory        = common.HexToAddress("0x00000000000Fb5C9ADea0298D729A0CB3823Cc07")
	polymarketDepositWalletImplementation = common.HexToAddress("0x58CA52ebe0DadfdF531Cde7062e76746de4Db1eB")
	initConst1 = common.FromHex("0xcc3735a920a3ca505d382bbc545af43d6000803e6038573d6000fd5b3d6000f3")
	initConst2 = common.FromHex("0x5155f3363d3d373d3d363d7f360894a13ba1a3210667c828492db98dca3e2076")
)

func leftPadBigIntTo10Bytes(bi *big.Int) [10]byte {
	h := strings.TrimPrefix(bi.Text(16), "")
	if len(h)%2 == 1 {
		h = "0" + h
	}
	raw, err := hex.DecodeString(h)
	if err != nil {
		return [10]byte{}
	}
	var out [10]byte
	copyLen := len(raw)
	if copyLen > 10 {
		raw = raw[len(raw)-10:]
		copyLen = 10
	}
	copy(out[10-copyLen:], raw)
	return out
}

func initCodeHashERC1967(implementation common.Address, args []byte) common.Hash {
	prefix, _ := new(big.Int).SetString("61003D3D8160233D3973", 16)
	ln := big.NewInt(int64(len(args)))
	shifted := new(big.Int).Lsh(ln, 56)
	combined := new(big.Int).Add(prefix, shifted)
	head := leftPadBigIntTo10Bytes(combined)

	var buf []byte
	buf = append(buf, head[:]...)
	buf = append(buf, implementation.Bytes()...)
	buf = append(buf, 0x60, 0x09)
	buf = append(buf, initConst2...)
	buf = append(buf, initConst1...)
	buf = append(buf, args...)
	return crypto.Keccak256Hash(buf)
}

// DerivePolymarketDepositWallet returns the CREATE2 Polymarket deposit wallet for an EOA (POLY_1271 funder / order.signer).
func DerivePolymarketDepositWallet(owner common.Address) (common.Address, error) {
	owner = common.BytesToAddress(owner.Bytes())
	walletID := common.LeftPadBytes(owner.Bytes(), 32)

	tAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return common.Address{}, err
	}
	tB32, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return common.Address{}, err
	}
	args, err := abi.Arguments{{Type: tAddr}, {Type: tB32}}.Pack(polymarketDepositWalletFactory, common.BytesToHash(walletID))
	if err != nil {
		return common.Address{}, err
	}

	salt := crypto.Keccak256Hash(args)
	bytecodeHash := initCodeHashERC1967(polymarketDepositWalletImplementation, args)

	var create2Input []byte
	create2Input = append(create2Input, 0xff)
	create2Input = append(create2Input, polymarketDepositWalletFactory.Bytes()...)
	create2Input = append(create2Input, salt.Bytes()...)
	create2Input = append(create2Input, bytecodeHash.Bytes()...)

	h := crypto.Keccak256(create2Input)
	return common.BytesToAddress(h[12:32]), nil
}
