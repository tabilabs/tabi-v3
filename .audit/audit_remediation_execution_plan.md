# 审计问题修复执行计划（Tabi V3）

## 元信息

1. 生成时间: 2026-01-30T19:22:57+08:00
2. 分支: fix/security-audit
3. 依据文档: `.audit/report.md`
4. 计划约束: 从现在开始严格遵守“每个 issue 一个 commit”
5. 当前现实: 01-10 已合并在单个提交 a816ed3（无法在不重写历史的情况下拆分）

---

## 当前状态复核（基于代码推演）

1. 已修复到位
   1. 01-10 基本修复到位（见 a816ed3）
   2. 17 已修复到位（`x/evm/keeper/receipt.go` 在 Flush 阶段按 `TransactionIndex` 排序计算 `cumulativeGasUsed` 并写入持久化 store）
2. 未修复或仅部分修复
   1. 11 未修复（Bloom 缺失/不可用时会静默 skip block，造成 eth_getLogs 漏数据）
   2. 12 未修复（`SetState` 写零不 delete，状态膨胀）
   3. 13 未修复（PendingTxChecker 未复验余额）
   4. 14 未修复（eth_getTransactionByBlock*AndIndex index 仍按 Cosmos index 解释）
   5. 15 未修复（指标精度/语义问题）
   6. 16 未修复（tx RPC baseFee 取值路径错误，当前可能传 nil）
   7. 18 部分修复（EVM 执行侧已对齐共识 MaxGas；RPC block/newHeads/simulate header 仍不一致）

---

## 阻塞项（如果要“可验证交付”，必须先定）

1. 当前 `go build ./...` 失败
   1. `cmd/tabid` 链接失败: `github.com/fjl/memsize: invalid reference to runtime.stopTheWorld`（toolchain go1.24.7 下不兼容）
   2. `testutil/*` 包编译失败（引用 `app.Setup`/EpochKeeper 等不存在符号）
2. 当前 `go test ./...` 失败（除上述 build 失败外，还有明显坏测试）
   1. `app/antedecorators/traced_test.go` 依赖未定义符号
   2. `aclmapping/utils` 测试用例 panic（缺 mapping）

建议处理方式（推荐默认）:
1. 先做一个“非审计 issue”的基础设施修复提交，用来恢复 `go build ./...` 的可用性
2. 审计 issue 的提交严格按 11-18 每个 issue 一个 commit

---

## 需要你确认的 6 个决策点（我给默认值）

1. 是否允许先做 1 个非审计 commit 修复 build 阻塞（memsize + testutil）
   - 默认: 允许（否则后续无法提供基本编译证据）
2. Issue 14 的 index 语义是否需要同时对齐 tabi_ 命名空间
   - 默认: 只保证 eth_ 正确；tabi_ 若受 embedding 影响需要显式 override 以保持旧行为
3. Issue 13 余额不足时 PendingTxChecker 返回值
   - 默认: 返回 Pending（而不是 Rejected），避免误杀“后续入金即可变有效”的交易
4. Issue 11 Bloom 缺失/不可用时的行为
   - 默认: 不能用 bloom skip；必须 fetch block 做精确过滤；同时打点/日志一次用于观测
5. Issue 18 RPC gasLimit 来源
   - 默认: 优先用 `ctxProvider(height).ConsensusParams().Block.MaxGas`；不足时 fallback 到 `tmClient.ConsensusParams(height)`；再 fallback 到 `keeper.DefaultBlockGasLimit`
6. 是否把“ProcessProposalHandler else 分支潜在 nil deref 防御”纳入本轮
   - 默认: 作为额外 hardening（不占用审计 issue 编号，单独 commit）

---

## 执行顺序（按风险优先级，且保持一 issue 一 commit）

0. 可选 Prep Commit（非审计）: 修复 go build 阻塞（memsize + testutil）
1. Issue 18: GASLIMIT / gasLimit 对齐（补齐 RPC/newHeads/simulate）
2. Issue 13: PendingTxChecker 余额复验
3. Issue 11: Bloom 缺失导致 eth_getLogs 静默漏数据
4. Issue 14: eth_getTransactionByBlock*AndIndex index 语义修复
5. Issue 16: tx RPC baseFee 修复
6. Issue 12: SetState 零值删除
7. Issue 15: IncrGasCounter 精度/语义修复
8. 可选 Hardening Commit（非审计）: optimisticProcessingInfo nil guard

---

## 每个 issue 的实施要点（避免引入新问题）

## Issue 18（补齐 RPC 一致性）

1. 现状
   1. EVM 执行侧已修: `x/evm/keeper/keeper.go` 从共识 `MaxGas` 设置 `BlockContext.GasLimit`
   2. RPC 仍不一致:
      1. `evmrpc/block.go` 使用 `ConsensusParamUpdates.Block.MaxGas`
      2. `evmrpc/subscribe.go` newHeads 使用 `ConsensusParamUpdates.Block.MaxGas`
      3. `evmrpc/simulate.go` header.GasLimit 使用 GasCap
2. 修复策略（最小改动）
   1. block/newHeads/simulate 三处统一 gasLimit 来源（见“决策点 5”默认）
   2. 约束: `GasCap` 仍只作为 `eth_call/estimateGas` 的执行 cap，不作为 `block.gaslimit`
3. 验证
   1. 合约读取 `block.gaslimit`（链上 tx / eth_call）与 `eth_getBlockByNumber.gasLimit` 一致
   2. tracing/newHeads 输出 `gasLimit` 与 `eth_getBlockByNumber` 一致

建议 commit message:
- fix(audit): align rpc gasLimit with GASLIMIT opcode (issue 18)

---

## Issue 13（PendingTxChecker 余额复验）

1. 现状
   1. `x/evm/ante/sig.go` 在 nonce 连续时直接 `Accepted`，未复验余额
2. 修复策略（轻量）
   1. 在 PendingTxChecker 里使用 latestCtx 取 sender balance（必须与 stateDB 逻辑一致）
      1. `senderTabiAddr := evmKeeper.GetTabiAddressOrDefault(latestCtx, evmAddr)`
      2. `balanceWei := evmKeeper.GetBalance(latestCtx, senderTabiAddr)`
   2. 成本估算使用 go-ethereum `tx.Cost()`（max cost）
   3. 若余额不足: 返回 Pending（默认）
3. 验证
   1. 扩展/新增单测覆盖:
      1. nonce 连续但余额不足 -> Pending
      2. 余额足够 -> Accepted

建议 commit message:
- fix(audit): re-check balance in pending tx checker (issue 13)

---

## Issue 11（Bloom 缺失导致 eth_getLogs 静默漏数据）

1. 现状
   1. `evmrpc/filter.go` bloom 不匹配直接 skip
   2. 当 ctxProvider(height) 无法提供该高度的正确状态（pruned/失败 fallback）时，bloom 可能为“看似空”，导致错误 skip
2. 修复策略（区分“可用”与“不可用”）
   1. 仅当以下条件都满足，才允许 bloom skip:
      1. `providerCtx.BlockHeight() == height`
      2. 该高度 EVM store 版本存在（可用 `evmExists(providerCtx, k)` 作为 guard）
      3. 能取到 bloom 原始 bytes（对新 scheme: `store.Get(types.BlockBloomPrefix) != nil`；对 legacy: 需要显式判断 key 是否存在）
   2. 若 bloom 不可用:
      1. 禁止 skip，必须 fetch block 并走精确过滤
      2. 打点/日志一次，提示 bloom 不可用（便于运维定位 pruning/状态缺失）
3. 验证
   1. 单测覆盖分支（核心是保证“不可信 bloom 时不 skip”）
   2. 手工: 在 pruning 节点或故障注入下验证不再静默漏日志

建议 commit message:
- fix(audit): do not skip blocks when bloom is unavailable (issue 11)

---

## Issue 14（eth_getTransactionByBlock*AndIndex index 语义）

1. 现状
   1. `evmrpc/tx.go:getTransactionWithBlock` 直接用 `block.Block.Txs[int(index)]`（Cosmos index）
   2. block 返回的 `transactionIndex` 是 EVM-only（与查询语义不一致）
   3. 由于 TabiTransactionAPI embedding，会影响 tabi_ 命名空间（需按“决策点 2”处理）
2. 修复策略
   1. eth_：把 index 解释为 EVM-only index
      1. 遍历 block txs，按与 block API 相同的过滤规则计算“第 N 个 EVM tx”
      2. 命中后返回
   2. 修复 committed `GetTransactionByHash`：不能再用 receipt.TransactionIndex（Cosmos）去反查 ByBlockAndIndex
      1. 改为在 block 内按 hash 定位并计算 EVM-only index
   3. 若需要保持 tabi_ 旧行为: 在 TabiTransactionAPI 上显式 override 方法
3. 验证
   1. 混合块（Cosmos+EVM）场景:
      1. `eth_getBlockByNumber(fullTx=true)` 返回的 transactionIndex
      2. `eth_getTransactionByBlockNumberAndIndex` 用该 index 能取回同一 tx

建议 commit message:
- fix(audit): interpret tx index as evm-only index in block tx queries (issue 14)

---

## Issue 16（tx RPC baseFee 取值修复）

1. 现状
   1. `evmrpc/tx.go` 使用 `keeper.GetBaseFee`，正常链模式返回 nil
2. 修复策略
   1. 改为与 block API 一致的 baseFee 来源:
      - `keeper.GetCurrBaseFeePerGas(ctxProvider(height)).TruncateInt().BigInt()`
3. 验证
   1. type2 交易的 RPC 输出字段不再依赖 nil baseFee

建议 commit message:
- fix(audit): use per-block baseFee in tx rpc responses (issue 16)

---

## Issue 12（SetState 零值删除）

1. 现状
   1. `x/evm/keeper/state.go:SetState` 永远 Set
2. 修复策略
   1. `val == (common.Hash{})` 时 Delete(key)
   2. 否则 Set
3. 验证
   1. 单测: set non-zero -> get
   2. set zero -> key 不存在（或 IterateState 不再包含）

建议 commit message:
- fix(audit): delete storage slots when set to zero (issue 12)

---

## Issue 15（IncrGasCounter 精度/语义）

1. 现状
   1. `utils/metrics/metrics_util.go` 直接 `float32(value)`
2. 修复策略（最小）
   1. `value <= 0` 直接 return
   2. 大值拆分增量（上限保护，避免循环过久）
3. 验证
   1. 单测: 不对负值计数；大值不会 panic/不会明显失真

建议 commit message:
- fix(audit): harden gas counter metrics for float32 precision (issue 15)

---

## 验证策略（在当前仓库可执行的现实版）

1. 编译/测试现状
   1. `go build ./...` 当前失败（memsize + testutil）
   2. `go test ./...` 当前失败（除 build 失败外还有坏测试）
2. 推荐交付证据
   1. 每个 issue 完成后至少保证:
      1. `go fmt ./...`
      2. `go vet ./...`（若 vet 也受 build 阻塞，需要先修 build）
      3. 针对性 package build（例如 `go build ./evmrpc ./x/evm/...`）
   2. 在 build 阻塞修复后，再恢复 `go build ./...` 和（可行的话）最小 `go test` 集

---

## 输出物

1. 审计执行计划文件
   1. `.audit/audit_remediation_execution_plan.md`
   2. `~/coding/.codex/tasks/project_tabiv3/fix/security-audit/03_Audit_Remediation_Execution_Plan.md`
2. 后续每个 issue 的提交命名（建议）
   1. `fix(audit): ... (issue 18)`
   2. `fix(audit): ... (issue 13)`
   3. `fix(audit): ... (issue 11)`
   4. `fix(audit): ... (issue 14)`
   5. `fix(audit): ... (issue 16)`
   6. `fix(audit): ... (issue 12)`
   7. `fix(audit): ... (issue 15)`
