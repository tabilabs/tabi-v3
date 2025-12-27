package evmrpc_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tabilabs/tabi-v3/evmrpc"
)

func TestClientVersion(t *testing.T) {
	w := evmrpc.Web3API{}
	require.NotEmpty(t, w.ClientVersion())
}
