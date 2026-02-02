package evmrpc

import (
	"context"
	"sync"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	rpcclientmocks "github.com/tendermint/tendermint/rpc/client/mocks"
	"github.com/tendermint/tendermint/rpc/coretypes"
	tmtypes "github.com/tendermint/tendermint/types"
)

func TestLogFetcher_processBatch_DoesNotSkipWhenBloomUntrusted(t *testing.T) {
	oldCache := globalBlockCache
	oldSem := dbReadSemaphore
	oldMutex := cacheCreationMutex
	t.Cleanup(func() {
		globalBlockCache = oldCache
		dbReadSemaphore = oldSem
		cacheCreationMutex = oldMutex
	})

	globalBlockCache = NewBlockCache(10)
	dbReadSemaphore = make(chan struct{}, MaxDBReadConcurrency)
	cacheCreationMutex = sync.Mutex{}

	height := int64(10)

	tmClient := &rpcclientmocks.Client{}
	tmClient.On("Block", mock.Anything, mock.Anything).Return(
		func(_ context.Context, h *int64) *coretypes.ResultBlock {
			b := &tmtypes.Block{}
			b.Header.Height = *h
			return &coretypes.ResultBlock{Block: b}
		},
		nil,
	)

	f := &LogFetcher{
		tmClient: tmClient,
		// Force a ctxProvider fallback (height mismatch) to exercise the
		// "bloom cannot be trusted" path.
		ctxProvider: func(_ int64) sdk.Context {
			return sdk.Context{}.WithBlockHeight(height - 1)
		},
	}

	crit := filters.FilterCriteria{Addresses: []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111")}}
	bloomIdx := EncodeFilters(crit.Addresses, crit.Topics)

	res := make(chan *coretypes.ResultBlock, 1)
	errChan := make(chan error, 1)
	f.processBatch(context.Background(), height, height, crit, bloomIdx, res, errChan)

	select {
	case blk := <-res:
		require.NotNil(t, blk)
		require.NotNil(t, blk.Block)
		require.Equal(t, height, blk.Block.Height)
	default:
		t.Fatalf("expected block at height %d to be fetched, but res channel is empty", height)
	}
}
