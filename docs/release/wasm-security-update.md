# Wasm runtime security update

This update moves the active Wasm runtime to 1.5.9 and includes the corresponding
Linux and macOS libraries. It addresses contract execution failures and excessive
execution time covered by CWA-2025-001 and CWA-2025-002, and the tracing-related
consensus issue covered by CWA-2025-003. Existing custom staking queries are retained.

## Upgrade requirements

This is a consensus-breaking update. Validators must switch at the agreed upgrade
height. Do not deploy it as a rolling upgrade or mix runtime versions on an active
network. The upgrade height and plan name must be agreed separately through the
network's normal upgrade process; this code change does not schedule an upgrade.

Ship the node binary and its matching library bundle together. Replacing only the
binary, or only changing the Go dependency, is insufficient. Confirm the library
loaded by the installed binary before starting the node:

```sh
./tabid query wasm libwasmvm-version
```

The result must be `1.5.9`. Release bundle smoke checks enforce this on Linux amd64
and macOS arm64. Release builds also run the application contract lifecycle and
runtime version tests. The historical v152 and v155 query libraries remain unchanged;
they are not the active transaction execution runtime.

Before production rollout, run the release build and validate contract upload,
instantiation, execution, queries and block replay against an isolated copy of
chain data. Mainnet deployment and replay acceptance are not implied by this commit.

## References

- [CWA-2025-001](https://github.com/CosmWasm/advisories/blob/main/CWAs/CWA-2025-001.md)
- [CWA-2025-002](https://github.com/CosmWasm/advisories/blob/main/CWAs/CWA-2025-002.md)
- [CWA-2025-003](https://github.com/CosmWasm/advisories/blob/main/CWAs/CWA-2025-003.md)
