package ctf

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Standard partition for binary markets (YES/NO).
var (
	BinaryPartition = []*big.Int{big.NewInt(1), big.NewInt(2)}
)

// Request types.
type (
	PrepareConditionRequest struct {
		Oracle           common.Address
		QuestionID       common.Hash
		OutcomeSlotCount *big.Int
	}
	ConditionIDRequest struct {
		Oracle           common.Address
		QuestionID       common.Hash
		OutcomeSlotCount *big.Int
	}
	CollectionIDRequest struct {
		ParentCollectionID common.Hash
		ConditionID        common.Hash
		IndexSet           *big.Int
	}
	PositionIDRequest struct {
		CollateralToken common.Address
		CollectionID    common.Hash
	}
	SplitPositionRequest struct {
		CollateralToken    common.Address
		ParentCollectionID common.Hash
		ConditionID        common.Hash
		Partition          []*big.Int
		Amount             *big.Int
	}
	MergePositionsRequest struct {
		CollateralToken    common.Address
		ParentCollectionID common.Hash
		ConditionID        common.Hash
		Partition          []*big.Int
		Amount             *big.Int
	}
	RedeemPositionsRequest struct {
		CollateralToken    common.Address
		ParentCollectionID common.Hash
		ConditionID        common.Hash
		IndexSets          []*big.Int
	}
	RedeemNegRiskRequest struct {
		ConditionID common.Hash
		Amounts     []*big.Int
	}
)

// Response types.
type (
	PrepareConditionResponse struct {
		TransactionHash common.Hash
		BlockNumber     uint64
	}
	ConditionIDResponse struct {
		ConditionID common.Hash
	}
	CollectionIDResponse struct {
		CollectionID common.Hash
	}
	PositionIDResponse struct {
		PositionID *big.Int
	}
	SplitPositionResponse struct {
		TransactionHash common.Hash
		BlockNumber     uint64
	}
	MergePositionsResponse struct {
		TransactionHash common.Hash
		BlockNumber     uint64
	}
	RedeemPositionsResponse struct {
		TransactionHash common.Hash
		BlockNumber     uint64
	}
	RedeemNegRiskResponse struct {
		TransactionHash common.Hash
		BlockNumber     uint64
	}
)
