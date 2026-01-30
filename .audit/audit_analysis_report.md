# Tabi V3 审计问题分析报告

## 概述

本报告对照分析了 Tabi V3 中两个已知的审计问题，验证了它们在代码中的存在，并分析了 Sei Chain 如何修复这些问题，为 Tabi V3 的修复提供参考方案。

---

## 问题1: CumulativeGasUsed 计算错误

### 问题描述

`SetTransientReceipt` 仅使用交易哈希作为键来存储收据。当 `flushTransientReceipts` 遍历瞬态存储以计算累计 Gas 时，收据是按哈希顺序（交易哈希的字典序）返回的，而不是按区块内的交易索引顺序返回的。

### 影响

- **收据数据错误**：所有交易收据中的 `cumulativeGasUsed` 字段均为 0 或错误值
- **违反以太坊 JSON-RPC 规范**：该规范要求此字段反映截至当前交易为止所有交易使用的总 Gas

### Tabi V3 问题代码验证

#### 文件: `x/evm/keeper/receipt.go`

```go
// 第22-31行 - SetTransientReceipt
func (k *Keeper) SetTransientReceipt(ctx sdk.Context, txHash common.Hash, receipt *types.Receipt) error {
    store := ctx.TransientStore(k.transientStoreKey)
    bz, err := receipt.Marshal()
    if err != nil {
        return err
    }
    // 问题：仅使用交易哈希作为键，没有包含交易索引
    store.Set(types.ReceiptKey(txHash), bz)
    return nil
}
```

```go
// 第117-136行 - FlushTransientReceipts
func (k *Keeper) FlushTransientReceipts(ctx sdk.Context) error {
    // 问题：直接遍历，按哈希的字典序而不是交易索引顺序
    iter := prefix.NewStore(ctx.TransientStore(k.transientStoreKey), types.ReceiptKeyPrefix).Iterator(nil, nil)
    defer iter.Close()
    var pairs []*iavl.KVPair
    var changesets []*proto.NamedChangeSet
    for ; iter.Valid(); iter.Next() {
        // 问题：没有计算累积 Gas，直接使用原始值
        kvPair := &iavl.KVPair{Key: types.ReceiptKey(common.Hash(iter.Key())), Value: iter.Value()}
        pairs = append(pairs, kvPair)
    }
    // ...
}
```

```go
// 第149-151行 - WriteReceipt
receipt := &types.Receipt{
    TxType:            txType,
    CumulativeGasUsed: uint64(0),  // 问题：硬编码为0
    // ...
}
```

#### 文件: `x/evm/types/keys.go`

```go
// 第85-87行 - ReceiptKey 仅使用哈希
func ReceiptKey(txHash common.Hash) []byte {
    return append(ReceiptKeyPrefix, txHash[:]...)
}
// 问题：没有类似 Sei 的 NewTransientReceiptKey 函数
```

### Sei Chain 修复方案

#### 文件: `x/evm/types/keys.go` (Sei Chain)

```go
// Sei Chain 新增了 TransientReceiptKey 类型
type TransientReceiptKey []byte

// 新增函数：使用交易索引+交易哈希作为键，确保按正确顺序存储
func NewTransientReceiptKey(txIndex uint64, txHash common.Hash) TransientReceiptKey {
    // 格式：前缀 + "%020d:%s" (20位填充的交易索引 + 冒号 + 交易哈希)
    // 这确保了迭代器按交易索引顺序返回
    return append(ReceiptKeyPrefix, fmt.Sprintf("%020d:%s", txIndex, txHash.String())[:]...)
}

func (trk TransientReceiptKey) TransactionHash() common.Hash {
    if i := bytes.LastIndexByte(trk, ':'); i != -1 {
        return common.HexToHash(string(trk[i+1:]))
    }
    return common.Hash{}
}
```

#### 文件: `x/evm/keeper/receipt.go` (Sei Chain)

```go
// Sei Chain 的 SetTransientReceipt - 使用包含交易索引的键
func (k *Keeper) SetTransientReceipt(ctx sdk.Context, txHash common.Hash, receipt *types.Receipt) error {
    store := ctx.TransientStore(k.transientStoreKey)
    bz, err := receipt.Marshal()
    if err != nil {
        return err
    }
    // 修复：使用交易索引+哈希作为键
    store.Set(types.NewTransientReceiptKey(uint64(receipt.TransactionIndex), txHash), bz)
    return nil
}

// Sei Chain 的 GetTransientReceipt - 需要交易索引参数
func (k *Keeper) GetTransientReceipt(ctx sdk.Context, txHash common.Hash, txIndex uint64) (*types.Receipt, error) {
    store := ctx.TransientStore(k.transientStoreKey)
    bz := store.Get(types.NewTransientReceiptKey(txIndex, txHash))
    // ...
}
```

```go
// Sei Chain 的 flushTransientReceipts - 正确计算累积 Gas
func (k *Keeper) flushTransientReceipts(ctx sdk.Context, sync bool) error {
    transientReceiptStore := prefix.NewStore(ctx.TransientStore(k.transientStoreKey), types.ReceiptKeyPrefix)
    iter := transientReceiptStore.Iterator(nil, nil)
    defer func() { _ = iter.Close() }()
    var pairs []*iavl.KVPair

    // 修复：使用 map 跟踪每个区块的累积 Gas
    cumulativeGasUsedPerBlock := make(map[uint64]uint64)
    for ; iter.Valid(); iter.Next() {
        receipt := &types.Receipt{}
        if err := receipt.Unmarshal(iter.Value()); err != nil {
            return err
        }

        // 修复：累积计算 Gas（跳过遗留收据）
        if !isLegacyReceipt(ctx, receipt) {
            cumulativeGasUsedPerBlock[receipt.BlockNumber] += receipt.GasUsed
            receipt.CumulativeGasUsed = cumulativeGasUsedPerBlock[receipt.BlockNumber]
        }

        marshalledReceipt, err := receipt.Marshal()
        if err != nil {
            return err
        }

        // 修复：提取原始交易哈希作为持久化存储的键
        kvPair := &iavl.KVPair{
            Key:   types.ReceiptKey(types.TransientReceiptKey(iter.Key()).TransactionHash()),
            Value: marshalledReceipt,
        }
        pairs = append(pairs, kvPair)
    }
    // ...
}
```

### 推荐修复步骤

1. **修改 `x/evm/types/keys.go`**：
   - 新增 `TransientReceiptKey` 类型
   - 新增 `NewTransientReceiptKey(txIndex uint64, txHash common.Hash)` 函数
   - 确保键格式为 `prefix + "%020d:%s"` 以保证正确排序

2. **修改 `x/evm/keeper/receipt.go`**：
   - `SetTransientReceipt`: 使用 `NewTransientReceiptKey` 包含交易索引
   - `GetTransientReceipt`: 增加 `txIndex` 参数
   - `FlushTransientReceipts`: 实现累积 Gas 计算逻辑

3. **更新所有调用点**：确保 `GetTransientReceipt` 的所有调用都传递正确的交易索引

---

## 问题2: RPC 响应与 GASLIMIT 操作码之间的 Gas Limit 不一致

### 问题描述

EVM 的 `GASLIMIT` 操作码返回的值与通过 RPC 端点（如 `eth_getBlockByNumber`）报告的值不同。EVM 的 `BlockContext.GasLimit` 被设置为了交易的 Gas 池（`gp.Gas()`），而不是实际区块的共识 Gas 上限。

### 影响

- **智能合约逻辑故障**：依赖 `block.gaslimit` 进行计算的合约将收到错误的值
- **违反以太坊兼容性**：根据以太坊黄皮书，`GASLIMIT` 操作码 (0x45) 必须返回区块的 Gas 上限

### Tabi V3 问题代码验证

#### 文件: `x/evm/keeper/keeper.go`

```go
// 第238-281行 - GetVMBlockContext
func (k *Keeper) GetVMBlockContext(ctx sdk.Context, gp core.GasPool) (*vm.BlockContext, error) {
    // ...
    return &vm.BlockContext{
        CanTransfer: core.CanTransfer,
        Transfer:    txfer,
        GetHash:     k.GetHashFn(ctx),
        Coinbase:    coinbase,
        GasLimit:    gp.Gas(),  // 问题：使用了交易 Gas 池剩余量，而非区块 Gas 上限
        BlockNumber: big.NewInt(ctx.BlockHeight()),
        Time:        uint64(ctx.BlockHeader().Time.Unix()),
        Difficulty:  utils.Big0,
        BaseFee:     baseFee,
        BlobBaseFee: utils.Big1,
        Random:      &rh,
    }, nil
}
```

### Sei Chain 修复方案

#### 文件: `x/evm/keeper/keeper.go` (Sei Chain)

```go
// Sei Chain 定义了默认区块 Gas 上限常量
const DefaultBlockGasLimit = 10000000

func (k *Keeper) GetVMBlockContext(ctx sdk.Context, gp core.GasPool) (*vm.BlockContext, error) {
    // ...
    return &vm.BlockContext{
        CanTransfer: core.CanTransfer,
        Transfer:    txfer,
        GetHash:     k.GetHashFn(ctx),
        Coinbase:    coinbase,
        // 修复：使用真正的区块 Gas 上限
        GasLimit: func() uint64 {
            if ctx.ConsensusParams() != nil && ctx.ConsensusParams().Block != nil {
                return uint64(ctx.ConsensusParams().Block.MaxGas)
            }
            return DefaultBlockGasLimit
        }(),
        BlockNumber: big.NewInt(ctx.BlockHeight()),
        Time:        uint64(ctx.BlockHeader().Time.Unix()),
        Difficulty:  utils.Big0,
        BaseFee:     baseFee,
        BlobBaseFee: utils.Big1,
        Random:      &rh,
    }, nil
}
```

### 推荐修复步骤

1. **在 `x/evm/keeper/keeper.go` 中添加常量**：
   ```go
   const DefaultBlockGasLimit = 10000000  // 或其他合适的默认值
   ```

2. **修改 `GetVMBlockContext` 函数**：
   将 `GasLimit: gp.Gas()` 替换为：
   ```go
   GasLimit: func() uint64 {
       if ctx.ConsensusParams() != nil && ctx.ConsensusParams().Block != nil {
           return uint64(ctx.ConsensusParams().Block.MaxGas)
       }
       return DefaultBlockGasLimit
   }(),
   ```

3. **确保 RPC 一致性**：验证 RPC 端点返回的 `gasLimit` 也使用相同的值

---

## 总结对比

| 问题 | Tabi V3 现状 | Sei Chain 修复 |
|------|-------------|---------------|
| CumulativeGasUsed | 仅用哈希作为键，硬编码为0 | 用交易索引+哈希作为键，累积计算 |
| GASLIMIT 操作码 | 使用 `gp.Gas()` 交易池剩余量 | 使用 `ConsensusParams().Block.MaxGas` |

## 参考资料

- Sei Chain 源码：https://github.com/sei-protocol/sei-chain
- 相关文件：
  - `x/evm/keeper/receipt.go`
  - `x/evm/keeper/keeper.go`
  - `x/evm/types/keys.go`
