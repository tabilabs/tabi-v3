package state_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/state"
)

func TestEventlessTransfer(t *testing.T) {
	k := &testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
	db := state.NewDBImpl(ctx, k, false)
	_, fromAddr := testkeeper.MockAddressPair()
	_, toAddr := testkeeper.MockAddressPair()

	beforeLen := len(ctx.EventManager().ABCIEvents())

	state.TransferWithoutEvents(db, fromAddr, toAddr, big.NewInt(100))

	// should be unchanged
	require.Len(t, ctx.EventManager().ABCIEvents(), beforeLen)
}
