package keeper

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/common"
	"github.com/facjas/tabi-db/proto"

	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tabilabs/tabi-v3/utils"
	"github.com/tabilabs/tabi-v3/x/evm/state"
	"github.com/tabilabs/tabi-v3/x/evm/types"
)

// SetTransientReceipt sets a data structure that stores EVM specific transaction metadata.
func (k *Keeper) SetTransientReceipt(ctx sdk.Context, txHash common.Hash, receipt *types.Receipt) error {
	store := ctx.TransientStore(k.transientStoreKey)
	bz, err := receipt.Marshal()
	if err != nil {
		return err
	}
	store.Set(types.ReceiptKey(txHash), bz)
	return nil
}

func (k *Keeper) GetTransientReceipt(ctx sdk.Context, txHash common.Hash) (*types.Receipt, error) {
	store := ctx.TransientStore(k.transientStoreKey)
	bz := store.Get(types.ReceiptKey(txHash))
	if bz == nil {
		return nil, errors.New("not found")
	}
	r := &types.Receipt{}
	if err := r.Unmarshal(bz); err != nil {
		return nil, err
	}
	return r, nil
}

func (k *Keeper) DeleteTransientReceipt(ctx sdk.Context, txHash common.Hash) {
	store := ctx.TransientStore(k.transientStoreKey)
	store.Delete(types.ReceiptKey(txHash))
}

// GetReceipt returns a data structure that stores EVM specific transaction metadata.
// Many EVM applications (e.g. MetaMask) relies on being on able to query receipt
// by EVM transaction hash (not Tabi transaction hash) to function properly.
func (k *Keeper) GetReceipt(ctx sdk.Context, txHash common.Hash) (*types.Receipt, error) {

	// receipts are immutable, use latest version
	lv, err := k.receiptStore.GetLatestVersion()
	if err != nil {
		return nil, err
	}

	// try persistent store
	bz, err := k.receiptStore.Get(types.ReceiptStoreKey, lv, types.ReceiptKey(txHash))
	if err != nil {
		return nil, err
	}

	if bz == nil {
		// try legacy store for older receipts
		store := ctx.KVStore(k.storeKey)
		bz = store.Get(types.ReceiptKey(txHash))
		if bz == nil {
			return nil, errors.New("not found")
		}
	}

	var r types.Receipt
	if err := r.Unmarshal(bz); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetReceiptWithRetry attempts to get a receipt with retries to handle race conditions
// where the receipt might not be immediately available after the transaction.
func (k *Keeper) GetReceiptWithRetry(ctx sdk.Context, txHash common.Hash, maxRetries int) (*types.Receipt, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		receipt, err := k.GetReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		// If it's not a "not found" error, return immediately
		if err.Error() != "not found" {
			return nil, err
		}

		// Wait before retrying, with increasing delay, 200ms, 400ms, 600ms, etc.
		time.Sleep(time.Millisecond * 200 * time.Duration(i+1))
		lastErr = err
	}
	return nil, lastErr
}

//	MockReceipt sets a data structure that stores EVM specific transaction metadata.
//
// this is currently used by a number of tests to set receipts at the moment
func (k *Keeper) MockReceipt(ctx sdk.Context, txHash common.Hash, receipt *types.Receipt) error {
	fmt.Printf("MOCK RECEIPT height=%d, tx=%s\n", ctx.BlockHeight(), txHash.Hex())
	if err := k.SetTransientReceipt(ctx, txHash, receipt); err != nil {
		return err
	}
	return k.FlushTransientReceipts(ctx)
}

func (k *Keeper) FlushTransientReceipts(ctx sdk.Context) error {
	iter := prefix.NewStore(ctx.TransientStore(k.transientStoreKey), types.ReceiptKeyPrefix).Iterator(nil, nil)
	defer iter.Close()

	type receiptWithKey struct {
		txHash  common.Hash
		receipt *types.Receipt
	}
	var receipts []receiptWithKey

	for ; iter.Valid(); iter.Next() {
		r := &types.Receipt{}
		if err := r.Unmarshal(iter.Value()); err != nil {
			return err
		}
		receipts = append(receipts, receiptWithKey{
			txHash:  common.BytesToHash(iter.Key()),
			receipt: r,
		})
	}

	if len(receipts) == 0 {
		return nil
	}

	blockReceipts := make(map[uint64][]receiptWithKey)
	for _, r := range receipts {
		blockReceipts[r.receipt.BlockNumber] = append(blockReceipts[r.receipt.BlockNumber], r)
	}

	for _, recs := range blockReceipts {
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].receipt.TransactionIndex < recs[j].receipt.TransactionIndex
		})
		var cumGas uint64
		for i := range recs {
			if recs[i].receipt.EffectiveGasPrice > 0 {
				cumGas += recs[i].receipt.GasUsed
			}
			recs[i].receipt.CumulativeGasUsed = cumGas
		}
	}

	sort.Slice(receipts, func(i, j int) bool {
		return bytes.Compare(receipts[i].txHash[:], receipts[j].txHash[:]) < 0
	})

	var pairs []*iavl.KVPair
	for _, r := range receipts {
		bz, err := r.receipt.Marshal()
		if err != nil {
			return err
		}
		pairs = append(pairs, &iavl.KVPair{Key: types.ReceiptKey(r.txHash), Value: bz})
	}

	ncs := &proto.NamedChangeSet{
		Name:      types.ReceiptStoreKey,
		Changeset: iavl.ChangeSet{Pairs: pairs},
	}

	return k.receiptStore.ApplyChangesetAsync(ctx.BlockHeight(), []*proto.NamedChangeSet{ncs})
}

func (k *Keeper) WriteReceipt(
	ctx sdk.Context,
	stateDB *state.DBImpl,
	msg *core.Message,
	txType uint32,
	txHash common.Hash,
	gasUsed uint64,
	vmError string,
) (*types.Receipt, error) {
	ethLogs := stateDB.GetAllLogs()
	bloom := ethtypes.CreateBloom(ethtypes.Receipts{&ethtypes.Receipt{Logs: ethLogs}})
	receipt := &types.Receipt{
		TxType:            txType,
		CumulativeGasUsed: uint64(0),
		TxHashHex:         txHash.Hex(),
		GasUsed:           gasUsed,
		BlockNumber:       uint64(ctx.BlockHeight()),
		TransactionIndex:  uint32(ctx.TxIndex()),
		EffectiveGasPrice: msg.GasPrice.Uint64(),
		VmError:           vmError,
		Logs:              utils.Map(ethLogs, ConvertEthLog),
		LogsBloom:         bloom[:],
	}

	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(msg.From, msg.Nonce).Hex()
	} else {
		receipt.To = msg.To.Hex()
		if len(msg.Data) > 0 {
			receipt.ContractAddress = msg.To.Hex()
		}
	}

	if vmError == "" {
		receipt.Status = uint32(ethtypes.ReceiptStatusSuccessful)
	} else {
		receipt.Status = uint32(ethtypes.ReceiptStatusFailed)
	}

	if perr := stateDB.GetPrecompileError(); perr != nil {
		if receipt.Status > 0 {
			ctx.Logger().Error(fmt.Sprintf("Transaction %s succeeded in execution but has precompile error %s", receipt.TxHashHex, perr.Error()))
		} else {
			// append precompile error to VM error
			receipt.VmError = fmt.Sprintf("%s|%s", receipt.VmError, perr.Error())
		}
	}

	receipt.From = msg.From.Hex()

	return receipt, k.SetTransientReceipt(ctx, txHash, receipt)
}
