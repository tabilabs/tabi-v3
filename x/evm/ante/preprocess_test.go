package ante_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/ante"
	"github.com/tabilabs/tabi-v3/x/evm/types"
)

func TestIsAccountBalancePositive(t *testing.T) {
	k := &testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
	preprocessor := ante.NewEVMPreprocessDecorator(k, k.AccountKeeper())
	tabiAddr, evmAddr := testkeeper.MockAddressPair()

	require.False(t, preprocessor.IsAccountBalancePositive(ctx, tabiAddr, evmAddr))

	mintAndSendAtabi := func(addr sdk.AccAddress, amount int64) {
		funds := sdk.NewCoins(sdk.NewCoin(k.GetBaseDenom(ctx), sdk.NewInt(amount)))
		require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, funds))
		require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, funds))
	}

	mintAndSendAtabi(tabiAddr, 1)
	require.True(t, preprocessor.IsAccountBalancePositive(ctx, tabiAddr, evmAddr))

	ctx = testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
	preprocessor = ante.NewEVMPreprocessDecorator(k, k.AccountKeeper())
	tabiAddr, evmAddr = testkeeper.MockAddressPair()
	mintAndSendAtabi(sdk.AccAddress(evmAddr[:]), 1)
	require.True(t, preprocessor.IsAccountBalancePositive(ctx, tabiAddr, evmAddr))

	ctx = testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
	preprocessor = ante.NewEVMPreprocessDecorator(k, k.AccountKeeper())
	tabiAddr, evmAddr = testkeeper.MockAddressPair()
	require.NoError(t, k.BankKeeper().AddWei(ctx, tabiAddr, sdk.OneInt()))
	require.True(t, preprocessor.IsAccountBalancePositive(ctx, tabiAddr, evmAddr))

	ctx = testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
	preprocessor = ante.NewEVMPreprocessDecorator(k, k.AccountKeeper())
	tabiAddr, evmAddr = testkeeper.MockAddressPair()
	require.NoError(t, k.BankKeeper().AddWei(ctx, sdk.AccAddress(evmAddr[:]), sdk.OneInt()))
	require.True(t, preprocessor.IsAccountBalancePositive(ctx, tabiAddr, evmAddr))
}
