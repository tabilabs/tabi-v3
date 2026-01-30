package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/server"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

type mapAppOptions map[string]interface{}

func (m mapAppOptions) Get(key string) interface{} {
	return m[key]
}

// Setup creates an App instance for tests.
//
// - isCheckTx: when true, skips InitChain.
// - enableCustomEVMPrecompiles: enables custom precompiles in app.New.
// - skipGenesis: when true, skips InitChain even if isCheckTx is false.
func Setup(isCheckTx bool, enableCustomEVMPrecompiles bool, skipGenesis bool) *App {
	encCfg := MakeEncodingConfig()
	chainID := "tendermint_test"

	homePath, err := os.MkdirTemp("", "tabiv3-test-")
	if err != nil {
		panic(err)
	}
	// Ensure directories used by the app exist.
	if err := os.MkdirAll(filepath.Join(homePath, "data"), 0o755); err != nil {
		panic(err)
	}

	appOpts := mapAppOptions{
		"chain-id":                 chainID,
		"evm.http_enabled":         false,
		"evm.ws_enabled":           false,
		server.FlagMinRetainBlocks: 10,
		FlagSCEnable:               false,
		FlagSSEnable:               false,
	}

	a := New(
		log.NewNopLogger(),
		tmdb.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		homePath,
		0,
		enableCustomEVMPrecompiles,
		nil,
		encCfg,
		wasm.EnableAllProposals,
		appOpts,
		nil,
		EmptyACLOpts,
		[]AppOption{
			func(a *App) {
				// Use an in-memory receipt store for tests. This avoids async write timing
				// issues and filesystem dependencies.
				a.receiptStore = NewInMemoryStateStore()
			},
		},
	)

	if isCheckTx || skipGenesis {
		return a
	}

	cp := tmtypes.DefaultConsensusParams().ToProto()
	gs := NewDefaultGenesisState(encCfg.Marshaler)
	gbz, err := json.Marshal(gs)
	if err != nil {
		panic(err)
	}
	_, err = a.InitChain(context.Background(), &abci.RequestInitChain{
		Time:            time.Now(),
		ChainId:         chainID,
		ConsensusParams: &cp,
		Validators:      []abci.ValidatorUpdate{},
		InitialHeight:   1,
		AppStateBytes:   gbz,
	})
	if err != nil {
		panic(err)
	}
	// InitChain runs at height 0. Many tests expect a deliver context at height 1,
	// which normally happens at the first FinalizeBlock.
	if _, err := a.FinalizeBlock(context.Background(), &abci.RequestFinalizeBlock{
		Txs:               nil,
		DecidedLastCommit: abci.CommitInfo{},
		Hash:              []byte("genesis"),
		Height:            1,
		Time:              time.Now(),
	}); err != nil {
		panic(err)
	}

	return a
}
