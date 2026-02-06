package evmrpc

import (
	"context"
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

// The upstream tendermint rpc client interface includes LagStatus, but the
// generated testify mock in tendermint/rpc/client/mocks is missing it in our
// fork. Embed it and add the method so it satisfies rpcclient.Client.
type tmClientMock struct{ rpcclientmocks.Client }

func (m *tmClientMock) LagStatus(ctx context.Context) (*coretypes.ResultLagStatus, error) {
	ret := m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for LagStatus")
	}

	var r0 *coretypes.ResultLagStatus
	if rf, ok := ret.Get(0).(func(context.Context) (*coretypes.ResultLagStatus, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *coretypes.ResultLagStatus); ok {
		r0 = rf(ctx)
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).(*coretypes.ResultLagStatus)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func TestLogFetcher_processBatch_DoesNotSkipWhenBloomUntrusted(t *testing.T) {
	oldCache := globalBlockCache
	oldSem := dbReadSemaphore
	t.Cleanup(func() {
		globalBlockCache = oldCache
		dbReadSemaphore = oldSem
	})

	globalBlockCache = NewBlockCache(10)
	dbReadSemaphore = make(chan struct{}, MaxDBReadConcurrency)

	height := int64(10)

	tmClient := &tmClientMock{}
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
