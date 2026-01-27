package utils

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// maxNestedOracleMsgs 限制 authz 嵌套消息递归深度
const maxNestedOracleMsgs = 5

var oracleMsgTypeURLs = map[string]struct{}{
	// Sei / Oracle 风格的核心消息类型
	"/seiprotocol.sei.oracle.MsgAggregateExchangeRatePrevote": {},
	"/seiprotocol.sei.oracle.MsgAggregateExchangeRateVote":    {},
	"/seiprotocol.sei.oracle.MsgDelegateFeedConsent":          {},
	// 通用 oracle 模块消息类型
	"/oracle.MsgAggregateExchangeRatePrevote": {},
	"/oracle.MsgAggregateExchangeRateVote":    {},
	"/oracle.MsgDelegateFeedConsent":          {},
}

func IsTxPrioritized(tx sdk.Tx) bool {
	if tx == nil {
		return false
	}
	for _, msg := range tx.GetMsgs() {
		if isOracleMsg(msg) {
			return true
		}
		if exec, ok := msg.(*authz.MsgExec); ok {
			contains, err := containsOracleInAuthz(exec, 0)
			if err == nil && contains {
				return true
			}
		}
	}
	return false
}

func isOracleMsg(msg sdk.Msg) bool {
	typeURL := sdk.MsgTypeURL(msg)
	if _, ok := oracleMsgTypeURLs[typeURL]; ok {
		return true
	}
	lower := strings.ToLower(typeURL)
	return strings.Contains(lower, ".oracle.") || strings.Contains(lower, "/oracle.")
}

func containsOracleInAuthz(authzMsg *authz.MsgExec, nestedLvl int) (bool, error) {
	if nestedLvl >= maxNestedOracleMsgs {
		return false, nil
	}
	msgs, err := authzMsg.GetMessages()
	if err != nil {
		return false, err
	}
	for _, msg := range msgs {
		if isOracleMsg(msg) {
			return true, nil
		}
		if exec, ok := msg.(*authz.MsgExec); ok {
			found, err := containsOracleInAuthz(exec, nestedLvl+1)
			if err != nil {
				return false, err
			}
			if found {
				return true, nil
			}
		}
	}
	return false, nil
}
