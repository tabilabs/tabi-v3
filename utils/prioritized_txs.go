package utils

import sdk "github.com/cosmos/cosmos-sdk/types"

func IsTxPrioritized(tx sdk.Tx) bool {
	return false
}
