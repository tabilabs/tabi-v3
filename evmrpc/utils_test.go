package evmrpc_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tabilabs/tabi-v3/app"
	"github.com/tabilabs/tabi-v3/evmrpc"
)

func TestCheckVersion(t *testing.T) {
	testApp := app.Setup(false, false, false)
	k := &testApp.EvmKeeper
	ctx := testApp.GetContextForDeliverTx([]byte{}).WithBlockHeight(1)
	testApp.Commit(context.Background()) // bump store version to 1
	require.Nil(t, evmrpc.CheckVersion(ctx, k))
	ctx = ctx.WithBlockHeight(2)
	require.NotNil(t, evmrpc.CheckVersion(ctx, k))
}

func TestParallelRunnerPanicRecovery(t *testing.T) {
	r := evmrpc.NewParallelRunner(10, 10)
	r.Queue <- func() {
		panic("should be handled")
	}
	close(r.Queue)
	require.NotPanics(t, r.Done.Wait)
}
