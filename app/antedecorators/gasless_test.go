package antedecorators_test

import (
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkacltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tabilabs/tabi-v3/app/antedecorators"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

// =============================================================================
// Test Fixtures
// =============================================================================

// FakeTxForGasless implements sdk.Tx for testing
type FakeTxForGasless struct {
	msgs []sdk.Msg
}

func (f FakeTxForGasless) GetMsgs() []sdk.Msg { return f.msgs }

func (f FakeTxForGasless) ValidateBasic() error { return nil }

func (f FakeTxForGasless) GetGasEstimate() uint64 { return 0 }

func (f FakeTxForGasless) GetSigners() []sdk.AccAddress { return nil }

func (f FakeTxForGasless) GetPubKeys() ([]cryptotypes.PubKey, error) { return nil, nil }

func (f FakeTxForGasless) GetSignaturesV2() ([]signing.SignatureV2, error) { return nil, nil }

// TrackingAnteDecorator tracks whether AnteHandle was called and with what context
type TrackingAnteDecorator struct {
	Called        bool
	CalledWith    sdk.Context
	SimulateValue bool
}

func (t *TrackingAnteDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	t.Called = true
	t.CalledWith = ctx
	t.SimulateValue = simulate
	return next(ctx, tx, simulate)
}

func (t *TrackingAnteDecorator) AnteDeps(txDeps []sdkacltypes.AccessOperation, tx sdk.Tx, txIndex int, next sdk.AnteDepGenerator) ([]sdkacltypes.AccessOperation, error) {
	return next(txDeps, tx, txIndex)
}

// depAddingDecorator appends a sentinel AccessOperation in AnteDeps so tests can
// verify deps aggregation and ordering.
type depAddingDecorator struct {
	Called bool
	Op     sdkacltypes.AccessOperation
}

func (d *depAddingDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}

func (d *depAddingDecorator) AnteDeps(txDeps []sdkacltypes.AccessOperation, tx sdk.Tx, txIndex int, next sdk.AnteDepGenerator) ([]sdkacltypes.AccessOperation, error) {
	d.Called = true
	txDeps = append(txDeps, d.Op)
	return next(txDeps, tx, txIndex)
}

// gasMeterProbeDecorator consumes 1 gas inside AnteHandle and records the delta.
// If a NoConsumptionInfiniteGasMeter leaks into wrapped handlers, delta will be 0.
type gasMeterProbeDecorator struct {
	Delta sdk.Gas
}

func (p *gasMeterProbeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	before := ctx.GasMeter().GasConsumed()
	ctx.GasMeter().ConsumeGas(1, "probe")
	after := ctx.GasMeter().GasConsumed()
	p.Delta = after - before
	return next(ctx, tx, simulate)
}

func (p *gasMeterProbeDecorator) AnteDeps(txDeps []sdkacltypes.AccessOperation, tx sdk.Tx, txIndex int, next sdk.AnteDepGenerator) ([]sdkacltypes.AccessOperation, error) {
	return next(txDeps, tx, txIndex)
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestGaslessDecorator_NewGaslessDecorator(t *testing.T) {
	wrapped := []sdk.AnteFullDecorator{}
	decorator := antedecorators.NewGaslessDecorator(wrapped, nil)

	assert.NotNil(t, decorator)
}

// =============================================================================
// Wrapped Handlers Called Tests - 验证 wrapped handlers 被调用
// =============================================================================

func TestGaslessDecorator_WrappedHandlers_AreCalled(t *testing.T) {
	t.Run("all wrapped handlers are called for non-gasless tx", func(t *testing.T) {
		tracker1 := &TrackingAnteDecorator{}
		tracker2 := &TrackingAnteDecorator{}
		tracker3 := &TrackingAnteDecorator{}

		wrapped := []sdk.AnteFullDecorator{
			sdk.DefaultWrappedAnteDecorator(tracker1),
			sdk.DefaultWrappedAnteDecorator(tracker2),
			sdk.DefaultWrappedAnteDecorator(tracker3),
		}
		decorator := antedecorators.NewGaslessDecorator(wrapped, nil)

		ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
		tx := FakeTxForGasless{msgs: []sdk.Msg{}} // 空 tx = 非 gasless

		nextCalled := false
		next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			nextCalled = true
			return ctx, nil
		}

		wrappedDecorator := sdk.DefaultWrappedAnteDecorator(decorator)
		_, err := wrappedDecorator.AnteHandle(ctx, tx, false, next)

		require.NoError(t, err)

		// 关键断言：验证每个 wrapped handler 都被调用
		assert.True(t, tracker1.Called, "tracker1 should be called")
		assert.True(t, tracker2.Called, "tracker2 should be called")
		assert.True(t, tracker3.Called, "tracker3 should be called")
		assert.True(t, nextCalled, "next handler should be called")
	})

	t.Run("empty wrapped handlers still calls next", func(t *testing.T) {
		decorator := antedecorators.NewGaslessDecorator([]sdk.AnteFullDecorator{}, nil)

		ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
		tx := FakeTxForGasless{msgs: []sdk.Msg{}}

		nextCalled := false
		next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			nextCalled = true
			return ctx, nil
		}

		wrappedDecorator := sdk.DefaultWrappedAnteDecorator(decorator)
		_, err := wrappedDecorator.AnteHandle(ctx, tx, false, next)

		require.NoError(t, err)
		assert.True(t, nextCalled, "next should be called even with empty wrapped handlers")
	})
}

// =============================================================================
// Context Propagation Tests - 验证上下文正确传递
// =============================================================================

func TestGaslessDecorator_ContextPropagation(t *testing.T) {
	t.Run("simulate flag propagates correctly", func(t *testing.T) {
		tracker := &TrackingAnteDecorator{}

		wrapped := []sdk.AnteFullDecorator{
			sdk.DefaultWrappedAnteDecorator(tracker),
		}
		decorator := antedecorators.NewGaslessDecorator(wrapped, nil)

		ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
		tx := FakeTxForGasless{msgs: []sdk.Msg{}}

		next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			return ctx, nil
		}

		// 测试 simulate = true
		wrappedDecorator := sdk.DefaultWrappedAnteDecorator(decorator)
		_, err := wrappedDecorator.AnteHandle(ctx, tx, true, next)

		require.NoError(t, err)
		assert.True(t, tracker.Called, "tracker should be called")
		assert.True(t, tracker.SimulateValue, "simulate flag should be true when passed as true")
	})
}

// =============================================================================
// IsTxGasless Tests
// =============================================================================

func TestIsTxGasless_EmptyTx_ReturnsFalse(t *testing.T) {
	tx := FakeTxForGasless{msgs: []sdk.Msg{}}
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)

	result, err := antedecorators.IsTxGasless(tx, ctx, nil)

	require.NoError(t, err)
	assert.False(t, result, "empty tx should not be gasless")
}

func TestIsTxGasless_NilMsgs_ReturnsFalse(t *testing.T) {
	tx := FakeTxForGasless{msgs: nil}
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)

	result, err := antedecorators.IsTxGasless(tx, ctx, nil)

	require.NoError(t, err)
	assert.False(t, result, "tx with nil msgs should not be gasless")
}

// =============================================================================
// Gas Meter Behavior Tests
// =============================================================================

func TestNoConsumptionInfiniteGasMeter_DoesNotConsumeGas(t *testing.T) {
	meter := storetypes.NewNoConsumptionInfiniteGasMeter()

	// 消耗 gas
	meter.ConsumeGas(1000, "test operation")
	meter.ConsumeGas(5000, "another operation")

	// Gas consumed 应该仍然是 0
	assert.Equal(t, sdk.Gas(0), meter.GasConsumed(),
		"NoConsumptionInfiniteGasMeter should not consume gas")
}

func TestNoConsumptionInfiniteGasMeter_HasInfiniteLimit(t *testing.T) {
	meter := storetypes.NewNoConsumptionInfiniteGasMeter()

	// 不应该 panic，即使消耗大量 gas
	assert.NotPanics(t, func() {
		meter.ConsumeGas(1<<62, "huge amount")
	}, "should not panic on large gas consumption")
}

// =============================================================================
// Error Propagation Tests
// =============================================================================

func TestGaslessDecorator_ErrorPropagation(t *testing.T) {
	t.Run("error in wrapped handler stops chain", func(t *testing.T) {
		errorHandler := &ErrorAnteDecorator{err: assert.AnError}
		tracker := &TrackingAnteDecorator{}

		wrapped := []sdk.AnteFullDecorator{
			sdk.DefaultWrappedAnteDecorator(errorHandler),
			sdk.DefaultWrappedAnteDecorator(tracker),
		}
		decorator := antedecorators.NewGaslessDecorator(wrapped, nil)

		ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
		tx := FakeTxForGasless{msgs: []sdk.Msg{}}

		nextCalled := false
		next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			nextCalled = true
			return ctx, nil
		}

		wrappedDecorator := sdk.DefaultWrappedAnteDecorator(decorator)
		_, err := wrappedDecorator.AnteHandle(ctx, tx, false, next)

		require.Error(t, err, "error should propagate")
		assert.False(t, tracker.Called, "subsequent handlers should not be called after error")
		assert.False(t, nextCalled, "next should not be called after error")
	})
}

// ErrorAnteDecorator always returns an error
type ErrorAnteDecorator struct {
	err error
}

func (e *ErrorAnteDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return ctx, e.err
}

func (e *ErrorAnteDecorator) AnteDeps(txDeps []sdkacltypes.AccessOperation, tx sdk.Tx, txIndex int, next sdk.AnteDepGenerator) ([]sdkacltypes.AccessOperation, error) {
	return next(txDeps, tx, txIndex)
}

// =============================================================================
// AnteDeps Tests
// =============================================================================

func TestGaslessDecorator_AnteDeps_CallsAllDeps(t *testing.T) {
	dep1 := &depAddingDecorator{Op: sdkacltypes.AccessOperation{IdentifierTemplate: "dep1"}}
	dep2 := &depAddingDecorator{Op: sdkacltypes.AccessOperation{IdentifierTemplate: "dep2"}}
	wrapped := []sdk.AnteFullDecorator{dep1, dep2}
	decorator := antedecorators.NewGaslessDecorator(wrapped, nil)

	tx := FakeTxForGasless{msgs: []sdk.Msg{}}
	initialDeps := []sdkacltypes.AccessOperation{}

	nextCalled := false
	next := func(txDeps []sdkacltypes.AccessOperation, _ sdk.Tx, _ int) ([]sdkacltypes.AccessOperation, error) {
		nextCalled = true
		require.Len(t, txDeps, 2)
		require.Equal(t, "dep1", txDeps[0].IdentifierTemplate)
		require.Equal(t, "dep2", txDeps[1].IdentifierTemplate)
		return txDeps, nil
	}

	_, err := decorator.AnteDeps(initialDeps, tx, 0, next)
	require.NoError(t, err)
	require.True(t, dep1.Called)
	require.True(t, dep2.Called)
	require.True(t, nextCalled)
}

func TestGaslessDecorator_NonGaslessTx_DoesNotLeakNoConsumptionGasMeter(t *testing.T) {
	probe := &gasMeterProbeDecorator{}
	decorator := antedecorators.NewGaslessDecorator([]sdk.AnteFullDecorator{probe}, nil)

	ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil).WithGasMeter(sdk.NewGasMeter(1000, 1, 1))
	tx := FakeTxForGasless{msgs: []sdk.Msg{}} // empty msgs => non-gasless

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.NoError(t, err)
	require.Equal(t, sdk.Gas(1), probe.Delta)
}

// =============================================================================
// CheckTx vs DeliverTx Behavior Tests
// 注意：完整测试需要 evmKeeper mock，这里只测试可测试的部分
// =============================================================================

func TestGaslessDecorator_CheckTxContext(t *testing.T) {
	tracker := &TrackingAnteDecorator{}

	wrapped := []sdk.AnteFullDecorator{
		sdk.DefaultWrappedAnteDecorator(tracker),
	}
	decorator := antedecorators.NewGaslessDecorator(wrapped, nil)

	// CheckTx context
	ctx := sdk.NewContext(nil, tmproto.Header{}, true, nil) // isCheckTx = true
	tx := FakeTxForGasless{msgs: []sdk.Msg{}}

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	wrappedDecorator := sdk.DefaultWrappedAnteDecorator(decorator)
	_, err := wrappedDecorator.AnteHandle(ctx, tx, false, next)

	require.NoError(t, err)
	// 非 gasless tx 在 checkTx 时应该调用 wrapped handlers
	assert.True(t, tracker.Called, "wrapped handler should be called for non-gasless tx in checkTx")
}

func TestGaslessDecorator_DeliverTxContext(t *testing.T) {
	tracker := &TrackingAnteDecorator{}

	wrapped := []sdk.AnteFullDecorator{
		sdk.DefaultWrappedAnteDecorator(tracker),
	}
	decorator := antedecorators.NewGaslessDecorator(wrapped, nil)

	// DeliverTx context (isCheckTx = false, isReCheckTx = false, simulate = false)
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
	tx := FakeTxForGasless{msgs: []sdk.Msg{}}

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	wrappedDecorator := sdk.DefaultWrappedAnteDecorator(decorator)
	_, err := wrappedDecorator.AnteHandle(ctx, tx, false, next)

	require.NoError(t, err)
	// DeliverTx 时始终调用 wrapped handlers（根据 gasless.go:33 的逻辑）
	assert.True(t, tracker.Called, "wrapped handler should be called in deliverTx")
}
