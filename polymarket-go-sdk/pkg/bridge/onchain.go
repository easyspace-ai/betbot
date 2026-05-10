package bridge

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// WithdrawRequest describes an on-chain withdrawal transfer.
type WithdrawRequest struct {
	Amount *big.Int
	Asset  common.Address
	To     common.Address
}
