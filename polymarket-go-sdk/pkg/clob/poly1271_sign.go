package clob

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Mirrors @polymarket/clob-client-v2 exchangeOrderBuilderV2.ts (POLY_1271 path).
const (
	ctfExchangeV2DomainName    = "Polymarket CTF Exchange"
	ctfExchangeV2DomainVer     = "2"
	orderV2TypeString          = "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"
	eip712DomainTypeString     = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
	depositWalletTypedDataName = "DepositWallet"
	depositWalletTypedDataVer  = "1"
)

var (
	eip712DomainSeparatorTypeHash = crypto.Keccak256Hash([]byte(eip712DomainTypeString))
	ctfExchangeV2NameHash         = crypto.Keccak256Hash([]byte(ctfExchangeV2DomainName))
	ctfExchangeV2VersionHash      = crypto.Keccak256Hash([]byte(ctfExchangeV2DomainVer))
	orderV2StructTypeHash         = crypto.Keccak256Hash([]byte(orderV2TypeString))
)

func mustABIType(t string) abi.Type {
	ty, err := abi.NewType(t, "", nil)
	if err != nil {
		panic("clob poly1271 abi type: " + err.Error())
	}
	return ty
}

var (
	packArgsAppDomain = abi.Arguments{
		{Type: mustABIType("bytes32")},
		{Type: mustABIType("bytes32")},
		{Type: mustABIType("bytes32")},
		{Type: mustABIType("uint256")},
		{Type: mustABIType("address")},
	}
	packArgsOrderContents = abi.Arguments{
		{Type: mustABIType("bytes32")},
		{Type: mustABIType("uint256")},
		{Type: mustABIType("address")},
		{Type: mustABIType("address")},
		{Type: mustABIType("uint256")},
		{Type: mustABIType("uint256")},
		{Type: mustABIType("uint256")},
		{Type: mustABIType("uint8")},
		{Type: mustABIType("uint8")},
		{Type: mustABIType("uint256")},
		{Type: mustABIType("bytes32")},
		{Type: mustABIType("bytes32")},
	}
)

func exchangeV2AppDomainSeparator(chainID *big.Int, verifyingContract common.Address) (common.Hash, error) {
	packed, err := packArgsAppDomain.Pack(
		eip712DomainSeparatorTypeHash,
		ctfExchangeV2NameHash,
		ctfExchangeV2VersionHash,
		chainID,
		verifyingContract,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("pack app domain sep: %w", err)
	}
	return crypto.Keccak256Hash(packed), nil
}

func orderMetadataBuilderHashes(order *clobtypes.Order) (meta, builder common.Hash, err error) {
	mb, err := hexutil.Decode(padBytes32(order.Metadata))
	if err != nil || len(mb) != 32 {
		return common.Hash{}, common.Hash{}, fmt.Errorf("metadata bytes32: %w", err)
	}
	bb, err := hexutil.Decode(padBytes32(order.Builder))
	if err != nil || len(bb) != 32 {
		return common.Hash{}, common.Hash{}, fmt.Errorf("builder bytes32: %w", err)
	}
	meta.SetBytes(mb)
	builder.SetBytes(bb)
	return meta, builder, nil
}

func exchangeV2OrderContentsHash(order *clobtypes.Order, sideInt, sigTypeVal int, timestamp int64) (common.Hash, error) {
	meta, bld, err := orderMetadataBuilderHashes(order)
	if err != nil {
		return common.Hash{}, err
	}
	packed, err := packArgsOrderContents.Pack(
		orderV2StructTypeHash,
		order.Salt.Int,
		common.Address(order.Maker),
		common.Address(order.Signer),
		order.TokenID.Int,
		order.MakerAmount.BigInt(),
		order.TakerAmount.BigInt(),
		uint8(sideInt),
		uint8(sigTypeVal),
		big.NewInt(timestamp),
		meta,
		bld,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("pack order contents: %w", err)
	}
	return crypto.Keccak256Hash(packed), nil
}

func typedDataSignTypes(orderTypes []apitypes.Type) apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Order": orderTypes,
		"TypedDataSign": {
			{Name: "contents", Type: "Order"},
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
			{Name: "salt", Type: "bytes32"},
		},
	}
}

// signExchangeV2Poly1271 returns the compound 0x signature expected by the CLOB for signature_type=3.
func signExchangeV2Poly1271(
	signer auth.Signer,
	domain *apitypes.TypedDataDomain,
	orderTypes []apitypes.Type,
	order *clobtypes.Order,
	orderMessage apitypes.TypedDataMessage,
	sideInt, sigTypeVal int,
	timestamp int64,
	exchangeAddr common.Address,
) ([]byte, error) {
	chainID := signer.ChainID()
	if chainID == nil {
		chainID = big.NewInt(0)
	}

	appDomainSep, err := exchangeV2AppDomainSeparator(chainID, exchangeAddr)
	if err != nil {
		return nil, err
	}

	contentsHash, err := exchangeV2OrderContentsHash(order, sideInt, sigTypeVal, timestamp)
	if err != nil {
		return nil, err
	}

	zeroSalt, err := hexutil.Decode(padBytes32(""))
	if err != nil || len(zeroSalt) != 32 {
		return nil, fmt.Errorf("zero salt bytes32")
	}

	wrapMsg := apitypes.TypedDataMessage{
		"contents":          orderMessage,
		"name":              depositWalletTypedDataName,
		"version":           depositWalletTypedDataVer,
		"chainId":           (*math.HexOrDecimal256)(new(big.Int).Set(chainID)),
		"verifyingContract": common.Address(order.Signer).Hex(),
		"salt":              hexutil.Encode(zeroSalt),
	}

	innerTypes := typedDataSignTypes(orderTypes)
	innerSig, err := signer.SignTypedData(domain, innerTypes, wrapMsg, "TypedDataSign")
	if err != nil {
		return nil, fmt.Errorf("poly1271 inner typed data sign: %w", err)
	}

	typeUTF8 := []byte(orderV2TypeString)
	typeHex := hex.EncodeToString(typeUTF8)
	lenHex := fmt.Sprintf("%04x", len(typeUTF8))

	innerHex := hexutil.Encode(innerSig)[2:]
	fullHex := "0x" + innerHex + hex.EncodeToString(appDomainSep.Bytes()) + hex.EncodeToString(contentsHash.Bytes()) + typeHex + lenHex
	out, err := hexutil.Decode(fullHex)
	if err != nil {
		return nil, fmt.Errorf("decode compound poly1271 sig: %w", err)
	}
	return out, nil
}
