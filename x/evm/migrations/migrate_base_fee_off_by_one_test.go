package migrations_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/migrations"
)

func TestMigrateBaseFeeOffByOne(t *testing.T) {
	k := testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockHeight(8)
	bf := sdk.NewDec(100)
	k.SetCurrBaseFeePerGas(ctx, bf)
	require.Equal(t, k.GetMinimumFeePerGas(ctx), k.GetNextBaseFeePerGas(ctx))
	// do the migration
	require.Nil(t, migrations.MigrateBaseFeeOffByOne(ctx, &k))
	require.Equal(t, bf, k.GetNextBaseFeePerGas(ctx))
	require.Equal(t, bf, k.GetCurrBaseFeePerGas(ctx))
}
