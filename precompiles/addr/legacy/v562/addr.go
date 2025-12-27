package v562

import (
	"bytes"
	"embed"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	pcommon "github.com/tabilabs/tabi-v3/precompiles/common/legacy/v562"
	"github.com/tabilabs/tabi-v3/utils/metrics"
	"github.com/tabilabs/tabi-v3/x/evm/types"
)

const (
	GetTabiAddressMethod = "getTabiAddr"
	GetEvmAddressMethod  = "getEvmAddr"
)

const (
	AddrAddress = "0x0000000000000000000000000000000000001004"
)

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

type PrecompileExecutor struct {
	evmKeeper pcommon.EVMKeeper

	GetTabiAddressID []byte
	GetEvmAddressID  []byte
}

func NewPrecompile(evmKeeper pcommon.EVMKeeper) (*pcommon.Precompile, error) {
	abiBz, err := f.ReadFile("abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading the addr ABI %s", err)
	}

	newAbi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return nil, err
	}

	p := &PrecompileExecutor{
		evmKeeper: evmKeeper,
	}

	for name, m := range newAbi.Methods {
		switch name {
		case GetTabiAddressMethod:
			p.GetTabiAddressID = m.ID
		case GetEvmAddressMethod:
			p.GetEvmAddressID = m.ID
		}
	}

	return pcommon.NewPrecompile(newAbi, p, common.HexToAddress(AddrAddress), "addr"), nil
}

// RequiredGas returns the required bare minimum gas to execute the precompile.
func (p PrecompileExecutor) RequiredGas(input []byte, method *abi.Method) uint64 {
	return pcommon.DefaultGasCost(input, p.IsTransaction(method.Name))
}

func (p PrecompileExecutor) Execute(ctx sdk.Context, method *abi.Method, _ common.Address, _ common.Address, args []interface{}, value *big.Int, _ bool, _ *vm.EVM, hooks *tracing.Hooks) (bz []byte, err error) {
	switch method.Name {
	case GetTabiAddressMethod:
		return p.getTabiAddr(ctx, method, args, value)
	case GetEvmAddressMethod:
		return p.getEvmAddr(ctx, method, args, value)
	}
	return
}

func (p PrecompileExecutor) getTabiAddr(ctx sdk.Context, method *abi.Method, args []interface{}, value *big.Int) ([]byte, error) {
	if err := pcommon.ValidateNonPayable(value); err != nil {
		return nil, err
	}

	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}

	tabiAddr, found := p.evmKeeper.GetTabiAddress(ctx, args[0].(common.Address))
	if !found {
		metrics.IncrementAssociationError("getTabiAddr", types.NewAssociationMissingErr(args[0].(common.Address).Hex()))
		return nil, fmt.Errorf("EVM address %s is not associated", args[0].(common.Address).Hex())
	}
	return method.Outputs.Pack(tabiAddr.String())
}

func (p PrecompileExecutor) getEvmAddr(ctx sdk.Context, method *abi.Method, args []interface{}, value *big.Int) ([]byte, error) {
	if err := pcommon.ValidateNonPayable(value); err != nil {
		return nil, err
	}

	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}

	tabiAddr, err := sdk.AccAddressFromBech32(args[0].(string))
	if err != nil {
		return nil, err
	}

	evmAddr, found := p.evmKeeper.GetEVMAddress(ctx, tabiAddr)
	if !found {
		metrics.IncrementAssociationError("getEvmAddr", types.NewAssociationMissingErr(args[0].(string)))
		return nil, fmt.Errorf("tabi address %s is not associated", args[0].(string))
	}
	return method.Outputs.Pack(evmAddr)
}

func (PrecompileExecutor) IsTransaction(string) bool {
	return false
}
