package utils

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	aclsdktypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	acltypes "github.com/cosmos/cosmos-sdk/x/accesscontrol/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	feegranttypes "github.com/cosmos/cosmos-sdk/x/feegrant"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	evmtypes "github.com/tabilabs/tabi-v3/x/evm/types"
)

const (
	DefaultIDTemplate = "*"
)

var StoreKeyToResourceTypePrefixMap = aclsdktypes.StoreKeyToResourceTypePrefixMap{
	aclsdktypes.ParentNodeKey: {
		aclsdktypes.ResourceType_ANY: aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV:  aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_Mem: aclsdktypes.EmptyPrefix,
		// NOTE: this chain does not include an epoch module, but the accesscontrol
		// resource tree defines KV_EPOCH. Map it to the parent node as a safe
		// fallback to keep resource mappings exhaustive.
		aclsdktypes.ResourceType_KV_EPOCH: aclsdktypes.EmptyPrefix,
		// NOTE: this chain does not include oracle/tokenfactory modules, but the
		// accesscontrol resource tree defines these resource types. Map them to the
		// parent node as a safe fallback to keep resource mappings exhaustive.
		aclsdktypes.ResourceType_KV_ORACLE:                      aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_ORACLE_AGGREGATE_VOTES:      aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_ORACLE_VOTE_TARGETS:         aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_ORACLE_FEEDERS:              aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_ORACLE_EXCHANGE_RATE:        aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_ORACLE_VOTE_PENALTY_COUNTER: aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_ORACLE_PRICE_SNAPSHOT:       aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_TOKENFACTORY:                aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_TOKENFACTORY_DENOM:          aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_TOKENFACTORY_METADATA:       aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_TOKENFACTORY_ADMIN:          aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_TOKENFACTORY_CREATOR:        aclsdktypes.EmptyPrefix,
	},
	banktypes.StoreKey: {
		aclsdktypes.ResourceType_KV_BANK:             aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_BANK_BALANCES:    banktypes.BalancesPrefix,
		aclsdktypes.ResourceType_KV_BANK_SUPPLY:      banktypes.SupplyKey,
		aclsdktypes.ResourceType_KV_BANK_DENOM:       banktypes.DenomMetadataPrefix,
		aclsdktypes.ResourceType_KV_BANK_WEI_BALANCE: banktypes.WeiBalancesPrefix,
	},
	banktypes.DeferredCacheStoreKey: {
		aclsdktypes.ResourceType_KV_BANK_DEFERRED:                 aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_BANK_DEFERRED_MODULE_TX_INDEX: banktypes.DeferredCachePrefix,
	},
	authtypes.StoreKey: {
		aclsdktypes.ResourceType_KV_AUTH:                       aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_AUTH_ADDRESS_STORE:         authtypes.AddressStoreKeyPrefix,
		aclsdktypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER: authtypes.GlobalAccountNumberKey,
	},
	authztypes.StoreKey: {
		aclsdktypes.ResourceType_KV_AUTHZ: aclsdktypes.EmptyPrefix,
	},
	distributiontypes.StoreKey: {
		aclsdktypes.ResourceType_KV_DISTRIBUTION:                         aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_FEE_POOL:                distributiontypes.FeePoolKey,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_PROPOSER_KEY:            distributiontypes.ProposerKey,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_OUTSTANDING_REWARDS:     distributiontypes.ValidatorOutstandingRewardsPrefix,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_WITHDRAW_ADDR: distributiontypes.DelegatorWithdrawAddrPrefix,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_STARTING_INFO: distributiontypes.DelegatorStartingInfoPrefix,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_VAL_HISTORICAL_REWARDS:  distributiontypes.ValidatorHistoricalRewardsPrefix,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_VAL_CURRENT_REWARDS:     distributiontypes.ValidatorCurrentRewardsPrefix,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_VAL_ACCUM_COMMISSION:    distributiontypes.ValidatorAccumulatedCommissionPrefix,
		aclsdktypes.ResourceType_KV_DISTRIBUTION_SLASH_EVENT:             distributiontypes.ValidatorSlashEventPrefix,
	},
	feegranttypes.StoreKey: {
		aclsdktypes.ResourceType_KV_FEEGRANT:           aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_FEEGRANT_ALLOWANCE: feegranttypes.FeeAllowanceKeyPrefix,
	},
	stakingtypes.StoreKey: {
		aclsdktypes.ResourceType_KV_STAKING:                          aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_STAKING_VALIDATION_POWER:         stakingtypes.LastValidatorPowerKey,
		aclsdktypes.ResourceType_KV_STAKING_TOTAL_POWER:              stakingtypes.LastTotalPowerKey,
		aclsdktypes.ResourceType_KV_STAKING_VALIDATOR:                stakingtypes.ValidatorsKey,
		aclsdktypes.ResourceType_KV_STAKING_VALIDATORS_CON_ADDR:      stakingtypes.ValidatorsByConsAddrKey,
		aclsdktypes.ResourceType_KV_STAKING_VALIDATORS_BY_POWER:      stakingtypes.ValidatorsByPowerIndexKey,
		aclsdktypes.ResourceType_KV_STAKING_DELEGATION:               stakingtypes.DelegationKey,
		aclsdktypes.ResourceType_KV_STAKING_UNBONDING_DELEGATION:     stakingtypes.UnbondingDelegationKey,
		aclsdktypes.ResourceType_KV_STAKING_UNBONDING_DELEGATION_VAL: stakingtypes.UnbondingDelegationByValIndexKey,
		aclsdktypes.ResourceType_KV_STAKING_REDELEGATION:             stakingtypes.RedelegationKey,
		aclsdktypes.ResourceType_KV_STAKING_REDELEGATION_VAL_SRC:     stakingtypes.RedelegationByValSrcIndexKey,
		aclsdktypes.ResourceType_KV_STAKING_REDELEGATION_VAL_DST:     stakingtypes.RedelegationByValDstIndexKey,
		aclsdktypes.ResourceType_KV_STAKING_UNBONDING:                stakingtypes.UnbondingQueueKey,
		aclsdktypes.ResourceType_KV_STAKING_REDELEGATION_QUEUE:       stakingtypes.RedelegationQueueKey,
		aclsdktypes.ResourceType_KV_STAKING_VALIDATOR_QUEUE:          stakingtypes.ValidatorQueueKey,
		aclsdktypes.ResourceType_KV_STAKING_HISTORICAL_INFO:          stakingtypes.HistoricalInfoKey,
	},
	slashingtypes.StoreKey: {
		aclsdktypes.ResourceType_KV_SLASHING:                          aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_SLASHING_VAL_SIGNING_INFO:         slashingtypes.ValidatorSigningInfoKeyPrefix,
		aclsdktypes.ResourceType_KV_SLASHING_ADDR_PUBKEY_RELATION_KEY: slashingtypes.AddrPubkeyRelationKeyPrefix,
	},

	acltypes.StoreKey: {
		aclsdktypes.ResourceType_KV_ACCESSCONTROL:                         aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_ACCESSCONTROL_WASM_DEPENDENCY_MAPPING: acltypes.GetWasmMappingKey(),
	},
	wasmtypes.StoreKey: {
		aclsdktypes.ResourceType_KV_WASM:                       aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_WASM_CODE:                  wasmtypes.CodeKeyPrefix,
		aclsdktypes.ResourceType_KV_WASM_CONTRACT_ADDRESS:      wasmtypes.ContractKeyPrefix,
		aclsdktypes.ResourceType_KV_WASM_CONTRACT_STORE:        wasmtypes.ContractStorePrefix,
		aclsdktypes.ResourceType_KV_WASM_SEQUENCE_KEY:          wasmtypes.SequenceKeyPrefix,
		aclsdktypes.ResourceType_KV_WASM_CONTRACT_CODE_HISTORY: wasmtypes.ContractCodeHistoryElementPrefix,
		aclsdktypes.ResourceType_KV_WASM_CONTRACT_BY_CODE_ID:   wasmtypes.ContractByCodeIDAndCreatedSecondaryIndexPrefix,
		aclsdktypes.ResourceType_KV_WASM_PINNED_CODE_INDEX:     wasmtypes.PinnedCodeIndexPrefix,
	},
	evmtypes.StoreKey: {
		aclsdktypes.ResourceType_KV_EVM:                   aclsdktypes.EmptyPrefix,
		aclsdktypes.ResourceType_KV_EVM_BALANCE:           aclsdktypes.EmptyPrefix, // EVM_BALANCE is deprecated and not used anymore
		aclsdktypes.ResourceType_KV_EVM_TRANSIENT:         evmtypes.TransientStateKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_ACCOUNT_TRANSIENT: evmtypes.AccountTransientStateKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_MODULE_TRANSIENT:  evmtypes.TransientModuleStateKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_NONCE:             evmtypes.NonceKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_RECEIPT:           evmtypes.ReceiptKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_S2E:               evmtypes.TabiAddressToEVMAddressKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_E2S:               evmtypes.EVMAddressToTabiAddressKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_CODE_HASH:         evmtypes.CodeHashKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_CODE:              evmtypes.CodeKeyPrefix,
		aclsdktypes.ResourceType_KV_EVM_CODE_SIZE:         evmtypes.CodeSizeKeyPrefix,
	},
}

// ResourceTypeToStoreKeyMap this maps between resource types and their respective storekey
var ResourceTypeToStoreKeyMap = aclsdktypes.ResourceTypeToStoreKeyMap{
	// ANY, KV, and MEM are intentionally excluded because they don't map to a specific store key
	// ~~~~ BANK Resource Types ~~~~
	aclsdktypes.ResourceType_KV_BANK:             banktypes.StoreKey,
	aclsdktypes.ResourceType_KV_BANK_BALANCES:    banktypes.StoreKey,
	aclsdktypes.ResourceType_KV_BANK_SUPPLY:      banktypes.StoreKey,
	aclsdktypes.ResourceType_KV_BANK_DENOM:       banktypes.StoreKey,
	aclsdktypes.ResourceType_KV_BANK_WEI_BALANCE: banktypes.StoreKey,

	// ~~~~ BANK DEFERRED Resource Types ~~~~
	aclsdktypes.ResourceType_KV_BANK_DEFERRED:                 banktypes.DeferredCacheStoreKey,
	aclsdktypes.ResourceType_KV_BANK_DEFERRED_MODULE_TX_INDEX: banktypes.DeferredCacheStoreKey,

	// ~~~~ AUTH Resource Types ~~~~
	aclsdktypes.ResourceType_KV_AUTH:                       authtypes.StoreKey,
	aclsdktypes.ResourceType_KV_AUTH_ADDRESS_STORE:         authtypes.StoreKey,
	aclsdktypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER: authtypes.StoreKey,

	// ~~~~ AUTHZ Resource Types ~~~~
	aclsdktypes.ResourceType_KV_AUTHZ: authztypes.StoreKey,

	// ~~~~ DISTRIBUTION Resource Types ~~~~
	aclsdktypes.ResourceType_KV_DISTRIBUTION:                         distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_FEE_POOL:                distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_PROPOSER_KEY:            distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_OUTSTANDING_REWARDS:     distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_WITHDRAW_ADDR: distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_STARTING_INFO: distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_VAL_HISTORICAL_REWARDS:  distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_VAL_CURRENT_REWARDS:     distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_VAL_ACCUM_COMMISSION:    distributiontypes.StoreKey,
	aclsdktypes.ResourceType_KV_DISTRIBUTION_SLASH_EVENT:             distributiontypes.StoreKey,

	// ~~~~ FEEGRANT Resource Types ~~~~
	aclsdktypes.ResourceType_KV_FEEGRANT:           feegranttypes.StoreKey,
	aclsdktypes.ResourceType_KV_FEEGRANT_ALLOWANCE: feegranttypes.StoreKey,

	// ~~~~ STAKING Resource Types ~~~~
	aclsdktypes.ResourceType_KV_STAKING:                          stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_VALIDATION_POWER:         stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_TOTAL_POWER:              stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_VALIDATOR:                stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_VALIDATORS_CON_ADDR:      stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_VALIDATORS_BY_POWER:      stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_DELEGATION:               stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_UNBONDING_DELEGATION:     stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_UNBONDING_DELEGATION_VAL: stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_REDELEGATION:             stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_REDELEGATION_VAL_SRC:     stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_REDELEGATION_VAL_DST:     stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_UNBONDING:                stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_REDELEGATION_QUEUE:       stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_VALIDATOR_QUEUE:          stakingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_STAKING_HISTORICAL_INFO:          stakingtypes.StoreKey,

	// ~~~~ SLASHING Resource Types ~~~~
	aclsdktypes.ResourceType_KV_SLASHING:                          slashingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_SLASHING_VAL_SIGNING_INFO:         slashingtypes.StoreKey,
	aclsdktypes.ResourceType_KV_SLASHING_ADDR_PUBKEY_RELATION_KEY: slashingtypes.StoreKey,

	// ~~~~ EPOCH Resource Types ~~~~
	aclsdktypes.ResourceType_KV_EPOCH: aclsdktypes.ParentNodeKey,

	// ~~~~ ORACLE Resource Types ~~~~
	aclsdktypes.ResourceType_KV_ORACLE:                      aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_ORACLE_AGGREGATE_VOTES:      aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_ORACLE_VOTE_TARGETS:         aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_ORACLE_FEEDERS:              aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_ORACLE_EXCHANGE_RATE:        aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_ORACLE_VOTE_PENALTY_COUNTER: aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_ORACLE_PRICE_SNAPSHOT:       aclsdktypes.ParentNodeKey,

	// ~~~~ TOKENFACTORY Resource Types ~~~~
	aclsdktypes.ResourceType_KV_TOKENFACTORY:          aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_TOKENFACTORY_DENOM:    aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_TOKENFACTORY_METADATA: aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_TOKENFACTORY_ADMIN:    aclsdktypes.ParentNodeKey,
	aclsdktypes.ResourceType_KV_TOKENFACTORY_CREATOR:  aclsdktypes.ParentNodeKey,

	// ~~~~ ACCESSCONTROL Resource Types ~~~~
	aclsdktypes.ResourceType_KV_ACCESSCONTROL:                         acltypes.StoreKey,
	aclsdktypes.ResourceType_KV_ACCESSCONTROL_WASM_DEPENDENCY_MAPPING: acltypes.StoreKey,

	// ~~~~ WASM Resource Types ~~~~
	aclsdktypes.ResourceType_KV_WASM:                       wasmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_WASM_CODE:                  wasmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_WASM_CONTRACT_ADDRESS:      wasmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_WASM_CONTRACT_STORE:        wasmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_WASM_SEQUENCE_KEY:          wasmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_WASM_CONTRACT_CODE_HISTORY: wasmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_WASM_CONTRACT_BY_CODE_ID:   wasmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_WASM_PINNED_CODE_INDEX:     wasmtypes.StoreKey,

	// ~~~~ EVM Resource Types ~~~~
	aclsdktypes.ResourceType_KV_EVM:                   evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_BALANCE:           evmtypes.StoreKey, // EVM_BALANCE is deprecated and not used anymore
	aclsdktypes.ResourceType_KV_EVM_TRANSIENT:         evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_ACCOUNT_TRANSIENT: evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_MODULE_TRANSIENT:  evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_NONCE:             evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_RECEIPT:           evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_S2E:               evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_E2S:               evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_CODE_HASH:         evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_CODE:              evmtypes.StoreKey,
	aclsdktypes.ResourceType_KV_EVM_CODE_SIZE:         evmtypes.StoreKey,
}
