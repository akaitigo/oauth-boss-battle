# Plan: Issue #11 — Include header in JWT signature

## Summary
Fix JWT-style token signing to cover `base64url(header) + "." + base64url(payload)` per JWS (RFC 7515), so header tampering is detected.

## Tasks

- [x] 1. Refactor `MarshalToken()` in `jwks.go` — accept a signing function, build header internally, sign `headerB64.payloadB64`
- [x] 2. Refactor `ParseToken()` in `jwks.go` — return `signingInput` (headerB64.payloadB64) so callers can verify against it
- [x] 3. Update `boss4.go` call sites — pass signing function to `MarshalToken()`, use `signingInput` from `ParseToken()` for verification
- [x] 4. Remove `rebuildHeaderPayload()` from `boss4.go` — replaced by `ParseToken()` returning signing input
- [x] 5. Update existing tests in `jwks_test.go` and `boss4_test.go` to match new API
- [x] 6. Add regression test: header tamper detection (tamper `alg` field, verify must fail)
- [x] 7. `make check` passes

## API Changes

### `MarshalToken()` — before
```go
func MarshalToken(payload []byte, kid, signature string) string
```

### `MarshalToken()` — after
```go
type Signer func(data []byte) (string, error)
func MarshalToken(payload []byte, kid string, sign Signer) (string, error)
```

### `ParseToken()` — before
```go
func ParseToken(tokenStr string) (kid string, payload []byte, signature string, err error)
```

### `ParseToken()` — after
```go
func ParseToken(tokenStr string) (kid string, payload []byte, signingInput string, signature string, err error)
```

## Risks
- Existing tokens become invalid (acceptable: no persistent storage in this learning tool)
- Boss4 handler tests need updating (covered in task 5)
