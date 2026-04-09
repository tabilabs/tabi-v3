package state

import (
	"encoding/binary"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AtabiToWeiMultiplier converts atabi amounts to wei.
//
// On this chain, 1 atabi = 10^12 wei, so 1 TABI = 10^18 wei.
var AtabiToWeiMultiplier = big.NewInt(1_000_000_000_000)
var SdkAtabiToWeiMultiplier = sdk.NewIntFromBigInt(AtabiToWeiMultiplier)

var CoinbaseAddressPrefix = []byte("evm_coinbase")

func GetCoinbaseAddress(txIdx int) sdk.AccAddress {
	txIndexBz := make([]byte, 8)
	binary.BigEndian.PutUint64(txIndexBz, uint64(txIdx))
	return append(CoinbaseAddressPrefix, txIndexBz...)
}

func SplitAtabiWeiAmount(amt *big.Int) (sdk.Int, sdk.Int) {
	wei := new(big.Int).Mod(amt, AtabiToWeiMultiplier)
	atabi := new(big.Int).Quo(amt, AtabiToWeiMultiplier)
	return sdk.NewIntFromBigInt(atabi), sdk.NewIntFromBigInt(wei)
}
