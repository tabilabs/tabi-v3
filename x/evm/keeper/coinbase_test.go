package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/keeper"
)

func TestGetFeeCollectorAddress(t *testing.T) {
	k := &testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{})
	addr, err := k.GetFeeCollectorAddress(ctx)
	require.Nil(t, err)
	expected := k.GetEVMAddressOrDefault(ctx, k.AccountKeeper().GetModuleAddress("fee_collector"))
	require.Equal(t, expected.Hex(), addr.Hex())
}

func TestGetCoinbaseAddress(t *testing.T) {
	require.Equal(t, "0x27F7B8B8B5A4e71E8E9aA671f4e4031E3773303F", keeper.GetCoinbaseAddress().Hex())
}
