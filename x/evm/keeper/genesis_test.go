package keeper_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/keeper"
)

func TestInitGenesis(t *testing.T) {
	k := &testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{})
	// coinbase address must be associated
	coinbaseTabiAddr, associated := k.GetTabiAddress(ctx, keeper.GetCoinbaseAddress())
	require.True(t, associated)
	require.True(t, bytes.Equal(coinbaseTabiAddr, k.AccountKeeper().GetModuleAddress("fee_collector")))
}
