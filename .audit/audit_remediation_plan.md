# Tabi V3 审计修复计划（EVM 兼容性）

## 元信息

1. 日期: 2026-01-30T12:03:11+08:00
2. 分支: fix/security-audit
3. 输入报告: `.audit/audit_analysis_report.md`
4. 约束: 每修复一个审计 issue 提交一个 commit（issue 内允许包含必要的测试/文档/回归脚本）

## 范围与优先级

1. Issue A（Critical）: GASLIMIT 操作码与 RPC `gasLimit` 不一致
2. Issue B（High）: Receipt 的 `cumulativeGasUsed` 计算/排序错误

说明:
- 先修 Issue A，因为它会影响合约运行时语义（`block.gaslimit`）且当前链上返回值可能是 `math.MaxUint64`。
- Issue B 主要影响 RPC 兼容性与索引/钱包体验，但对链上状态转移影响较小。

## 关键假设与待确认点

1. Gas 单位假设: `ctx.ConsensusParams().Block.MaxGas` 与 `receipt.GasUsed` 使用同一计量单位（当前代码多处隐含该假设）。
2. Synthetic receipt 假设: `TxType == ShellEVMTxType` 或 `EffectiveGasPrice == 0` 的 receipt 视为 synthetic（现有 RPC 过滤逻辑已使用该判定）。
3. RPC GasCap 假设: `GasCap` 仅用于限制 `eth_call/estimateGas` 的单次执行资源，不应成为 `block.gaslimit` 的来源。

如果以上任何假设不成立，需要在实现前先定口径（会显著扩大工作量与审计沟通成本）。

## 提交策略（每个 issue 一个 commit）

1. Commit #1: 修复 Issue A（GASLIMIT / gasLimit 不一致）
2. Commit #2: 修复 Issue B（cumulativeGasUsed）

建议 commit message 直接标注审计 issue，方便审计方按提交定位:
1. `fix(evm): align GASLIMIT with consensus MaxGas (audit issue A)`
2. `fix(evm): compute receipt cumulativeGasUsed in tx order (audit issue B)`

## Commit #1 - 修复 GASLIMIT / gasLimit 不一致（Issue A, Critical）

### 目标（验收标准）

1. 链上执行时（真实交易）: `GASLIMIT` opcode / `block.gaslimit` 返回值等于共识 `Block.MaxGas`（或其等价换算值）。
2. RPC `eth_getBlockByNumber`/`eth_getBlockByHash` 返回的 `gasLimit` 与链上执行一致。
3. RPC 模拟（`eth_call`/`eth_estimateGas`/tracing）里合约读取到的 `block.gaslimit` 与链上一致。

### 变更点（建议最小实现）

1. `x/evm/keeper/keeper.go`
   1. 引入 helper: `getBlockGasLimit(ctx sdk.Context) uint64`
      1. 优先取 `ctx.ConsensusParams().Block.MaxGas`（>0）
      2. 否则使用 `DefaultBlockGasLimit` 兜底（值需与链默认共识配置保持一致）
   2. 修改 `GetVMBlockContext`:
      1. `GasLimit: gp.Gas()` -> `GasLimit: getBlockGasLimit(ctx)`
2. `evmrpc/block.go` 与 `evmrpc/subscribe.go`
   1. `gasLimit` 的来源统一到“该高度生效的共识参数 Block.MaxGas”（避免依赖 `ConsensusParamUpdates` 可能为空、或语义为下一高度生效而产生 off-by-one）
3. `evmrpc/simulate.go`
   1. 复核 `eth_call/estimateGas` 的 block context 获取路径，确保最终 `BlockContext.GasLimit` 走共识参数
   2. `GasCap` 仍作为 RPC 侧的执行 cap（不要把它当作区块 gasLimit）
   3. 将 `Backend.getHeader().GasLimit` 也切到共识参数（block.gaslimit 口径）；`GasCap` 只用于 `DoCall/DoEstimateGas` 的资源上限

### 验证与回归（建议给审计方的可复现步骤）

1. 链上交易验证
   1. 部署一个合约，返回 `block.gaslimit`
   2. 发送交易调用该方法，记录返回值
   3. 对同高度调用 `eth_getBlockByNumber`，对比 `gasLimit` 一致
2. RPC 模拟验证
   1. 对同高度用 `eth_call` 调用合约方法
   2. 确认返回值与链上交易/区块查询一致

### 风险与回滚

1. 若 PriorityNormalizer 非 1，需要确认 gas 单位换算策略（否则可能导致 `block.gaslimit` 不再等于 RPC 的 `gasLimit`）。
2. 本 commit 可独立 revert（不依赖 Issue B）。

## Commit #2 - 修复 cumulativeGasUsed（Issue B, High）

### 目标（验收标准）

1. `eth_getTransactionReceipt` 返回的 `cumulativeGasUsed` 符合以太坊语义: 按区块内交易顺序累计到当前交易。
2. 交易顺序以区块内 `TransactionIndex` 对应的顺序为准（Tabi 内部存的是 Cosmos TxIndex，EVM 交易是其子序列）。
3. 不将 synthetic receipt 计入 eth_ 视图的累计（与现有过滤语义一致）。

### 方案选择原因（采用 Flush 内排序计算，不改 transient key）

选择“在 Flush 阶段按 `TransactionIndex` 排序后计算累计值”，而不是“修改 transient key 结构（txIndex+txHash）”的原因:

1. 变更面最小
   1. 不改 transient key schema
   2. 不改 `SetTransientReceipt/GetTransientReceipt/DeleteTransientReceipt` 签名
   3. 不需要全量更新调用点（`app/receipt.go`、`x/evm/keeper/evm.go`、`x/evm/keeper/msg_server.go` 等）
2. 避免引入确定性的 panic
   1. 现有 `FlushTransientReceipts` 里存在 `common.Hash(iter.Key())` 的强假设（iter.Key() 必须是 32 bytes txHash）
   2. 一旦把 transient key 改为字符串格式，现有逻辑会 runtime panic，属于“必炸”而非“概率 bug”
3. 避免存储层/changeset 顺序风险
   1. 当前写持久化走 `iavl.ChangeSet{Pairs: ...}` -> `receiptStore.ApplyChangesetAsync`
   2. 常见 IAVL changeset 实现要求 pairs 按 key 排序；如果为了累计值按 txIndex 顺序写入，会导致持久化 key（txHash）无序，可能引入难复现的写入/回放问题
4. 满足审计语义且更易审计
   1. 累计 gas 的正确性来自“按交易顺序累计”，与 transient key 的字典序无强绑定
   2. 改动集中在一个函数（`FlushTransientReceipts`），更符合“一个 issue 一个 commit”的可追溯性与审计对照效率

### 变更点（最小改动实现）

1. `x/evm/keeper/receipt.go`
   1. 保持 transient key 仍为 `types.ReceiptKey(txHash)`（不改写入/读取调用点）
   2. 修改 `FlushTransientReceipts`:
      1. 遍历 transient receipt store，收集 `{txHash, receipt}` 记录（保留原迭代顺序用于最终写入）
      2. 按 `receipt.BlockNumber` 分组（仅为测试/极端情况兜底；正常情况下 transient store 只含一个区块）
      3. 每个 block 内按 `receipt.TransactionIndex` 升序排序后计算累计值
      4. 累计规则（与现有 RPC 过滤语义保持一致）:
         1. 仅当 `receipt.EffectiveGasPrice != 0` 时计入累计（避免把 shell receipt / ante-error receipt 混入 eth_ 视图）
         2. 对不计入累计的 receipt，可保持 `CumulativeGasUsed` 为 0（或不变）
      5. 将更新后的 receipt 重新 Marshal，写入 changeset
      6. 写入顺序保持为“按持久化 key（txHash）排序”的顺序（复用 iterator 的天然顺序），避免 changeset key-order 风险

### 验证与回归

1. 手工验证
   1. 在同一高度连续发送 2 笔 EVM 交易（确保都成功）
   2. 查询两笔 receipt:
      1. 第一笔 `cumulativeGasUsed == gasUsed1`
      2. 第二笔 `cumulativeGasUsed == gasUsed1 + gasUsed2`
2. 单元测试（优先写纯函数/轻依赖测试）
   1. 累计逻辑（排序 + 过滤 + 累加）: 构造 3 笔 receipt（含 1 笔 `EffectiveGasPrice==0`），断言 cumulative 仅对真实 EVM receipt 正确累加
   2. 多 block 兜底: 构造不同 `BlockNumber` 的 receipt，断言分别累计（避免测试环境 transient store 混入多 block 时出现串账）

### 风险与回滚

1. 需要显式保证: “计算顺序按 TransactionIndex” 与 “写入顺序按 key 排序” 分离，否则可能引入 changeset 顺序问题。
2. 对 synthetic receipt 的累计语义如果需要在 `tabi_` 视图里也严格一致，需要额外定义口径（不建议塞进本次审计 issue commit）。
3. 本 commit 可独立 revert。

## 已知阻碍（影响验证，不建议混入审计 issue commit）

1. 当前 `go test` 可能无法编译: `testutil/keeper/evm.go` 引用了 `app.Setup`，但仓库内不存在该函数。
2. 需要在开始实现前决定验证策略:
   1. 选项 A: 额外用一个单独 commit 修复测试基建（不计入审计 issue，避免污染 issue->commit 对应关系）
   2. 选项 B: 审计期以手工可复现验证 + 静态检查（`go vet`/`go build`）兜底
