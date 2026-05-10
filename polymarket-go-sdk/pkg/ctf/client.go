package ctf

import "context"

// Client defines the CTF interface.
type Client interface {
	PrepareCondition(ctx context.Context, req *PrepareConditionRequest) (PrepareConditionResponse, error)
	ConditionID(ctx context.Context, req *ConditionIDRequest) (ConditionIDResponse, error)
	CollectionID(ctx context.Context, req *CollectionIDRequest) (CollectionIDResponse, error)
	PositionID(ctx context.Context, req *PositionIDRequest) (PositionIDResponse, error)

	// Transaction methods
	SplitPosition(ctx context.Context, req *SplitPositionRequest) (SplitPositionResponse, error)
	MergePositions(ctx context.Context, req *MergePositionsRequest) (MergePositionsResponse, error)
	RedeemPositions(ctx context.Context, req *RedeemPositionsRequest) (RedeemPositionsResponse, error)
	RedeemNegRisk(ctx context.Context, req *RedeemNegRiskRequest) (RedeemNegRiskResponse, error)
}
