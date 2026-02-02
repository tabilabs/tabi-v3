package antedecorators_test

import (
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
	"github.com/tabilabs/tabi-v3/app/antedecorators"
	"github.com/tabilabs/tabi-v3/utils"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

var output string

type FakeTx struct{}

func (FakeTx) GetMsgs() []sdk.Msg { return nil }

func (FakeTx) ValidateBasic() error { return nil }

func (FakeTx) GetGasEstimate() uint64 { return 0 }

func (FakeTx) GetSigners() []sdk.AccAddress { return nil }

func (FakeTx) GetPubKeys() ([]cryptotypes.PubKey, error) { return nil, nil }

func (FakeTx) GetSignaturesV2() ([]signing.SignatureV2, error) { return nil, nil }

type FakeAnteDecoratorOne struct{}

func (FakeAnteDecoratorOne) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	output += "one"
	return next(ctx, tx, simulate)
}

type FakeAnteDecoratorTwo struct{}

func (FakeAnteDecoratorTwo) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	output += "two"
	return next(ctx, tx, simulate)
}

type FakeAnteDecoratorThree struct{}

func (FakeAnteDecoratorThree) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	output += "three"
	return next(ctx, tx, simulate)
}

func TestTracedDecorator(t *testing.T) {
	output = ""
	anteDecorators := []sdk.AnteFullDecorator{
		sdk.DefaultWrappedAnteDecorator(FakeAnteDecoratorOne{}),
		sdk.DefaultWrappedAnteDecorator(FakeAnteDecoratorTwo{}),
		sdk.DefaultWrappedAnteDecorator(FakeAnteDecoratorThree{}),
	}
	tracedDecorators := utils.Map(anteDecorators, func(d sdk.AnteFullDecorator) sdk.AnteFullDecorator {
		return sdk.DefaultWrappedAnteDecorator(antedecorators.NewTracedAnteDecorator(d, nil))
	})
	chainedHandler, _ := sdk.ChainAnteDecorators(tracedDecorators...)
	chainedHandler(sdk.NewContext(nil, tmproto.Header{}, false, nil), FakeTx{}, false)
	require.Equal(t, "onetwothree", output)
}
