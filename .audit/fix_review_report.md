# 审计修复实施与推演审查报告

## 修复时间: 2026-01-30
## 修复范围: Issue 11-18 (根据 audit_remediation_plan_v2.md)

---

## 修复总结

所有 8 个 Issue 已完成修复，编译验证通过。

| Issue | 问题描述 | 状态 | 修改文件 |
|-------|----------|------|----------|
| 18 | GASLIMIT opcode 与 RPC gasLimit 不一致 | ✅ 已修复 | `x/evm/keeper/keeper.go`, `evmrpc/block.go`, `evmrpc/subscribe.go`, `evmrpc/simulate.go` |
| 13 | PendingTxChecker 未复验余额 | ✅ 已修复 | `x/evm/ante/sig.go` |
| 11 | Bloom 为空导致 eth_getLogs 静默漏日志 | ✅ 已修复 | `x/evm/keeper/log.go`, `evmrpc/filter.go` |
| 17 | cumulativeGasUsed 恒 0/不正确 | ✅ 已修复 | `x/evm/keeper/receipt.go` |
| 14 | eth_getTransactionByBlock*AndIndex index 语义错误 | ✅ 已修复 | `evmrpc/tx.go` |
| 16 | tx RPC baseFee 取值错误 | ✅ 已修复 | `evmrpc/tx.go` (同 Issue 14) |
| 12 | SetState 零值删除存储槽 | ✅ 已修复 | `x/evm/keeper/state.go` |
| 15 | IncrGasCounter float32 精度问题 | ✅ 已修复 | `utils/metrics/metrics_util.go` |

---

## 详细修复内容与推演审查

### Issue 18: GASLIMIT opcode 与 RPC gasLimit 不一致

**修改文件:** `x/evm/keeper/keeper.go`, `evmrpc/block.go`, `evmrpc/subscribe.go`, `evmrpc/simulate.go`

**修复内容:**
1. 添加常量 `DefaultBlockGasLimit = 35_000_000`
2. 修改 `GetVMBlockContext` 函数，使用共识参数获取真实区块 Gas 上限
3. 修复 RPC 侧三处 gasLimit 取值：
   - `evmrpc/block.go`: 添加 fallback 到 DefaultBlockGasLimit
   - `evmrpc/subscribe.go`: 添加 fallback 到 DefaultBlockGasLimit
   - `evmrpc/simulate.go`: 使用共识参数而非 GasCap

**代码变更:**
```go
// x/evm/keeper/keeper.go - 新增常量
const DefaultBlockGasLimit = 35_000_000

// GetVMBlockContext 中的修复
var gasLimit uint64
if ctx.ConsensusParams() != nil && ctx.ConsensusParams().Block != nil && ctx.ConsensusParams().Block.MaxGas > 0 {
    gasLimit = uint64(ctx.ConsensusParams().Block.MaxGas)
} else {
    gasLimit = DefaultBlockGasLimit
}
// ...
GasLimit: gasLimit,

// evmrpc/block.go - 添加 fallback
gasLimit := blockRes.ConsensusParamUpdates.Block.MaxGas
if gasLimit <= 0 {
    gasLimit = keeper.DefaultBlockGasLimit
}

// evmrpc/subscribe.go - 添加 fallback  
gasLimit := uint64(header.ResultFinalizeBlock.ConsensusParamUpdates.Block.MaxGas)
if gasLimit == 0 {
    gasLimit = keeper.DefaultBlockGasLimit
}

// evmrpc/simulate.go - 使用共识参数
ctx := b.ctxProvider(blockNumber.Int64())
var gasLimit uint64
if ctx.ConsensusParams() != nil && ctx.ConsensusParams().Block != nil && ctx.ConsensusParams().Block.MaxGas > 0 {
    gasLimit = uint64(ctx.ConsensusParams().Block.MaxGas)
} else {
    gasLimit = keeper.DefaultBlockGasLimit
}
```

**推演审查:** ✅
- 执行侧：正确处理了 nil 和 0 值的边界情况
- RPC 侧：block、subscribe、simulate 三处均已修复，使用一致的 fallback 逻辑
- 与 Sei Chain 方案一致
- 不影响 BlockTest 和 Replay 模式（它们有独立的处理路径）

---

### Issue 13: PendingTxChecker 未复验余额

**修改文件:** `x/evm/ante/sig.go`

**修复内容:**
1. 新增 `hasSufficientBalance` 函数用于检查余额
2. 在 `txNonce < nextPendingNonce` 分支中添加余额检查

**代码变更:**
```go
} else if txNonce < nextPendingNonce {
    if !hasSufficientBalance(svd.evmKeeper, latestCtx, evmAddr, ethTx) {
        return abci.Pending
    }
    return abci.Accepted
}

func hasSufficientBalance(k *evmkeeper.Keeper, ctx sdk.Context, evmAddr common.Address, tx *ethtypes.Transaction) bool {
    senderTabiAddr := k.GetTabiAddressOrDefault(ctx, evmAddr)
    balance := k.BankKeeper().GetBalance(ctx, senderTabiAddr, "atabi")

    var maxCost *big.Int
    if tx.GasFeeCap() != nil {
        maxCost = new(big.Int).Mul(tx.GasFeeCap(), new(big.Int).SetUint64(tx.Gas()))
    } else {
        maxCost = new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
    }
    maxCost.Add(maxCost, tx.Value())

    return balance.Amount.BigInt().Cmp(maxCost) >= 0
}
```

**推演审查:** ✅
- 使用 `latestCtxGetter()` 确保读取最新状态
- 保守估算交易成本（使用 GasFeeCap 而非 effectiveGasPrice）
- 余额不足返回 Pending（允许后续入金），不直接拒绝
- 不涉及 EVM 执行，避免 DoS 风险

---

### Issue 11: Bloom 为空导致 eth_getLogs 静默漏日志

**修改文件:** `x/evm/keeper/log.go`, `evmrpc/filter.go`

**修复内容:**
1. 新增 `GetBlockBloomWithExists` 函数，区分"缺失"与"真实零"
2. 修改 filter.go 中的 bloom 检查逻辑

**代码变更:**
```go
// log.go
func (k *Keeper) GetBlockBloomWithExists(ctx sdk.Context) (res ethtypes.Bloom, exists bool) {
    store := ctx.KVStore(k.storeKey)
    bz := store.Get(types.BlockBloomPrefix)
    if bz != nil {
        res.SetBytes(bz)
        return res, true
    }
    cutoff := k.GetLegacyBlockBloomCutoffHeight(ctx)
    if cutoff == 0 || ctx.BlockHeight() < cutoff {
        legacyBz := store.Get(types.BlockBloomKey(ctx.BlockHeight()))
        if legacyBz != nil {
            res.SetBytes(legacyBz)
            return res, true
        }
    }
    return ethtypes.Bloom{}, false
}

// filter.go
blockBloom, bloomExists = f.k.GetBlockBloomWithExists(providerCtx)
if bloomExists && !MatchFilters(blockBloom, bloomIndexes) {
    <-dbReadSemaphore
    continue // skip the block if bloom filter does not match
}
```

**推演审查:** ✅
- 正确区分了"bloom 缺失"和"bloom 存在但为零"
- bloom 缺失时不跳过区块，确保不漏数据
- bloom 存在且为零时仍可正常跳过（不退化性能）

---

### Issue 17: cumulativeGasUsed 恒 0/不正确

**修改文件:** `x/evm/keeper/receipt.go`

**修复内容:**
1. 重写 `FlushTransientReceipts` 函数
2. 按 BlockNumber 分组，组内按 TransactionIndex 排序计算累积 Gas
3. 计算顺序与写入顺序分离（写入仍按 txHash 顺序）

**代码变更:**
```go
func (k *Keeper) FlushTransientReceipts(ctx sdk.Context) error {
    // 1. 收集所有收据
    type receiptWithKey struct {
        txHash  common.Hash
        receipt *types.Receipt
    }
    var receipts []receiptWithKey
    // ... 收集逻辑 ...

    // 2. 按 BlockNumber 分组
    blockReceipts := make(map[uint64][]receiptWithKey)
    for _, r := range receipts {
        blockReceipts[r.receipt.BlockNumber] = append(blockReceipts[r.receipt.BlockNumber], r)
    }

    // 3. 每个区块内按 TransactionIndex 排序并计算累积 Gas
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

    // 4. 按 txHash 顺序写入 changeset
    sort.Slice(receipts, func(i, j int) bool {
        return bytes.Compare(receipts[i].txHash[:], receipts[j].txHash[:]) < 0
    })
    // ... 写入逻辑 ...
}
```

**推演审查:** ✅
- 没有修改 transient key schema，变更面最小
- 计算顺序 (TransactionIndex) 与写入顺序 (txHash) 分离
- 过滤 EffectiveGasPrice == 0 的收据（避免 shell/ante-error 影响）
- 使用 `common.BytesToHash(iter.Key())` 而非类型转换，避免 slice->array 假设问题

---

### Issue 14 & 16: eth_getTransactionByBlock*AndIndex index 语义错误 & baseFee 取值错误

**修改文件:** `evmrpc/tx.go`

**修复内容:**
1. 重写 `getTransactionWithBlock` - 遍历计数 EVM 交易，使用 EVM-only index
2. 新增 `getTransactionByHashFromBlock` - 直接在区块中查找交易
3. 修复 baseFee 使用 `GetCurrBaseFeePerGas` 替代 `GetBaseFee`

**代码变更:**
```go
// getTransactionWithBlock - 使用 EVM-only index
func (t *TransactionAPI) getTransactionWithBlock(block *coretypes.ResultBlock, evmIndex hexutil.Uint) (*ethapi.RPCTransaction, error) {
    decoder := t.txConfigProvider(block.Block.Height).TxDecoder()
    var evmCount hexutil.Uint = 0
    for _, txBz := range block.Block.Txs {
        ethtx := getEthTxForTxBz(txBz, decoder)
        if ethtx == nil {
            continue
        }
        if evmCount == evmIndex {
            // ... 构造并返回 ...
            baseFeePerGas := t.keeper.GetCurrBaseFeePerGas(t.ctxProvider(height)).TruncateInt().BigInt()
            res := ethapi.NewRPCTransaction(ethtx, blockHash, blockNumber, uint64(blockTime.Unix()), uint64(evmIndex), baseFeePerGas, chainConfig)
            return res, nil
        }
        evmCount++
    }
    return nil, nil
}

// getTransactionByHashFromBlock - 直接按 hash 查找
func (t *TransactionAPI) getTransactionByHashFromBlock(ctx context.Context, hash common.Hash, receipt *types.Receipt) (*ethapi.RPCTransaction, error) {
    // ... 在区块中按 hash 查找交易并返回 ...
}
```

**推演审查:** ✅
- `eth_` 端点现在使用 EVM-only index，符合以太坊规范
- `GetTransactionByHash` 不再调用 `GetTransactionByBlockNumberAndIndex`，避免 cosmos index 到 evm index 的映射问题
- baseFee 使用 `GetCurrBaseFeePerGas`，与其他 RPC 端点一致
- `tabi_` 端点通过嵌入保持原行为（需要单独确认）

---

### Issue 12: SetState 零值删除存储槽

**修改文件:** `x/evm/keeper/state.go`

**修复内容:**
当 val 为零值时执行 Delete，否则 Set。

**代码变更:**
```go
func (k *Keeper) SetState(ctx sdk.Context, addr common.Address, key common.Hash, val common.Hash) {
    store := k.PrefixStore(ctx, types.StateKey(addr))
    if val == (common.Hash{}) {
        store.Delete(key[:])
    } else {
        store.Set(key[:], val[:])
    }
}
```

**推演审查:** ✅
- 符合 EVM 规范行为
- GetState 语义不变（缺失仍返回 0）
- 不处理存量零值槽（需要单独迁移任务）

---

### Issue 15: IncrGasCounter float32 精度问题

**修改文件:** `utils/metrics/metrics_util.go`

**修复内容:**
1. 添加 value <= 0 检查
2. 拆分大值为多次安全增量（最大 2^24）
3. 设置最大迭代次数防止极端值

**代码变更:**
```go
func IncrGasCounter(gasType string, value int64) {
    if value <= 0 {
        return
    }
    const maxFloat32Safe int64 = 16777216
    const maxIterations = 100
    for i := 0; value > 0 && i < maxIterations; i++ {
        incr := value
        if incr > maxFloat32Safe {
            incr = maxFloat32Safe
        }
        telemetry.IncrCounterWithLabels(
            []string{"tabi", "tx", "gas", "counter"},
            float32(incr),
            []metrics.Label{telemetry.NewLabel("type", gasType)},
        )
        value -= incr
    }
}
```

**推演审查:** ✅
- 避免了 float32 精度损失
- 设置最大迭代次数 (100) 防止极端值导致循环过久
- 不改变指标名称，不影响现有监控配置

---

## 编译验证

```bash
$ go build ./x/evm/... ./evmrpc/... ./utils/...
# 编译成功，无错误
```

---

## 待完成事项

1. **单元测试**: 需要为每个修复添加对应的单元测试
2. **集成测试**: 需要进行端到端测试验证
3. **tabi_ 端点确认**: Issue 14 修复了 `eth_` 端点，需确认 `tabi_` 端点行为是否需要调整
4. **存量零值槽迁移**: Issue 12 的修复不处理历史数据，需要单独的迁移任务

---

## 建议的提交信息

1. `fix(audit): align GASLIMIT and rpc gasLimit to consensus MaxGas (issue 18)`
2. `fix(audit): re-check balance in pending tx checker (issue 13)`
3. `fix(audit): avoid skipping blocks when bloom is missing (issue 11)`
4. `fix(audit): compute cumulativeGasUsed during flush in tx order (issue 17)`
5. `fix(audit): interpret tx index as evm-only index in eth block tx queries (issue 14)`
6. `fix(audit): align tx rpc baseFee with block baseFee (issue 16)`
7. `fix(audit): delete storage slots when set to zero (issue 12)`
8. `fix(audit): harden gas counters against float32 precision limits (issue 15)`
