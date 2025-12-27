package wasmbinding

import (
	"encoding/json"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	OracleRoute       = "oracle"
	EpochRoute        = "epoch"
	TokenFactoryRoute = "tokenfactory"
	EVMRoute          = "evm"
	StakingExtRoute   = "stakingext"
)

type TabiQueryWrapper struct {
	// specifies which module handler should handle the query
	Route string `json:"route,omitempty"`
	// The query data that should be parsed into the module query
	QueryData json.RawMessage `json:"query_data,omitempty"`
}

func CustomQuerier(qp *QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var contractQuery TabiQueryWrapper
		if err := json.Unmarshal(request, &contractQuery); err != nil {
			return nil, sdkerrors.Wrap(err, "Error parsing request data")
		}
		switch contractQuery.Route {
		case EVMRoute:
			return qp.HandleEVMQuery(ctx, contractQuery.QueryData)
		case StakingExtRoute:
			return qp.HandleStakingExtQuery(ctx, contractQuery.QueryData)
		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "Unknown Tabi Query Route"}
		}
	}
}
