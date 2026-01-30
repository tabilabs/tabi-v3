# Tabi V3 审计修复计划 V2（基于 .audit/report.md）

## 元信息

1. 生成时间: 2026-01-30T16:08:38+08:00
2. 仓库: /home/computer/dev/testing/tabiv3-latest-worktree
3. 分支: fix/security-audit
4. 输入文件: `.audit/report.md`
5. 基线提交: a816ed3
6. 审计约束: 后续严格“一 issue 一 commit”
7. 重要现实约束:
   1. `.audit/report.md` 实际列出 01-18 共 18 个问题（不是 10 个）
   2. 01-10 已在单个提交 a816ed3 合并修复，无法在不重写历史的情况下拆分为 10 个提交
   3. 因此本计划把 01-10 做“issue -> a816ed3 映射”，把 11-18 做“逐个 issue 独立提交”

---

## 推演审查结论（避免引入新问题）

1. 优先修复会影响链上可见语义/共识流程/静默数据丢失的问题
   1. `block.gaslimit` 与 RPC `gasLimit` 不一致（Issue 18）会直接影响合约逻辑与兼容性
   2. PendingTxChecker 不复验余额（Issue 13）会把明显无效 tx 推向执行阶段，放大资源消耗面
   3. Bloom 缺失被当作空 bloom（Issue 11）会造成 eth_getLogs 静默漏数据（最难排查）
2. 修复必须显式约束“计算顺序”与“写入/输出顺序”
   1. Issue 17 cumulativeGasUsed 的正确顺序来自 `TransactionIndex`（交易顺序）
   2. receiptStore 写入 changeset 需要稳定 key 顺序（txHash），两者必须分离，否则容易引入存储层难复现问题
3. 修复 eth_ 端点时要防止意外改变 tabi_ 端点语义
   1. `TabiTransactionAPI` 嵌入 `TransactionAPI`，直接改 TransactionAPI 方法会连带改变 tabi 命名空间
   2. 因此 Issue 14 需要明确：eth_ 修复的同时，tabi_ 端点要么显式保持原行为，要么显式对齐（不能“顺手一起变了”）

---

## 问题清单与当前状态（01-18）

1. 01 MetricsPanicCallback 缺少 panic recover
   - 状态: 已修复（a816ed3）
   - 代码: `utils/panic.go` 已加 recover，且函数 Deprecated/无调用点
2. 02 isOracleTx 空 switch 恒 false
   - 状态: 已修复（a816ed3）
   - 代码: 使用 `utils/IsTxPrioritized`（含 authz 嵌套）替代：`utils/prioritized_txs.go`
3. 03 GaslessDecorator.handleWrapped ctx 更新丢失
   - 状态: 已修复（a816ed3）
   - 代码: `app/antedecorators/gasless.go` 已正确 `ctx = newCtx`
4. 04 optimistic processing goroutine 未 recover panic
   - 状态: 已修复（a816ed3）
5. 05 optimistic processing 初始化竞态
   - 状态: 已修复（a816ed3）
6. 06 FinalizeBlocker 忽略 ProcessBlock error
   - 状态: 已修复（a816ed3）
7. 07 RPC sender 恢复错误（历史 signer/legacy fallback）
   - 状态: 已修复（a816ed3，含测试 `evmrpc/sender_recovery_test.go`）
8. 08 checkTotalBlockGas 估算绕过误杀合法交易
   - 状态: 已修复（a816ed3）
9. 09 gasPrice 阈值/percentile 错误
   - 状态: 已修复（a816ed3）
10. 10 eth_getBlockByNumber/Hash gasUsed 过滤顺序错误
   - 状态: 已修复（a816ed3）

11. 11 Bloom 为空导致 eth_getLogs 静默漏日志
   - 状态: 待修复
12. 12 SetState 设置 0 值不删除导致状态膨胀
   - 状态: 待修复
13. 13 PendingTxChecker 未复验余额即 Accepted
   - 状态: 待修复
14. 14 eth_getTransactionByBlock*AndIndex index 语义错误
   - 状态: 待修复
15. 15 IncrGasCounter int64->float32 精度/语义问题
   - 状态: 待修复（指标问题，优先级后置）
16. 16 tx RPC baseFee 取值错误（当前路径可能传 nil）
   - 状态: 待修复
17. 17 cumulativeGasUsed 恒 0/不正确
   - 状态: 待修复
18. 18 GASLIMIT opcode 与 RPC gasLimit 不一致
   - 状态: 待修复

---

## 优先级与提交策略（11-18）

1. Commit 1: Issue 18（EVM 语义一致性）
2. Commit 2: Issue 13（资源/DoS 面）
3. Commit 3: Issue 11（静默数据丢失）
4. Commit 4: Issue 17（EVM 兼容性）
5. Commit 5: Issue 14（EVM 兼容性）
6. Commit 6: Issue 16（EVM 兼容性）
7. Commit 7: Issue 12（长期状态膨胀）
8. Commit 8: Issue 15（指标精度/语义，最低优先级）

说明:
1. Issue 15 在报告中标“严重”，但它不影响共识/资产安全，属于监控指标精度问题，因此后置
2. 每个 commit 允许包含该 issue 所需的最小测试与验证辅助，但不混入无关重构

---

## 逐 issue 修复计划（每项一个 commit）

### Commit 1 - Issue 18：GASLIMIT opcode 与 RPC gasLimit 不一致

目标（验收）:
1. 链上执行合约读取 `block.gaslimit` 与 `eth_getBlockByNumber.gasLimit` 一致
2. `eth_call` 中合约读取 `block.gaslimit` 与链上/区块查询一致
3. `GasCap` 仍只作为 `eth_call/estimateGas` 的执行 cap，不污染 `block.gaslimit`

现状定位:
1. `x/evm/keeper/keeper.go`：`GetVMBlockContext` 使用 `gp.Gas()`
2. `x/evm/keeper/msg_server.go`：`GetGasPool` 返回 `math.MaxUint64`，导致链上 gaslimit 可能极大值
3. `evmrpc/block.go`：`gasLimit` 使用 `blockRes.ConsensusParamUpdates.Block.MaxGas`（语义可能 off-by-one）
4. `evmrpc/simulate.go`：header.gasLimit 使用 GasCap
5. `evmrpc/subscribe.go`：newHeads 使用 `ResultFinalizeBlock.ConsensusParamUpdates.Block.MaxGas`

修复策略（最小且一致）:
1. 执行路径:
   1. `GetVMBlockContext` 正常链模式改为读取 `ctx.ConsensusParams().Block.MaxGas`（>0）
   2. nil/0 fallback 常量（建议 35,000,000，与默认 genesis 一致）
2. RPC block:
   1. 改为 `tmClient.ConsensusParams(ctx, &height)` 获取“该高度生效” MaxGas
   2. 若 TM RPC 失败: fallback 到旧逻辑/默认值（保证 RPC 可用）
3. RPC simulate:
   1. `Backend.getHeader().GasLimit` 改为共识 MaxGas
   2. `GasCap` 继续只用于 `DoCall/DoEstimateGas` 参数
4. newHeads:
   1. 不再直接用 `ConsensusParamUpdates` 作为 gasLimit 来源
   2. 在 newHeads 编码处取 `tmClient.ConsensusParams(height)`，失败用 fallback（可用“上一次成功值”做兜底，避免每块都硬失败）

推演风险与防回归:
1. 不依赖 `ConsensusParamUpdates` 作为“本高度生效值”，避免未来参数调整出现 off-by-one
2. 不能把 GasCap 当 gasLimit（否则修复无意义且可能影响合约语义）
3. 新增 TM RPC 调用要有降级策略，避免网络抖动导致 eth_* 不可用

建议提交信息:
- fix(audit): align GASLIMIT and rpc gasLimit to consensus MaxGas (issue 18)

---

### Commit 2 - Issue 13：PendingTxChecker 未复验余额

目标（验收）:
1. pending tx 从 Pending -> Accepted 的判定包含余额充足性复验
2. 余额不足时返回 Pending（允许后续入金），而不是 Accepted（浪费执行资源）

现状定位:
- `x/evm/ante/sig.go`：PendingTxChecker 在 nonce 连续时直接 `Accepted`

修复策略（轻量，不引入重执行）:
1. 在 PendingTxChecker 的 “txNonce < nextPendingNonce -> Accepted” 分支前增加余额复验
2. 复验数据源:
   1. latestCtx: `svd.latestCtxGetter()`（只读）
   2. sender tabi 地址: `evmKeeper.GetTabiAddressOrDefault(latestCtx, evmAddr)`
   3. balance: `evmKeeper.GetBalance(latestCtx, senderTabiAddr)`（已扣 locked coins）
3. 成本估算:
   1. 使用 tx 最大成本（保守）:
      - legacy/accessList: gasPrice
      - EIP-1559: gasFeeCap
      - 统一可用 go-ethereum `tx.Cost()`（应为 maxCost）
4. 若 balance < maxCost:
   - 返回 Pending
5. balance 足够:
   - 返回 Accepted（保留原 nonce 逻辑）

推演风险与防回归:
1. 禁止在 PendingTxChecker 内创建 EVM/执行 BuyGas（会放大 DoS 面）
2. 必须使用 latestCtx，不用当前 ctx（否则余额视图可能不代表链最新状态）

建议提交信息:
- fix(audit): re-check balance in pending tx checker (issue 13)

---

### Commit 3 - Issue 11：Bloom 为空导致 eth_getLogs 静默漏日志

目标（验收）:
1. 当 bloom “缺失/不可用”时，不会因为 bloom 不匹配而跳过区块
2. 当 bloom “存在但全零”（真实无日志）时，仍可正常跳过（不退化性能）
3. 不出现静默漏日志

现状定位:
- `evmrpc/filter.go:1003-1013` 对 `GetBlockBloom` 返回值缺少“是否存在”的判定

关键点（避免引入性能灾难）:
- 不能用“bloom==0 就不跳过”这种粗暴做法，因为“没有日志的真实区块 bloom 也会是 0”，那会导致日志查询退化为全量抓块。

修复策略（区分缺失 vs 真实零）:
1. 新增 helper: `getBlockBloomIfExists(ctx) (bloom, ok)`
   1. 直接检查 store 是否存在 `types.BlockBloomPrefix` 的值（存在即 ok）
   2. 若不存在且在 legacy 范围，则再查 legacy key `types.BlockBloomKey(height)` 是否存在
2. 仅当 ok==true 才允许用 `MatchFilters` 跳块
3. ok==false:
   - 禁止跳块，必须 fetch block 并走精确过滤（宁可慢，不可漏）

验证:
1. 单测优先验证“逻辑分支”:
   1. bloom key 缺失时不走 skip
   2. bloom key 存在但全零时仍可 skip
2. 手工验证（如可构造缺失 bloom 的历史高度）:
   - eth_getLogs 不再出现静默漏数据

建议提交信息:
- fix(audit): avoid skipping blocks when bloom is missing (issue 11)

---

### Commit 4 - Issue 17：cumulativeGasUsed 恒 0/不正确（Flush 内排序计算，不改 key）

目标（验收）:
1. `eth_getTransactionReceipt.cumulativeGasUsed` 按 EVM 交易顺序正确累加
2. 不因 key 顺序/存储 changeset 引入新问题

方案选择原因（必须写给审计）:
1. 不修改 transient key schema（变更面最小）
2. 避免现有 `common.Hash(iter.Key())` 对 key 长度的确定性 panic 风险
3. 避免 IAVL changeset 可能要求 key 有序的潜在风险（计算顺序与写入顺序分离）

修复策略:
1. `FlushTransientReceipts` 遍历收集 `{txHash, receipt}`（hash 顺序）
2. 按 `BlockNumber` 分组（兜底测试混块）
3. 每个 block 内按 `TransactionIndex` 升序计算 cumulative（EVM 子序列仍保持相对顺序）
4. 过滤规则:
   - `EffectiveGasPrice == 0` 的 receipt 不计入累计（避免 shell/ante-error 混入 eth_ 视图）
5. 写入 changeset:
   - 仍按 txHash key 顺序输出 pairs（复用 iterator 顺序），避免无序 changeset

推演风险与防回归:
1. 严格分离“计算顺序(TransactionIndex)”与“写入顺序(txHash key)”
2. key 转换用 `common.BytesToHash(iter.Key())`，避免 slice->array 假设

建议提交信息:
- fix(audit): compute cumulativeGasUsed during flush in tx order (issue 17)

---

### Commit 5 - Issue 14：eth_getTransactionByBlock*AndIndex index 语义错误

目标（验收）:
1. eth_ 命名空间下，index 语义为 EVM-only transaction index（符合以太坊 JSON-RPC）
2. 不意外改变 tabi_ 命名空间既有语义（除非明确决定对齐）

现状定位:
- `evmrpc/tx.go:getTransactionWithBlock` 直接把 index 当 cosmos tx 数组下标
- `GetTransactionByHash` committed 分支用 receipt.TransactionIndex（cosmos）去调用 ByBlockAndIndex（会被新语义打破）

修复策略（明确拆分 eth_/tabi_ 行为）:
1. eth_（TransactionAPI）:
   1. 将 ByBlockNumberAndIndex / ByBlockHashAndIndex 实现为:
      - 遍历 block.Block.Txs，计数 EVM tx（跳过 associate），命中第 N 个 EVM tx 返回
2. committed 的 GetTransactionByHash:
   1. 不再把 receipt.TransactionIndex（cosmos）直接传入 ByBlockAndIndex
   2. 改为按 hash 在 block 内扫描定位并构造返回（或先将 cosmosIndex 映射为 evmIndex）
3. tabi_（TabiTransactionAPI）:
   1. 为避免“改 eth_ 顺便把 tabi_ 改坏”，建议显式定义同名方法以保持原行为（cosmos index）或明确对齐（需确认）
   2. 本计划默认: tabi_ 保持原行为，不纳入本次审计修复范围

推演风险与防回归:
1. 直接改 TransactionAPI 方法会连带改变 tabi_ 服务（因为嵌入），必须显式处理
2. 需要回归对比:
   - `eth_getBlockByNumber(fullTx=true)` 返回的 transactionIndex 与 `eth_getTransactionByBlockNumberAndIndex` 一致

建议提交信息:
- fix(audit): interpret tx index as evm-only index in eth block tx queries (issue 14)

---

### Commit 6 - Issue 16：tx RPC baseFee 取值错误（当前路径可能传 nil）

目标（验收）:
1. tx RPC（按 hash / by block）返回的 baseFee/相关字段与 block RPC 口径一致
2. 不再依赖 `keeper.GetBaseFee`（正常链模式返回 nil）

现状定位:
- `evmrpc/tx.go:280-286` 使用 `keeper.GetBaseFee(ctx)`，正常链模式返回 nil

修复策略:
1. `getTransactionWithBlock` 的 `baseFeePerGas` 改为:
   - `keeper.GetCurrBaseFeePerGas(ctxProvider(height)).TruncateInt().BigInt()`
2. 与 `evmrpc/block.go` 的 baseFeePerGas 口径对齐

推演风险与防回归:
1. 避免使用 “当前高度/前一高度” 的臆测，直接与 block API 输出一致即可
2. height 边界（height=1）由 GetCurrBaseFeePerGas 内部兜底处理

建议提交信息:
- fix(audit): align tx rpc baseFee with block baseFee (issue 16)

---

### Commit 7 - Issue 12：SetState 零值删除存储槽

目标（验收）:
1. `SetState(..., 0)` 会删除 key，而不是持久化写入 32 字节 0
2. 不改变 GetState 语义（缺失仍返回 0）

现状定位:
- `x/evm/keeper/state.go:30-32` 永远 Set

修复策略:
1. val == (common.Hash{}) 时执行 Delete(key)
2. 否则 Set(key,val)

推演风险与防回归:
1. 存量零值槽不会自动清理（历史数据问题），不混入本 commit
2. 回归 IterateState 不应遍历到已清零槽

建议提交信息:
- fix(audit): delete storage slots when set to zero (issue 12)

---

### Commit 8 - Issue 15：IncrGasCounter float32 精度/语义问题（指标，后置）

目标（验收）:
1. 指标逻辑不因极大 gas 值出现明显错误/误导
2. 不引入明显性能开销

现状定位:
- `utils/metrics/metrics_util.go:246-252` 直接 `float32(value)`

修复策略（MVP）:
1. value <= 0 直接 return
2. 若 value 很大:
   1. 可选策略 A: 拆分为多次 <= 2^24 的增量（降低单次转换误差）
   2. 可选策略 B: 直接记录为 gauge/histogram（会涉及指标体系变更，建议单独立项）
3. 本计划默认采用 A（不改指标名，不改监控配置）

推演风险与防回归:
1. 不把指标问题当成共识/资产风险处理，避免扩大修改面
2. 拆分逻辑要有上限，防止极端值导致循环过久（可设置最大拆分次数）

建议提交信息:
- fix(audit): harden gas counters against float32 precision limits (issue 15)

---

## 建议的 commit message 模板（统一风格）

1. fix(audit): align GASLIMIT and rpc gasLimit to consensus MaxGas (issue 18)
2. fix(audit): re-check balance in pending tx checker (issue 13)
3. fix(audit): avoid skipping blocks when bloom is missing (issue 11)
4. fix(audit): compute cumulativeGasUsed during flush in tx order (issue 17)
5. fix(audit): interpret tx index as evm-only index in eth block tx queries (issue 14)
6. fix(audit): align tx rpc baseFee with block baseFee (issue 16)
7. fix(audit): delete storage slots when set to zero (issue 12)
8. fix(audit): harden gas counters against float32 precision limits (issue 15)

---

## 审计对照交付建议（解决 01-10 无法拆分的问题）

1. 提供映射表:
   1. issue 01-10 -> a816ed3
   2. 每条 issue 指向具体文件/关键行（便于审计复核）
2. 后续 11-18 每个 issue 独立提交，提交信息固定包含 issue 编号
3. 每个 issue 提供最小复现/验证步骤（至少手工可复现）
