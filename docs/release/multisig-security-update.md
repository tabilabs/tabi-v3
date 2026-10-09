# Multisig transaction security update

This update validates multisig transaction structure before signature verification.
Malformed transactions now return validation errors instead of triggering indexing
or nil-pointer panics. It checks outer signer counts, nested mode/signature counts,
bit-array bounds, signature counts and nested signature/key types.

The update also fixes bit counting at byte boundaries, including eight-key multisigs.
Previously accepted, correctly formed signatures retain their verification gas costs
and signature event encoding. Simulation placeholders with matching signer/signature
counts remain supported.

## Release checks

Release builds run the application wire-decoding regression and the SDK signature,
bit-array, multisig and ante-handler tests. The SDK tests cover malformed structures,
nested signatures, cryptographic verification and unchanged gas costs. The application
test uses the pinned SDK dependency rather than a locally substituted SDK checkout.

## Network rollout

This changes transaction validation outcomes. Validators must agree on an upgrade
height and use the same binary and pinned dependencies. Do not mix old and new
versions on an active network. This can be delivered with the pending Wasm runtime
security update, which also requires a coordinated upgrade.

No upgrade height or governance proposal is scheduled by this commit. Mainnet
deployment and replay validation against production chain data remain separate
release requirements.
