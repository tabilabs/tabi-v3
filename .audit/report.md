原文档链接：Tabi-v3 Issues
01 - 中危：MetricsPanicCallback 缺少 Panic 恢复机制
描述： 用于处理和记录 Panic 的 MetricsPanicCallback 函数本身没有 Panic 恢复机制。如果在该回调函数执行遥测（telemetry）或日志记录操作时发生 Panic，错误将向上传播至调用栈，可能导致整个节点崩溃。
02 - 低危：isOracleTx 函数因空 switch 语句始终返回 false
描述： isOracleTx 函数包含一个没有任何 case 条款的 switch 语句，仅有一个直接返回 false 的 default 分支。这导致该函数始终返回 false，使得 AnteHandle 中分配 OraclePriority 的逻辑永远无法被触发。
03 - 高危：GaslessDecorator.handleWrapped 可能丢失装饰器链的上下文更新
描述： 在 Cosmos SDK 中，sdk.Context 是不可变的，装饰器通过返回新上下文来应用更改。GaslessDecorator.handleWrapped 在循环执行被包裹的装饰器时，未能正确捕捉并传递每个装饰器返回的新上下文。这会导致后续装饰器和最终处理器在旧的（过时的）上下文上运行，从而静默丢失诸如 Gas 计费器更新、优先级变更或事件管理器更新等关键信息。
04 - 高危：乐观处理协程（Optimistic Processing）中存在未处理的 Panic
描述： ProcessProposalHandler 开启了一个用于乐观处理的协程，该协程异步运行 ProcessBlock 但未设置 Panic 恢复。如果任何交易在处理过程中（例如由于恶意构造的数据导致 GetSigners() 解析失败）触发 Panic，该协程会直接崩溃。由于该过程涉及共识逻辑，单次恶意交易即可导致整个区块链网络停机（DoS）。
05 - 高危：乐观处理初始化中的竞态条件
描述： 乐观处理的初始化存在“先检查后执行”的逻辑缺陷。代码在锁外检查完成状态，导致多个协程可能同时看到状态为 nil，随后分别获取锁并启动多个乐观处理协程。这会造成资源浪费和潜在的状态冲突。
06 - 中危：FinalizeBlocker 中的错误传播失效
描述： FinalizeBlocker 在调用 ProcessBlock 时忽略了返回的错误。这意味着即使区块处理过程中发生了严重失败，系统也会在不察觉错误的情况下继续执行，可能导致账本状态不一致。
07 - 中危：EVM RPC 响应中的发送者地址恢复错误
描述： EVM RPC 接口（如 eth_getTransactionByHash 等）在返回交易发起者地址（from）时存在缺陷：
1. 历史交易上下文错误： 查询历史交易时使用了最新区块的上下文创建签名器，导致签名版本选择错误。
2. Chain ID 为 0 的遗留交易处理： 对于 EIP-155 之前的遗留交易，标准恢复函数无法处理 Chain ID 为 0 的情况，导致返回全零地址。
08 - 严重：区块 Gas 估算绕过导致合法交易被拒绝
描述： checkTotalBlockGas 函数在验证区块 Gas 限制时，逻辑存在严重错误。当交易的 Gas 预估值（gasEstimate）超过交易自身的 Gas 上限（gasWanted）时，代码依然无条件使用预估值来计算区块总 Gas。这会导致由于预估偏差，区块总 Gas 虚假地超过 MaxBlockGas 限制，从而使合法的提案被错误地拒绝。
09 - 中危：硬编码拥堵阈值及百分位参数错误导致 Gas 价格估算不准
描述： eth_gasPrice 等接口存在两个逻辑缺陷：
1. 硬编码阈值： 使用固定的 8.5M Gas 作为拥堵判定标准，忽略了实际区块 Gas 上限的动态变化，导致在高上限或低上限链上判断失准。
2. 参数数量级错误： 在调用 FeeHistory 时传递了 0.5 而非 50，导致系统返回的是 0.5 百分位的极低费用，而非中位数费用。
10 - 中危：eth_getBlockByNumber/Hash 中区块已用 Gas 计算错误
描述： RPC 接口在计算区块总已用 Gas（gasUsed）时，错误地累加了所有交易的 Gas。由于代码在过滤掉某些特殊交易（如关联交易、合成交易或 Panic 交易）之前就进行了 Gas 累加，导致最终返回给用户的区块 Gas 指标偏大，包含了实际上并未计入区块响应的交易消耗。

修复方案：参考 Sei 官方机制，通过构建深度防御与上下文对齐完成修复：在共识层，利用强化异步协程的 Panic 隔离，并采用“锁内检查”消除乐观处理的竞态；在应用层，通过严格校验 边界防止合法交易被误杀；在 RPC 层，引入签名器及动态拥堵阈值，确保历史数据查询的准确性与 Gas 费率估算的合理性，全面提升了系统的健壮性与 EVM 兼容性。

11. 中等 - 当无法检索 EVM 专用 Bloom 时，Bloom 过滤器错误跳过区块 (Bloom Filter Incorrectly Skips Blocks)。  建议修复
描述
LogFetcher.processBatch() 中的 eth_getLogs RPC 端点存在逻辑缺陷，会导致无声的数据丢失（Silent Data Loss）。当使用地址或主题（topic）过滤器查询日志时，如果 GetEvmOnlyBlockBloom() 返回一个空的（全零）Bloom 过滤器，系统会错误地跳过该区块。
漏洞代码逻辑 - 针对空 Bloom 过滤器没有防护措施：
Go
// 如果 Bloom 过滤器不匹配，则跳过该区块if !MatchFilters(blockBloom, bloomIndexes) {
    <-dbReadSemaphore
    continue 
}
影响
- 受影响区块中的日志会从 eth_getLogs 的结果中被遗漏，且不会向调用者返回任何错误或警告。
- 依赖事件日志的应用程序（如代币转账、DEX 交易、治理投票、NFT 铸造）可能会错过关键事件，导致应用程序状态错误。
- 区块链索引器和分析平台将获取到不完整的历史数据，从而影响仪表盘、浏览器和报告工具的准确性。

---
12. 中等 - 归零的存储槽未从链状态中删除 (Zeroed Storage Slots Not Deleted)
描述
state.go 中的 SetState 函数在存储槽（Storage Slots）被设置为零时，没有将其删除。相反，它将“零值”作为数据存储，导致键值对（Key-Value Pair）无限期地保留在链状态中。
在 EVM 中，当合约将存储槽设置为零（例如 SSTORE(key, 0)）时，语义上等同于“清除”或“删除”该存储。目前的实现将零视为另一个需要存储的值：
Go
func (k *Keeper) SetState(ctx sdk.Context, addr common.Address, key common.Hash, val common.Hash) {
    // 即使 val 为 0，依然执行 Set 操作
    k.PrefixStore(ctx, types.StateKey(addr)).Set(key[:], val[:])
}
这种做法为每个“已清除”的存储槽存储了 32 字节的零，而不是直接删除该键。
影响
- 即使合约“删除”了数据，链状态也会单调增长。随着时间的推移，这会导致状态数据库显著变大。
- 节点必须存储和同步越来越大的状态数据，从而提高了基础设施的要求和成本。

---
13. 严重 - 未进行充分余额验证即接受挂起交易 (Pending Transactions Accepted Without Sufficient Balance Verification)建议修复
描述
sig.go 中的 EVMSigVerifyDecorator.AnteHandle 函数在处理挂起（Pending）的 EVM 交易时，未验证发送者是否有足够的资金。当交易的 Nonce 处于有效的挂起范围内（在 nextNonceToBeMined 和 nextPendingNonce 之间）时，系统会在不检查发送者余额的情况下接受交易。
漏洞代码路径：
Go
} else if txNonce < nextPendingNonce {
    // 此 Nonce 允许被处理，因为它是从 nextNonceToBeMined 到 nextPendingNonce // 的连续 Nonce 的一部分。// 该逻辑允许来自同一账户的多个 Nonce 在一个区块中被处理。return abci.Accepted
}
当用户提交具有连续 Nonce 的多笔交易时：
1. 提交 Nonce 为 N 的交易，发送者余额充足 → 被接受。
2. 提交 Nonce 为 N+1 的交易 → 进入挂起池（Pending Pool）。
3. 发送者的余额减少（例如，另一笔交易耗尽了资金）。
4. Nonce 为 N+1 的交易稍后通过 PendingTxChecker() 处理 → 被错误地接受。
  - 原因： 挂起交易检查器（Pending Transaction Checker）只验证 Nonce 的顺序，却从未重新验证发送者是否仍然支付得起 gasPrice * gasLimit + value。
影响
- 通过了 Ante 处理器验证的交易将在执行阶段因余额检查失败而失败，这会浪费验证者的计算资源，并可能引起用户的困惑。

---
14. 中等 - RPC eth_getTransactionByBlockAndIndex 中的 transactionIndex 解析错误
描述
eth_getTransactionByBlockNumberAndIndex 和 eth_getTransactionByBlockHashAndIndex RPC 方法错误地将调用者提供的 transactionIndex 解释为 Cosmos 区块级索引，而不是 EVM 专用索引。当一个区块混合包含 Cosmos 交易和 EVM 交易时，这将导致返回错误的交易。
这两个方法都将用户提供的索引直接传递给 getTransactionWithBlock，后者将其用作 block.Block.Txs（Cosmos 交易数组）的偏移量。根据以太坊 JSON-RPC 规范，该索引应仅引用区块中的 EVM 交易，而非所有 Cosmos 交易。
影响
- 交易检索错误： 当 Cosmos 交易位于区块中 EVM 交易之前时，按 EVM 索引查询的调用者会收到错误的交易。
- API 行为不一致： 响应中返回的 transactionIndex 字段反映了正确的 EVM 索引，但通过该索引进行查找却失败或返回不同的交易。
- 受影响的端点：
  - eth_getTransactionByBlockNumberAndIndex
  - eth_getTransactionByBlockHashAndIndex
  - eth_getTransactionByHash (当它内部调用上述方法时)

---
15. 严重 - IncrGasCounter 中的整数溢出 (Integer Overflow) 建议修复
描述
IncrGasCounter 函数存在整数溢出风险。
float32 只能安全地表示高达 $2^{24}$ (16,777,215) 的整数，而 Gas 值最大可达 $2^{63}-1$ (即 int64 的最大值)。将大的 int64 强制转换为 float32 会导致精度丢失和潜在的溢出。
Go
func IncrGasCounter(gasType string, value int64) {
    SafeTelemetryIncrCounterWithLabels(
        []string{"tabi", "tx", "gas", "counter"},
        float32(value),  // 漏洞：没有边界检查的直接强制转换
        []metrics.Label{telemetry.NewLabel("type", gasType)},
    )
}

---
16. 中等 - eth_getTransactionByBlock RPC 响应中的基础费用 (Base Fee) 计算错误
描述
tx.go 中的 getTransactionWithBlock 函数在构建 RPC 交易响应时，使用了错误的方法来检索每 Gas 基础费用（Base Fee Per Gas）。该函数调用 GetBaseFee(t.ctxProvider(height)) 获取的是当前区块的基础费用，但 EIP-1559 交易需要使用前一个区块的基础费用来正确计算有效 Gas 价格（Effective Gas Price）。
漏洞代码：
Go
height := int64(receipt.BlockNumber)
baseFeePerGas := t.keeper.GetBaseFee(t.ctxProvider(height))  // 错误：使用了当前区块
此外，该代码没有处理区块高度为 1 的边缘情况（此时没有前一个区块可供检索）。
影响
- Gas 价格报告错误： 对于 EIP-1559（Type 2）交易，响应中的 gasPrice 字段将不正确，因为它显示的值是基于错误区块的基础费用计算的。

---
17. 中等 - 由于瞬态收据存储无序导致 cumulativeGasUsed 计算错误
描述
交易收据中的 cumulativeGasUsed（累计 Gas 使用量）字段始终为 0 或不正确，这是因为瞬态收据存储机制（Transient Receipt Storage）没有保留区块内的交易顺序。当收据从瞬态存储刷新到持久存储时，无法正确执行累计 Gas 计算。
根本原因：
1. SetTransientReceipt 仅使用交易哈希作为键来存储收据。
2. 当 flushTransientReceipts 遍历瞬态存储以计算累计 Gas 时，收据是按哈希顺序（交易哈希的字典序）返回的，而不是按区块内的交易索引顺序返回的。
Go
// 遍历时按哈希顺序，而非交易顺序// 因此无法正确计算累计 Gasfor ; iter.Valid(); iter.Next() { ... }
影响
- 收据数据错误： 所有交易收据中的 cumulativeGasUsed 字段均为 0 或错误，违反了以太坊 JSON-RPC 规范（该规范要求此字段反映截至当前交易为止所有交易使用的总 Gas）。

---
18. 中等 - RPC 响应与 GASLIMIT 操作码之间的 Gas Limit 不一致 建议修复
描述
EVM 的 GASLIMIT 操作码返回的值与通过 RPC 端点（如 eth_getBlockByNumber）报告的值不同。这种不一致是因为 EVM 的 BlockContext.GasLimit 被设置为了交易的 Gas 池（gp.Gas()），而不是实际区块的共识 Gas 上限。
漏洞代码 (keeper.go):
Go
return &vm.BlockContext{
    // ...
    GasLimit: gp.Gas(),  // 错误：使用了交易 Gas 池剩余量，而非区块 Gas 上限// ...
}, nil
gp.Gas() 代表交易 Gas 池中的剩余 Gas。这会导致：
1. GASLIMIT 操作码不匹配： 执行 block.gaslimit 或内联汇编 gaslimit() 的智能合约接收到的是交易的 Gas 分配量，而不是区块的实际上限。
2. RPC 标头不一致： RPC 后端使用配置参数（GasCap）作为标头中的 GasLimit，导致数据对不上。
影响
- 智能合约逻辑故障： 依赖 block.gaslimit 进行计算（例如基于 Gas 的随机性、批处理决策或 Gas 限制检查）的合约将收到错误的值，可能导致意外行为或被利用。
- 违反以太坊兼容性： 根据以太坊黄皮书，GASLIMIT 操作码 (0x45) 必须返回区块的 Gas 上限。返回不同的值破坏了 EVM 等效性。



