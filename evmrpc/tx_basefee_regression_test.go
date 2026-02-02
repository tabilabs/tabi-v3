package evmrpc_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	tabiapp "github.com/tabilabs/tabi-v3/app"
	tabirpc "github.com/tabilabs/tabi-v3/evmrpc"
	evmt "github.com/tabilabs/tabi-v3/x/evm/types"
	tabiethtx "github.com/tabilabs/tabi-v3/x/evm/types/ethtx"
	rpcclientmocks "github.com/tendermint/tendermint/rpc/client/mocks"
	"github.com/tendermint/tendermint/rpc/coretypes"
	tmtypes "github.com/tendermint/tendermint/types"
)

func TestGetTransactionByBlockNumberAndIndex_UsesBaseFeeForEffectiveGasPrice(t *testing.T) {
	app := tabiapp.Setup(false, false, false)
	k := &app.EvmKeeper
	ctx := app.GetContextForDeliverTx([]byte{})
	height := ctx.BlockHeight()

	// Make baseFee != nil and distinct from fee cap.
	k.SetCurrBaseFeePerGas(ctx, sdk.NewDec(10))

	to := common.HexToAddress("0x1111111111111111111111111111111111111111")
	ethTx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     0,
		GasTipCap: big.NewInt(2),
		GasFeeCap: big.NewInt(100),
		Gas:       21000,
		To:        &to,
		Value:     big.NewInt(0),
		Data:      nil,
	})

	txData, err := tabiethtx.NewTxDataFromTx(ethTx)
	require.NoError(t, err)
	msg, err := evmt.NewMsgEVMTransaction(txData)
	require.NoError(t, err)

	enc := tabiapp.MakeEncodingConfig()
	txBuilder := enc.TxConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(msg))
	txBuilder.SetGasLimit(21000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("atabi", sdk.NewInt(1))))
	txBz, err := enc.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	// Persist a receipt so the RPC path can resolve block number and baseFee.
	txHash := ethTx.Hash()
	receipt := &evmt.Receipt{
		TxType:            uint32(ethTx.Type()),
		TxHashHex:         txHash.Hex(),
		GasUsed:           21000,
		BlockNumber:       uint64(height),
		TransactionIndex:  0,
		EffectiveGasPrice: 1,
		Status:            uint32(ethtypes.ReceiptStatusSuccessful),
	}
	require.NoError(t, k.SetTransientReceipt(ctx, txHash, receipt))
	require.NoError(t, k.FlushTransientReceipts(ctx))

	block := &coretypes.ResultBlock{
		BlockID: tmtypes.BlockID{Hash: []byte{0x01}},
		Block: &tmtypes.Block{
			Header: tmtypes.Header{Height: height, Time: time.Now()},
			Data:   tmtypes.Data{Txs: tmtypes.Txs{tmtypes.Tx(txBz)}},
		},
	}

	tmClient := &rpcclientmocks.Client{}
	tmClient.On("Block", mock.Anything, mock.Anything).Return(block, nil)

	ctxProvider := func(i int64) sdk.Context {
		if i == tabirpc.LatestCtxHeight {
			return ctx
		}
		return ctx.WithBlockHeight(i)
	}
	txConfigProvider := func(int64) client.TxConfig {
		return enc.TxConfig
	}

	api := tabirpc.NewTransactionAPI(tmClient, k, ctxProvider, txConfigProvider, "", tabirpc.ConnectionTypeHTTP)
	rpcTx, err := api.GetTransactionByBlockNumberAndIndex(context.Background(), rpc.BlockNumber(height), hexutil.Uint(0))
	require.NoError(t, err)
	require.NotNil(t, rpcTx)
	require.NotNil(t, rpcTx.GasPrice)

	// NewRPCTransaction uses: price = min(gasTipCap + baseFee, gasFeeCap)
	require.Equal(t, big.NewInt(12), rpcTx.GasPrice.ToInt())
}
