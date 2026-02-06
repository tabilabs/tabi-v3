package utils

import (
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	proto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FakeTx implements sdk.Tx for testing
type FakeTx struct {
	msgs []sdk.Msg
}

func (f FakeTx) GetMsgs() []sdk.Msg { return f.msgs }

func (f FakeTx) ValidateBasic() error { return nil }

func (f FakeTx) GetGasEstimate() uint64 { return 0 }

func (f FakeTx) GetSigners() []sdk.AccAddress { return nil }

func (f FakeTx) GetPubKeys() ([]cryptotypes.PubKey, error) { return nil, nil }

func (f FakeTx) GetSignaturesV2() ([]signing.SignatureV2, error) { return nil, nil }

// =============================================================================
// Test Proto Msg: 用于让 sdk.MsgTypeURL 返回可控值
// =============================================================================

// testTabiOracleVoteMsg 是一个最小可用的 sdk.Msg + proto.Message。
// 通过 proto.RegisterType 注册其 message name，从而让 sdk.MsgTypeURL(msg)
// 返回我们期望的 oracle typeURL。
type testTabiOracleVoteMsg struct{}

func (*testTabiOracleVoteMsg) Reset()         {}
func (*testTabiOracleVoteMsg) String() string { return "testTabiOracleVoteMsg" }
func (*testTabiOracleVoteMsg) ProtoMessage()  {}

func (*testTabiOracleVoteMsg) Route() string { return "oracle" }
func (*testTabiOracleVoteMsg) Type() string  { return "oracle_vote" }

func (*testTabiOracleVoteMsg) ValidateBasic() error { return nil }
func (*testTabiOracleVoteMsg) GetSignBytes() []byte { return nil }
func (*testTabiOracleVoteMsg) GetSigners() []sdk.AccAddress {
	return nil
}

func init() {
	proto.RegisterType((*testTabiOracleVoteMsg)(nil), "tabilabs.tabi.oracle.MsgAggregateExchangeRateVote")
}

func newMsgExec(t *testing.T, msgs ...sdk.Msg) *authz.MsgExec {
	t.Helper()
	grantee := sdk.AccAddress(make([]byte, 20))
	exec := authz.NewMsgExec(grantee, msgs)
	return &exec
}

func newNestedMsgExec(t *testing.T, depth int, leaf sdk.Msg) *authz.MsgExec {
	t.Helper()
	exec := newMsgExec(t, leaf)
	for i := 0; i < depth; i++ {
		exec = newMsgExec(t, exec)
	}
	return exec
}

// =============================================================================
// IsTxPrioritized Tests - False Cases
// =============================================================================

func TestIsTxPrioritized_NilTx(t *testing.T) {
	result := IsTxPrioritized(nil)
	assert.False(t, result, "nil tx should return false")
}

func TestIsTxPrioritized_EmptyMsgs(t *testing.T) {
	tx := FakeTx{msgs: []sdk.Msg{}}
	result := IsTxPrioritized(tx)
	assert.False(t, result, "tx with empty msgs should return false")
}

func TestIsTxPrioritized_NonOracleMsg(t *testing.T) {
	msg := &banktypes.MsgSend{
		FromAddress: "tabi1abc",
		ToAddress:   "tabi1def",
	}
	tx := FakeTx{msgs: []sdk.Msg{msg}}
	result := IsTxPrioritized(tx)
	assert.False(t, result, "non-oracle msg should return false")
}

// =============================================================================
// IsTxPrioritized Tests - True Cases (核心场景)
// =============================================================================

func TestIsTxPrioritized_OracleMsg_Direct(t *testing.T) {
	msg := &testTabiOracleVoteMsg{}
	assert.Equal(t, "/tabilabs.tabi.oracle.MsgAggregateExchangeRateVote", sdk.MsgTypeURL(msg))

	tx := FakeTx{msgs: []sdk.Msg{msg}}
	assert.True(t, IsTxPrioritized(tx), "direct oracle msg should be prioritized")
}

func TestIsTxPrioritized_OracleMsg_AuthzExec(t *testing.T) {
	exec := newMsgExec(t, &testTabiOracleVoteMsg{})

	tx := FakeTx{msgs: []sdk.Msg{exec}}
	assert.True(t, IsTxPrioritized(tx), "oracle msg nested in authz.MsgExec should be prioritized")
}

// =============================================================================
// containsOracleInAuthz Tests - 深度限制和嵌套场景
// =============================================================================

func TestContainsOracleInAuthz_NestedWithinLimit(t *testing.T) {
	// depth=4: nestedLvl 依次为 0..4，仍可到达 leaf 并返回 true
	exec := newNestedMsgExec(t, 4, &testTabiOracleVoteMsg{})

	result, err := containsOracleInAuthz(exec, 0)
	require.NoError(t, err)
	assert.True(t, result, "oracle msg should be discoverable within depth limit")
}

func TestContainsOracleInAuthz_DepthLimit(t *testing.T) {
	// depth=5: 将在 nestedLvl==5 处被截断，哪怕 leaf 是 oracle 也返回 false
	exec := newNestedMsgExec(t, 5, &testTabiOracleVoteMsg{})

	result, err := containsOracleInAuthz(exec, 0)
	require.NoError(t, err)
	assert.False(t, result, "oracle msg beyond depth limit should not be discovered")
}

func TestIsTxPrioritized_AuthzDepthLimitStops(t *testing.T) {
	exec := newNestedMsgExec(t, 5, &testTabiOracleVoteMsg{})
	tx := FakeTx{msgs: []sdk.Msg{exec}}
	assert.False(t, IsTxPrioritized(tx), "depth limit should stop oracle discovery through nested authz")
}

// =============================================================================
// isOracleTypeURL Tests - 直接验证生产 helper
// =============================================================================

func TestIsOracleTypeURL(t *testing.T) {
	testCases := []struct {
		name     string
		typeURL  string
		expected bool
	}{
		// Tabi oracle URLs (map)
		{"tabi oracle prevote", "/tabilabs.tabi.oracle.MsgAggregateExchangeRatePrevote", true},
		{"tabi oracle vote", "/tabilabs.tabi.oracle.MsgAggregateExchangeRateVote", true},
		{"tabi oracle delegate", "/tabilabs.tabi.oracle.MsgDelegateFeedConsent", true},

		// Generic oracle URLs (map)
		{"generic oracle prevote", "/oracle.MsgAggregateExchangeRatePrevote", true},
		{"generic oracle vote", "/oracle.MsgAggregateExchangeRateVote", true},
		{"generic oracle delegate", "/oracle.MsgDelegateFeedConsent", true},

		// Pattern matches (.oracle. or /oracle.)
		{"contains .oracle. pattern", "/somemodule.oracle.SomeMsg", true},
		{"contains /oracle. pattern", "/oracle.SomeOtherMsg", true},
		{"uppercase ORACLE via ToLower", "/somemodule.ORACLE.SomeMsg", true},

		// Legacy Sei URL (not in map, but still matches pattern)
		{"sei oracle still matches pattern", "/seiprotocol.sei.oracle.MsgAggregateExchangeRatePrevote", true},

		// Non-oracle
		{"bank send", "/cosmos.bank.v1beta1.MsgSend", false},
		{"evm transaction", "/tabilabs.tabiv2.evm.MsgEVMTransaction", false},
		{"staking delegate", "/cosmos.staking.v1beta1.MsgDelegate", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isOracleTypeURL(tc.typeURL))
		})
	}
}

// =============================================================================
// oracleMsgTypeURLs Map Tests - 验证修改后的 URL 正确
// =============================================================================

func TestOracleMsgTypeURLs_ContainsExpectedURLs(t *testing.T) {
	expectedURLs := []string{
		"/tabilabs.tabi.oracle.MsgAggregateExchangeRatePrevote",
		"/tabilabs.tabi.oracle.MsgAggregateExchangeRateVote",
		"/tabilabs.tabi.oracle.MsgDelegateFeedConsent",
		"/oracle.MsgAggregateExchangeRatePrevote",
		"/oracle.MsgAggregateExchangeRateVote",
		"/oracle.MsgDelegateFeedConsent",
	}

	// 验证预期的 URL 都存在
	for _, url := range expectedURLs {
		_, exists := oracleMsgTypeURLs[url]
		assert.True(t, exists, "oracleMsgTypeURLs should contain: %s", url)
	}

	// 验证 map 大小正确（没有多余的条目）
	assert.Equal(t, len(expectedURLs), len(oracleMsgTypeURLs),
		"oracleMsgTypeURLs should have exactly %d entries", len(expectedURLs))
}

func TestOracleMsgTypeURLs_DoesNotContainOldSeiURLs(t *testing.T) {
	oldURLs := []string{
		"/seiprotocol.sei.oracle.MsgAggregateExchangeRatePrevote",
		"/seiprotocol.sei.oracle.MsgAggregateExchangeRateVote",
		"/seiprotocol.sei.oracle.MsgDelegateFeedConsent",
	}

	for _, url := range oldURLs {
		_, exists := oracleMsgTypeURLs[url]
		assert.False(t, exists, "oracleMsgTypeURLs should NOT contain old sei URL: %s", url)
	}
}

// TestOldSeiURLsStillMatchViaPattern 验证旧的 sei URL 仍然可以通过 pattern 匹配
// 这确保了向后兼容性
func TestOldSeiURLsStillMatchViaPattern(t *testing.T) {
	oldURLs := []string{
		"/seiprotocol.sei.oracle.MsgAggregateExchangeRatePrevote",
		"/seiprotocol.sei.oracle.MsgAggregateExchangeRateVote",
		"/seiprotocol.sei.oracle.MsgDelegateFeedConsent",
	}

	for _, url := range oldURLs {
		assert.True(t, isOracleTypeURL(url), "old sei URL should still match via pattern: %s", url)
	}
}

// =============================================================================
// maxNestedOracleMsgs Constant Test
// =============================================================================

func TestMaxNestedOracleMsgs_Constant(t *testing.T) {
	assert.Equal(t, 5, maxNestedOracleMsgs, "maxNestedOracleMsgs should be 5")
}

// =============================================================================
// Regression Tests - 验证修改不会破坏现有功能
// =============================================================================

func TestIsOracleMsg_ToLowerBehavior(t *testing.T) {
	// 验证 ToLower 在 pattern 匹配中生效
	testCases := []struct {
		typeURL  string
		expected bool
	}{
		{"/module.oracle.Msg", true},
		{"/module.ORACLE.Msg", true}, // 大写应该匹配
		{"/module.Oracle.Msg", true}, // 混合大小写应该匹配
		{"/MODULE.ORACLE.MSG", true}, // 全大写应该匹配
		{"/oracle.Msg", true},
		{"/ORACLE.Msg", true},
		{"/Oracle.Msg", true},
	}

	for _, tc := range testCases {
		t.Run(tc.typeURL, func(t *testing.T) {
			assert.Equal(t, tc.expected, isOracleTypeURL(tc.typeURL))
		})
	}
}
