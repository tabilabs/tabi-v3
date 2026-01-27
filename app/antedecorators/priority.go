package antedecorators

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EVMAssociatePriority = math.MaxInt64 - 101
	MaxPriority          = math.MaxInt64 - 1000
)

type PriorityDecorator struct{}

func NewPriorityDecorator() PriorityDecorator {
	return PriorityDecorator{}
}

func intMin(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (pd PriorityDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	priority := intMin(ctx.Priority(), MaxPriority)
	newCtx := ctx.WithPriority(priority)
	return next(newCtx, tx, simulate)
}
