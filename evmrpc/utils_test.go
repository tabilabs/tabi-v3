package evmrpc_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tabilabs/tabi-v3/evmrpc"
)

func TestParallelRunnerPanicRecovery(t *testing.T) {
	r := evmrpc.NewParallelRunner(10, 10)
	r.Queue <- func() {
		panic("should be handled")
	}
	close(r.Queue)
	require.NotPanics(t, r.Done.Wait)
}
