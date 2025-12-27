package state

import (
	"encoding/binary"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AtabiToWeiMultiplier Fields that were denominated in atabi will be converted to swei (1atabi = 10^12swei)
// for existing Ethereum application (which assumes 18 decimal points) to display properly.
var AtabiToWeiMultiplier = big.NewInt(1)
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
