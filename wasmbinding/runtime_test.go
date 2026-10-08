//go:build cgo

package wasmbinding

import (
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm"
	"github.com/stretchr/testify/require"
)

func TestWasmRuntimeVersion(t *testing.T) {
	version, err := wasmvm.LibwasmvmVersion()
	require.NoError(t, err)
	require.Equal(t, "1.5.9", version, "the node must load the matching security-patched Wasm library")
}
