# MLS (RFC 9420) Implementation Plan

> Building a production-grade pure-Go RFC 9420 library directly in the server, extractable later as `go-mls`.

## Structure

```
internal/crypto/
├── mls/                      # Self-contained RFC 9420 implementation
│   ├── group.go             # MLS group state, epochs, tree
│   ├── key_schedule.go       # Key derivation (per RFC 9420 §7)
│   ├── encryption.go         # Encryption/decryption pipeline
│   ├── ciphersuite.go        # MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519
│   ├── credential.go         # UCP credential binding (identity + signing key)
│   ├── keypackage.go         # KeyPackage serialization/deserialization
│   ├── proposal.go           # Add/Remove/Update proposals
│   ├── commit.go             # Commit finalization
│   ├── welcome.go            # Welcome message for new members
│   ├── serialization.go      # TLS wire format encoding/decoding
│   ├── tree.go               # Binary tree operations
│   └── mls_test.go           # MLS-focused tests
├── manager.go                # Manager (current facade) → wraps MLS package
├── crypto.go                 # UCP integration layer
└── crypto_test.go            # Integration tests
```

## Isolation Rules

**MLS package (`internal/crypto/mls/`)**
- Pure RFC 9420 — no UCP types, no server logic
- No imports from `internal/` except stdlib + crypto libs
- No knowledge of threads, addresses, federation
- Testable in isolation with arbitrary group IDs and member lists

**Manager & integration (`internal/crypto/manager.go`, `crypto.go`)**
- Calls into `mls` package
- Maps UCP concepts → RFC 9420 primitives:
  - UCP thread IDs → MLS group IDs
  - UCP addresses → MLS member credentials
  - UCP signing keys → MLS credential fields
- Handles BCC groups, key share management, epoch sync with signing key rotation

## Implementation Phases

### Phase 1: Core Types & Serialization
- [ ] TLS wire encoding/decoding (`serialization.go`)
- [ ] Ciphersuite definition (`ciphersuite.go`)
- [ ] Credential binding to UCP identity (`credential.go`)
- [ ] KeyPackage structure and validation (`keypackage.go`)

### Phase 2: Group State Machine
- [ ] Group initialization and state (`group.go`)
- [ ] Binary tree for group members (`tree.go`)
- [ ] Epoch tracking and key schedule (`key_schedule.go`)

### Phase 3: Operations
- [ ] Encryption/decryption with key schedule (`encryption.go`)
- [ ] Add/Remove/Update proposals (`proposal.go`)
- [ ] Commit finalization (`commit.go`)

### Phase 4: New Member Onboarding
- [ ] Welcome message generation (`welcome.go`)
- [ ] Member Add flow (proposals → commit → welcome)

### Phase 5: Integration & Testing
- [ ] Wire up `Manager` to use `mls` package (replace AES-GCM mock)
- [ ] Integration tests with full group lifecycle
- [ ] Performance benchmarks
- [ ] Fuzzing for serialization robustness

## Dependencies

- `crypto/sha256`, `crypto/aes`, `crypto/cipher` (stdlib)
- `crypto/ed25519` (stdlib)
- `golang.org/x/crypto/hpke` or similar for HPKE (X25519 key agreement)

## Extraction Path (Future)

When `mls/` is production-ready:
1. Create new repo: `github.com/unifiedcommunicationsprotocol/go-mls`
2. Move `internal/crypto/mls/` → `go-mls/` (top-level package, no `internal/`)
3. Update server to import: `import "github.com/unifiedcommunicationsprotocol/go-mls"`
4. Release `go-mls` as a standalone library

No code changes needed — just a repo boundary move.

## Testing Strategy

- **Unit tests** in `mls/` — each file tests its primitives in isolation
- **Integration tests** in `crypto_test.go` — end-to-end group operations
- **Interop tests** — verify wire format against RFC 9420 test vectors (if available)
- **Property-based tests** — fuzzing for serialization round-trips

## Known Constraints

- **Ciphersuite**: `MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519` only (per UCP spec)
- **Credentials**: UCP-specific binding (identity key + signing key + address)
- **No roster privacy** — group members list is visible (not a concern for UCP's use case)

## Timeline Estimate

- Phase 1-2: ~4-6 weeks (types, tree, key schedule)
- Phase 3-4: ~3-4 weeks (encryption, add/remove, welcome)
- Phase 5: ~2-3 weeks (integration, benchmarks, docs)
- **Total: 2-3 months** for production-ready implementation

Parallel workstreams possible (e.g., integration layer while core is being built).
