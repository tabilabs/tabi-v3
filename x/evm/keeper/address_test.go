package keeper_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tabilabs/tabi-v3/testutil/keeper"
	evmkeeper "github.com/tabilabs/tabi-v3/x/evm/keeper"
)

func TestSetGetAddressMapping(t *testing.T) {
	k := &keeper.EVMTestApp.EvmKeeper
	ctx := keeper.EVMTestApp.GetContextForDeliverTx([]byte{})
	tabiAddr, evmAddr := keeper.MockAddressPair()
	_, ok := k.GetEVMAddress(ctx, tabiAddr)
	require.False(t, ok)
	_, ok = k.GetTabiAddress(ctx, evmAddr)
	require.False(t, ok)
	k.SetAddressMapping(ctx, tabiAddr, evmAddr)
	foundEVM, ok := k.GetEVMAddress(ctx, tabiAddr)
	require.True(t, ok)
	require.Equal(t, evmAddr, foundEVM)
	foundAtabi, ok := k.GetTabiAddress(ctx, evmAddr)
	require.True(t, ok)
	require.Equal(t, tabiAddr, foundAtabi)
	require.Equal(t, tabiAddr, k.AccountKeeper().GetAccount(ctx, tabiAddr).GetAddress())
}

func TestDeleteAddressMapping(t *testing.T) {
	k := &keeper.EVMTestApp.EvmKeeper
	ctx := keeper.EVMTestApp.GetContextForDeliverTx([]byte{})
	tabiAddr, evmAddr := keeper.MockAddressPair()
	k.SetAddressMapping(ctx, tabiAddr, evmAddr)
	foundEVM, ok := k.GetEVMAddress(ctx, tabiAddr)
	require.True(t, ok)
	require.Equal(t, evmAddr, foundEVM)
	foundAtabi, ok := k.GetTabiAddress(ctx, evmAddr)
	require.True(t, ok)
	require.Equal(t, tabiAddr, foundAtabi)
	k.DeleteAddressMapping(ctx, tabiAddr, evmAddr)
	_, ok = k.GetEVMAddress(ctx, tabiAddr)
	require.False(t, ok)
	_, ok = k.GetTabiAddress(ctx, evmAddr)
	require.False(t, ok)
}

func TestGetAddressOrDefault(t *testing.T) {
	k := &keeper.EVMTestApp.EvmKeeper
	ctx := keeper.EVMTestApp.GetContextForDeliverTx([]byte{})
	tabiAddr, evmAddr := keeper.MockAddressPair()
	defaultEvmAddr := k.GetEVMAddressOrDefault(ctx, tabiAddr)
	require.True(t, bytes.Equal(tabiAddr, defaultEvmAddr[:]))
	defaultTabiAddr := k.GetTabiAddressOrDefault(ctx, evmAddr)
	require.True(t, bytes.Equal(defaultTabiAddr, evmAddr[:]))
}

func TestSendingToCastAddress(t *testing.T) {
	a := keeper.EVMTestApp
	ctx := a.GetContextForDeliverTx([]byte{})
	tabiAddr, evmAddr := keeper.MockAddressPair()
	castAddr := sdk.AccAddress(evmAddr[:])
	sourceAddr, _ := keeper.MockAddressPair()
	require.Nil(t, a.BankKeeper.MintCoins(ctx, "evm", sdk.NewCoins(sdk.NewCoin("atabi", sdk.NewInt(10)))))
	require.Nil(t, a.BankKeeper.SendCoinsFromModuleToAccount(ctx, "evm", sourceAddr, sdk.NewCoins(sdk.NewCoin("atabi", sdk.NewInt(5)))))
	amt := sdk.NewCoins(sdk.NewCoin("atabi", sdk.NewInt(1)))
	require.Nil(t, a.BankKeeper.SendCoinsFromModuleToAccount(ctx, "evm", castAddr, amt))
	require.Nil(t, a.BankKeeper.SendCoins(ctx, sourceAddr, castAddr, amt))
	require.Nil(t, a.BankKeeper.SendCoinsAndWei(ctx, sourceAddr, castAddr, sdk.OneInt(), sdk.OneInt()))

	a.EvmKeeper.SetAddressMapping(ctx, tabiAddr, evmAddr)
	require.NotNil(t, a.BankKeeper.SendCoinsFromModuleToAccount(ctx, "evm", castAddr, amt))
	require.NotNil(t, a.BankKeeper.SendCoins(ctx, sourceAddr, castAddr, amt))
	require.NotNil(t, a.BankKeeper.SendCoinsAndWei(ctx, sourceAddr, castAddr, sdk.OneInt(), sdk.OneInt()))
}

func TestEvmAddressHandler_GetTabiAddressFromString(t *testing.T) {
	a := keeper.EVMTestApp
	ctx := a.GetContextForDeliverTx([]byte{})
	tabiAddr, evmAddr := keeper.MockAddressPair()
	a.EvmKeeper.SetAddressMapping(ctx, tabiAddr, evmAddr)

	_, notAssociatedEvmAddr := keeper.MockAddressPair()
	castAddr := sdk.AccAddress(notAssociatedEvmAddr[:])

	type args struct {
		ctx     sdk.Context
		address string
	}
	tests := []struct {
		name       string
		args       args
		want       sdk.AccAddress
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "returns associated Tabi address if input address is a valid 0x and associated",
			args: args{
				ctx:     ctx,
				address: evmAddr.String(),
			},
			want: tabiAddr,
		},
		{
			name: "returns default Tabi address if input address is a valid 0x not associated",
			args: args{
				ctx:     ctx,
				address: notAssociatedEvmAddr.String(),
			},
			want: castAddr,
		},
		{
			name: "returns Tabi address if input address is a valid bech32 address",
			args: args{
				ctx:     ctx,
				address: tabiAddr.String(),
			},
			want: tabiAddr,
		},
		{
			name: "returns error if address is invalid",
			args: args{
				ctx:     ctx,
				address: "invalid",
			},
			wantErr:    true,
			wantErrMsg: "decoding bech32 failed: invalid bech32 string length 7",
		}, {
			name: "returns error if address is empty",
			args: args{
				ctx:     ctx,
				address: "",
			},
			wantErr:    true,
			wantErrMsg: "empty address string is not allowed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := evmkeeper.NewEvmAddressHandler(&a.EvmKeeper)
			got, err := h.GetTabiAddressFromString(tt.args.ctx, tt.args.address)
			if tt.wantErr {
				require.NotNil(t, err)
				require.Equal(t, tt.wantErrMsg, err.Error())
				return
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
