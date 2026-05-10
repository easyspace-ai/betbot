package kms

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Default timeout for KMS operations
const defaultKMSTimeout = 10 * time.Second

// AWSSigner implements auth.Signer using AWS KMS.
type AWSSigner struct {
	client  *kms.Client
	keyID   string
	chainID *big.Int
	pubKey  *ecdsa.PublicKey
	address common.Address
	timeout time.Duration // Timeout for KMS operations
}

// NewAWSSigner creates a new signer backed by an AWS KMS key.
// It fetches the public key from KMS to compute the address.
func NewAWSSigner(ctx context.Context, client *kms.Client, keyID string, chainID int64) (*AWSSigner, error) {
	return NewAWSSignerWithTimeout(ctx, client, keyID, chainID, defaultKMSTimeout)
}

// NewAWSSignerWithTimeout creates a new signer with a custom timeout for KMS operations.
func NewAWSSignerWithTimeout(ctx context.Context, client *kms.Client, keyID string, chainID int64, timeout time.Duration) (*AWSSigner, error) {
	pubKeyResp, err := client.GetPublicKey(ctx, &kms.GetPublicKeyInput{
		KeyId: &keyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from KMS: %w", err)
	}

	// AWS KMS returns public key in DER format (SubjectPublicKeyInfo)
	parsedKey, err := x509.ParsePKIXPublicKey(pubKeyResp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an ECDSA key")
	}

	address := crypto.PubkeyToAddress(*ecdsaKey)

	return &AWSSigner{
		client:  client,
		keyID:   keyID,
		chainID: big.NewInt(chainID),
		pubKey:  ecdsaKey,
		address: address,
		timeout: timeout,
	}, nil
}

func (s *AWSSigner) Address() common.Address {
	return s.address
}

func (s *AWSSigner) ChainID() *big.Int {
	return s.chainID
}

// SignTypedData signs EIP-712 typed data using AWS KMS.
func (s *AWSSigner) SignTypedData(domain *apitypes.TypedDataDomain, typesDef apitypes.Types, message apitypes.TypedDataMessage, primaryType string) ([]byte, error) {
	typedData := apitypes.TypedData{
		Types:       typesDef,
		PrimaryType: primaryType,
		Domain:      *domain,
		Message:     message,
	}

	sighash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("failed to hash typed data: %w", err)
	}

	// Sign with KMS using a timeout context
	signInput := &kms.SignInput{
		KeyId:            &s.keyID,
		Message:          sighash,
		MessageType:      types.MessageTypeDigest,
		SigningAlgorithm: types.SigningAlgorithmSpecEcdsaSha256,
	}

	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	signOutput, err := s.client.Sign(ctx, signInput)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with KMS: %w", err)
	}

	// Unmarshal ASN.1 signature to R and S
	var sig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(signOutput.Signature, &sig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ASN.1 signature: %w", err)
	}

	// Canonicalize S: s = min(s, N-s) if s > N/2
	// secp256k1 N
	curveOrder := crypto.S256().Params().N
	halfOrder := new(big.Int).Div(curveOrder, big.NewInt(2))

	if sig.S.Cmp(halfOrder) > 0 {
		sig.S = new(big.Int).Sub(curveOrder, sig.S)
	}

	// Convert to 65-byte [R, S, V] format
	// R and S are 32 bytes each.
	rBytes := sig.R.Bytes()
	sBytes := sig.S.Bytes()

	// Pad R and S to 32 bytes
	sigBytes := make([]byte, 65)
	copy(sigBytes[32-len(rBytes):32], rBytes)
	copy(sigBytes[64-len(sBytes):64], sBytes)

	// Recover V
	// Ecrecover requires the signature to be [R || S || V]
	// We try V = 0 and V = 1
	var v byte
	found := false
	for _, candidateV := range []byte{0, 1} {
		sigBytes[64] = candidateV
		// Ecrecover expects [R || S || V] where V is 0 or 1
		pubKeyBytes, err := crypto.Ecrecover(sighash, sigBytes)
		if err == nil {
			recoveredPub, err := crypto.UnmarshalPubkey(pubKeyBytes)
			if err == nil {
				recoveredAddr := crypto.PubkeyToAddress(*recoveredPub)
				if recoveredAddr == s.address {
					v = candidateV
					found = true
					break
				}
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("failed to recover public key from signature")
	}

	// Adjust V for Ethereum (27 or 28)
	sigBytes[64] = v + 27

	return sigBytes, nil
}
