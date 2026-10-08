package app

import (
	"encoding/json"
	"os"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestWasmRuntimeContractLifecycle(t *testing.T) {
	a := Setup(false, false, false)
	ctx := a.GetContextForDeliverTx(nil)
	ctx = ctx.WithGasMeter(sdk.NewGasMeterWithMultiplier(ctx, 20_000_000))
	creator := sdk.AccAddress([]byte("wasm-runtime-creator"))
	recipient := sdk.AccAddress([]byte("wasm-runtime-receive"))
	a.AccountKeeper.SetAccount(ctx, a.AccountKeeper.NewAccountWithAddress(ctx, creator))
	keeper := wasmkeeper.NewGovPermissionKeeper(a.WasmKeeper)
	code, err := os.ReadFile("../contracts/wasm/cw20_base.wasm")
	require.NoError(t, err)
	codeID, err := keeper.Create(ctx, creator, code, nil)
	require.NoError(t, err)
	initMsg := []byte(`{"name":"Runtime Test","symbol":"RTEST","decimals":6,"initial_balances":[{"address":"` + creator.String() + `","amount":"100"}]}`)
	contract, _, err := keeper.Instantiate(ctx, codeID, creator, nil, initMsg, "runtime-test", nil)
	require.NoError(t, err)
	_, err = keeper.Execute(ctx, contract, creator, []byte(`{"transfer":{"recipient":"`+recipient.String()+`","amount":"10"}}`), nil)
	require.NoError(t, err)
	result, err := a.WasmKeeper.QuerySmart(ctx, contract, []byte(`{"balance":{"address":"`+recipient.String()+`"}}`))
	require.NoError(t, err)
	var balance struct {
		Balance string `json:"balance"`
	}
	require.NoError(t, json.Unmarshal(result, &balance))
	require.Equal(t, "10", balance.Balance)
}
