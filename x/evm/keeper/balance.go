package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tabilabs/tabi-v3/x/evm/state"
)

func (k *Keeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress) *big.Int {
	denom := k.GetBaseDenom(ctx)
	allAtabi := k.BankKeeper().GetBalance(ctx, addr, denom).Amount
	lockedAtabi := k.BankKeeper().LockedCoins(ctx, addr).AmountOf(denom) // LockedCoins doesn't use iterators
	atabi := allAtabi.Sub(lockedAtabi)
	wei := k.BankKeeper().GetWeiBalance(ctx, addr)
	return atabi.Mul(state.SdkAtabiToWeiMultiplier).Add(wei).BigInt()
}
