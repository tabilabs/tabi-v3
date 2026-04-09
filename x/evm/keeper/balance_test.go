package keeper_test

import (
	"math/big"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/state"
	"github.com/tabilabs/tabi-v3/x/evm/types"
)

func TestGetBalanceUsesAtabiToWeiMultiplier(t *testing.T) {
	k := &testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
	tabiAddr, evmAddr := testkeeper.MockAddressPair()
	k.SetAddressMapping(ctx, tabiAddr, evmAddr)

	require.Equal(t, "1000000000000", state.AtabiToWeiMultiplier.String())

	amt := sdk.NewCoins(sdk.NewCoin(k.GetBaseDenom(ctx), sdk.OneInt()))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, amt))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, tabiAddr, amt))
	require.NoError(t, k.BankKeeper().AddWei(ctx, tabiAddr, sdk.NewInt(7)))

	require.Equal(t, big.NewInt(1_000_000_000_007), k.GetBalance(ctx, tabiAddr))
}
