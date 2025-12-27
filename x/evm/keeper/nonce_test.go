package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
)

func TestNonce(t *testing.T) {
	k := &testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{})
	_, evmAddr := testkeeper.MockAddressPair()
	require.Equal(t, uint64(0), k.GetNonce(ctx, evmAddr))
	k.SetNonce(ctx, evmAddr, 1)
	require.Equal(t, uint64(1), k.GetNonce(ctx, evmAddr))
}
