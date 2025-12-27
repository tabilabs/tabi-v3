package keeper_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/tabilabs/tabi-v3/precompiles/bank"
	"github.com/tabilabs/tabi-v3/precompiles/gov"
	"github.com/tabilabs/tabi-v3/precompiles/staking"
	"github.com/tabilabs/tabi-v3/testutil/keeper"
	evmkeeper "github.com/tabilabs/tabi-v3/x/evm/keeper"
)

func toAddr(addr string) *common.Address {
	ca := common.HexToAddress(addr)
	return &ca
}

func TestIsPayablePrecompile(t *testing.T) {
	_, evmAddr := keeper.MockAddressPair()
	require.False(t, evmkeeper.IsPayablePrecompile(&evmAddr))
	require.False(t, evmkeeper.IsPayablePrecompile(nil))

	require.True(t, evmkeeper.IsPayablePrecompile(toAddr(bank.BankAddress)))
	require.True(t, evmkeeper.IsPayablePrecompile(toAddr(staking.StakingAddress)))
	require.True(t, evmkeeper.IsPayablePrecompile(toAddr(gov.GovAddress)))
}
