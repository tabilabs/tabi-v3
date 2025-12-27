package keeper

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/tabilabs/tabi-v3/precompiles/bank"
	"github.com/tabilabs/tabi-v3/precompiles/gov"
	"github.com/tabilabs/tabi-v3/precompiles/staking"
	"github.com/tabilabs/tabi-v3/precompiles/wasmd"
)

// add any payable precompiles here
// these will suppress transfer events to/from the precompile address
var payablePrecompiles = map[string]struct{}{
	bank.BankAddress:       {},
	staking.StakingAddress: {},
	gov.GovAddress:         {},
	wasmd.WasmdAddress:     {},
}

func IsPayablePrecompile(addr *common.Address) bool {
	if addr == nil {
		return false
	}
	_, ok := payablePrecompiles[addr.Hex()]
	return ok
}
