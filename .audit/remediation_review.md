# 修复方案推演审查报告

## 审查日期: 2026-01-30
## 审查对象: `.audit/audit_remediation_plan_v2.md`

---

## 总体评估

修复方案整体设计合理，优先级排序符合安全原则（EVM 语义一致性 > DoS 风险 > 静默数据丢失）。以下是对每个 Issue 修复方案的详细推演审查。

---

## Commit 1 - Issue 18：GASLIMIT opcode 与 RPC gasLimit 不一致

### 方案评估：✅ 合理

**当前代码确认：**
- `x/evm/keeper/keeper.go:273`: `GasLimit: gp.Gas()` - 确实使用 Gas 池剩余量

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| 使用 `ctx.ConsensusParams().Block.MaxGas` | ✅ 正确 | 与 Sei Chain 方案一致 |
| nil/0 fallback 常量 35,000,000 | ⚠️ 建议 | 应定义常量 `DefaultBlockGasLimit` |
| RPC block 改用 TM RPC 获取 | ✅ 合理 | 需要有降级策略 |

**潜在风险：**
1. `ctx.ConsensusParams()` 可能返回 nil（测试/特殊场景），必须有 fallback
2. 需要确保 `getBlockTestBlockCtx` 和 `getReplayBlockCtx` 不受影响（它们已经使用 `header.GasLimit`）

**修复建议代码：**
```go
// 建议在 GetVMBlockContext 中替换为：
var gasLimit uint64
if ctx.ConsensusParams() != nil && ctx.ConsensusParams().Block != nil && ctx.ConsensusParams().Block.MaxGas > 0 {
    gasLimit = uint64(ctx.ConsensusParams().Block.MaxGas)
} else {
    gasLimit = DefaultBlockGasLimit  // 建议 35_000_000
}
// ...
GasLimit: gasLimit,
```

**结论：方案可行，需要添加 MaxGas > 0 检查**

---

## Commit 2 - Issue 13：PendingTxChecker 未复验余额

### 方案评估：✅ 合理但需注意细节

**当前代码确认：**
- `x/evm/ante/sig.go:96-100`: 确实只检查 nonce，未检查余额直接返回 `Accepted`

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| 使用 `latestCtxGetter()` | ✅ 正确 | 确保使用最新状态 |
| 使用 `tx.Cost()` 估算成本 | ✅ 正确 | go-ethereum 的标准方法 |
| 余额不足返回 Pending | ✅ 合理 | 允许后续入金 |

**潜在风险：**
1. `tx.Cost()` 对 EIP-1559 交易使用 `gasFeeCap * gas + value`，这是保守估计，是正确的
2. 需要访问 `evmKeeper.BankKeeper().GetBalance()` 或使用 EVM balance

**关键点：**
- 方案建议使用 `evmKeeper.GetBalance(latestCtx, senderTabiAddr)`
- 但代码中 EVM keeper 没有直接的 `GetBalance` 方法，需要使用 `BankKeeper().GetBalance()` 或 `state.DBImpl.GetBalance()`

**建议修复代码：**
```go
// 在 txNonce < nextPendingNonce 分支前添加余额检查
latestCtx := svd.latestCtxGetter()
senderTabiAddr := svd.evmKeeper.GetTabiAddressOrDefault(latestCtx, evmAddr)
balance := svd.evmKeeper.BankKeeper().GetBalance(latestCtx, senderTabiAddr, "atabi")

// 计算交易成本 (保守估计)
maxCost := new(big.Int).Mul(ethTx.GasFeeCap(), new(big.Int).SetUint64(ethTx.Gas()))
maxCost.Add(maxCost, ethTx.Value())

if balance.Amount.BigInt().Cmp(maxCost) < 0 {
    return abci.Pending  // 余额不足，等待入金
}
return abci.Accepted
```

**结论：方案可行，需要明确使用哪个 balance 查询方法**

---

## Commit 3 - Issue 11：Bloom 为空导致 eth_getLogs 静默漏日志

### 方案评估：✅ 合理

**当前代码确认：**
- `evmrpc/filter.go:1008`: `blockBloom = f.k.GetBlockBloom(providerCtx)` 
- 第 1010 行直接用 `MatchFilters(blockBloom, bloomIndexes)`，没有检查 bloom 是否存在

**`GetBlockBloom` 行为分析：**
```go
// x/evm/keeper/log.go:14-26
func (k *Keeper) GetBlockBloom(ctx sdk.Context) (res ethtypes.Bloom) {
    store := ctx.KVStore(k.storeKey)
    bz := store.Get(types.BlockBloomPrefix)
    if bz != nil {
        res.SetBytes(bz)
        return
    }
    cutoff := k.GetLegacyBlockBloomCutoffHeight(ctx)
    if cutoff == 0 || ctx.BlockHeight() < cutoff {
        res = k.GetLegacyBlockBloom(ctx, ctx.BlockHeight())
    }
    return  // 如果都没找到，返回零值 ethtypes.Bloom{}
}
```

**问题确认：**
- 如果 bloom 不存在（既不在新 key 也不在 legacy key），返回零值 Bloom
- 零值 Bloom 会导致 `MatchFilters` 返回 false，跳过该区块
- 但零值 Bloom 也可能表示"真实没有日志的区块"

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| 区分"缺失"vs"真实零" | ✅ 关键 | 必须这样做 |
| 新增 `getBlockBloomIfExists` | ✅ 合理 | 清晰分离逻辑 |
| 缺失时不跳块 | ✅ 正确 | 宁可慢不可漏 |

**建议实现：**
```go
// x/evm/keeper/log.go 新增
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
```

**结论：方案可行且必要**

---

## Commit 4 - Issue 17：cumulativeGasUsed 恒 0/不正确

### 方案评估：✅ 合理但保守

**当前代码确认：**
- `x/evm/keeper/receipt.go:151`: `CumulativeGasUsed: uint64(0)` - 确实硬编码为 0
- `FlushTransientReceipts` 没有计算累积 gas

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| 不修改 transient key schema | ✅ 保守正确 | 最小变更面 |
| 按 TransactionIndex 排序计算 | ✅ 正确 | EVM 交易顺序 |
| 过滤 EffectiveGasPrice == 0 | ⚠️ 需确认 | 确保不遗漏真实 EVM 交易 |
| 计算顺序与写入顺序分离 | ✅ 关键 | 避免 IAVL changeset 问题 |

**与 Sei Chain 方案对比：**
- Sei 选择修改 key schema（添加 txIndex 前缀），可以直接按顺序迭代
- Tabi 选择不修改 key schema，需要先收集再排序

**Tabi 方案的优势：**
- 变更面最小
- 不影响现有存储结构

**Tabi 方案的风险：**
- 需要在内存中收集所有收据再排序，大区块可能有内存压力
- 但对于正常区块（几百到几千笔交易），这是可接受的

**建议实现骨架：**
```go
func (k *Keeper) FlushTransientReceipts(ctx sdk.Context) error {
    // 1. 收集所有收据
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
    
    // 2. 按 BlockNumber 分组，组内按 TransactionIndex 排序
    // 3. 计算每个区块的累积 gas
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
            if recs[i].receipt.EffectiveGasPrice > 0 { // 只累计真实 EVM 交易
                cumGas += recs[i].receipt.GasUsed
            }
            recs[i].receipt.CumulativeGasUsed = cumGas
        }
    }
    
    // 4. 重新按 txHash 顺序写入 changeset
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
    
    // 5. 写入
    ncs := &proto.NamedChangeSet{Name: types.ReceiptStoreKey, Changeset: iavl.ChangeSet{Pairs: pairs}}
    return k.receiptStore.ApplyChangesetAsync(ctx.BlockHeight(), []*proto.NamedChangeSet{ncs})
}
```

**结论：方案可行，设计合理**

---

## Commit 5 - Issue 14：eth_getTransactionByBlock*AndIndex index 语义错误

### 方案评估：⚠️ 需要仔细处理

**当前代码确认：**
- `evmrpc/tx.go:271`: `getTransactionWithBlock` 直接用 `index` 作为 `block.Block.Txs[int(index)]` 下标
- `evmrpc/tx.go:225`: `GetTransactionByHash` 使用 `receipt.TransactionIndex`（cosmos index）调用 `GetTransactionByBlockNumberAndIndex`

**核心问题：**
1. `eth_` 端点应该使用 EVM-only index
2. 但 `receipt.TransactionIndex` 存储的是 cosmos tx index
3. 需要在查询时进行映射

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| eth_ 遍历计数 EVM tx | ✅ 正确 | 符合以太坊规范 |
| committed 的 GetTransactionByHash 修复 | ⚠️ 复杂 | 需要将 cosmos index 映射为 evm index |
| tabi_ 保持原行为 | ✅ 安全 | 避免意外破坏 |

**关键实现细节：**

需要修改 `getTransactionWithBlock`:
```go
func (t *TransactionAPI) getTransactionWithBlock(block *coretypes.ResultBlock, evmIndex hexutil.Uint) (*ethapi.RPCTransaction, error) {
    // 遍历所有 tx，找到第 evmIndex 个 EVM tx
    var evmCount hexutil.Uint = 0
    for cosmosIdx, txBz := range block.Block.Txs {
        ethtx := getEthTxForTxBz(txBz, t.txConfigProvider(block.Block.Height).TxDecoder())
        if ethtx == nil {
            continue // 跳过非 EVM 交易
        }
        if evmCount == evmIndex {
            // 找到了第 evmIndex 个 EVM 交易
            receipt, err := t.keeper.GetReceipt(t.ctxProvider(LatestCtxHeight), ethtx.Hash())
            // ...构造返回
        }
        evmCount++
    }
    return nil, nil // 未找到
}
```

**GetTransactionByHash 的修复：**
```go
// 不再使用 ByBlockAndIndex，直接构造结果
receipt, err := t.keeper.GetReceipt(t.ctxProvider(LatestCtxHeight), hash)
if err != nil { ... }
height := int64(receipt.BlockNumber)
block, err := blockByNumberWithRetry(ctx, t.tmClient, &height, 1)
if err != nil { ... }

// 在区块中找到交易并直接返回
for _, txBz := range block.Block.Txs {
    ethtx := getEthTxForTxBz(txBz, t.txConfigProvider(height).TxDecoder())
    if ethtx != nil && ethtx.Hash() == hash {
        return constructRPCTransaction(ethtx, block, receipt)
    }
}
```

**结论：方案可行，但实现复杂度较高**

---

## Commit 6 - Issue 16：tx RPC baseFee 取值错误

### 方案评估：✅ 简单直接

**当前代码确认：**
- `evmrpc/tx.go:280`: `baseFeePerGas := t.keeper.GetBaseFee(t.ctxProvider(height))`
- `x/evm/keeper/keeper.go:480-493`: `GetBaseFee` 在正常模式下返回 nil

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| 改用 `GetCurrBaseFeePerGas` | ✅ 正确 | 与其他 RPC 端点一致 |
| 与 block API 口径对齐 | ✅ 必要 | 数据一致性 |

**建议修复代码：**
```go
// evmrpc/tx.go:280 修改为
baseFeePerGas := t.keeper.GetCurrBaseFeePerGas(t.ctxProvider(height)).TruncateInt().BigInt()
```

**结论：方案简单可行**

---

## Commit 7 - Issue 12：SetState 零值删除存储槽

### 方案评估：✅ 简单直接

**当前代码确认：**
- `x/evm/keeper/state.go:30-32`: 永远执行 `Set`，不删除

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| val == 零值时 Delete | ✅ 正确 | EVM 规范行为 |
| 不改变 GetState 语义 | ✅ 正确 | 缺失返回 0 |
| 存量零值不清理 | ✅ 保守 | 可以后续迁移处理 |

**建议修复代码：**
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

**结论：方案简单可行**

---

## Commit 8 - Issue 15：IncrGasCounter float32 精度问题

### 方案评估：✅ 合理但优先级最低

**当前代码确认：**
- `utils/metrics/metrics_util.go:249`: `float32(value)` 直接转换

**问题分析：**
- float32 有效精度约 7 位十进制数字
- 当 gas 值超过 16,777,216 (2^24) 时会损失精度
- 对于区块级别的 gas 统计（可达数百万），确实会有精度损失

**修复策略审查：**

| 项目 | 评估 | 说明 |
|------|------|------|
| 拆分为多次增量 | ⚠️ 可行但复杂 | 可能影响性能 |
| 使用 gauge/histogram | 🔄 更好 | 但需要改变监控配置 |
| 设置最大拆分次数 | ✅ 必要 | 防止极端值导致循环过久 |

**建议保守修复：**
```go
const maxFloat32Safe = float32(16777216) // 2^24

func IncrGasCounter(gasType string, value int64) {
    if value <= 0 {
        return
    }
    // 拆分为安全的增量，最多循环 100 次
    for i := 0; value > 0 && i < 100; i++ {
        incr := value
        if incr > int64(maxFloat32Safe) {
            incr = int64(maxFloat32Safe)
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

**结论：方案可行，优先级正确地设为最低**

---

## 总结与建议

### 优先级确认
1. **Issue 18 (GASLIMIT)**: ✅ 应优先修复，影响合约语义
2. **Issue 13 (余额检查)**: ✅ 优先级合适，防 DoS
3. **Issue 11 (Bloom 缺失)**: ✅ 优先级合适，防静默数据丢失
4. **Issue 17 (累积 Gas)**: ✅ 优先级合适，EVM 兼容性
5. **Issue 14 (Index 语义)**: ✅ 优先级合适，但实现复杂
6. **Issue 16 (baseFee)**: ✅ 简单修复
7. **Issue 12 (零值删除)**: ✅ 简单修复
8. **Issue 15 (指标精度)**: ✅ 正确后置

### 整体风险评估

| 风险类型 | 评估 | 说明 |
|----------|------|------|
| 引入新 bug | 低 | 修复方案设计保守 |
| 性能影响 | 低 | 大部分是轻量修改 |
| 兼容性破坏 | 中 | Issue 14 需仔细测试 |
| 回归风险 | 中 | 需要充分测试覆盖 |

### 关键测试建议

1. **Issue 18**: 部署合约读取 `block.gaslimit`，验证与 RPC 一致
2. **Issue 13**: 模拟余额不足场景，验证 tx 状态为 Pending
3. **Issue 11**: 构造缺失 bloom 的区块，验证 `eth_getLogs` 不漏数据
4. **Issue 17**: 验证多 tx 区块的 `cumulativeGasUsed` 正确累加
5. **Issue 14**: 验证 `eth_getTransactionByBlockNumberAndIndex` 与 `eth_getBlockByNumber(fullTx=true)` 中的 index 一致
