# Research: Issue #11 — JWT header not included in signature

## Problem

JWT-style tokens in `MarshalToken()`/`ParseToken()` sign only the payload bytes, not `base64url(header) + "." + base64url(payload)` as JWS (RFC 7515) requires. This means header tampering (e.g., changing `alg` from `RS256` to `none`) goes undetected.

## Affected Code

### 1. `backend/internal/jwks/jwks.go`

#### `MarshalToken()` (line 264)
- Receives pre-computed `signature` (signed against raw `payload` bytes only)
- Assembles `headerB64 + "." + payloadB64 + "." + signature`
- **Bug**: The signature was computed before this function is called, and it only covers payload

#### `Sign()` / `SignWithKid()` (lines 132, 151)
- `Sign()` receives `data []byte` and signs it directly
- Callers pass raw `payloadBytes` — never `header.payload`
- These functions are general-purpose and themselves are correct; the problem is the **call site**

### 2. `backend/internal/boss/boss4.go`

#### Call sites in `Sign()` handler (lines 153, 171)
- `h.keyStore.SignWithKid(payloadBytes, oldKid)` — signs only payload
- `h.keyStore.Sign(payloadBytes)` — signs only payload
- Then `jwks.MarshalToken(payloadBytes, oldKid, sig)` — header is created inside MarshalToken but never signed

#### `rebuildHeaderPayload()` (line 321)
- Despite its name, it returns only decoded payload bytes, not `header.payload`
- This means verification also checks only payload bytes against the signature

#### `Verify()` handler (lines 237, 274)
- Calls `rebuildHeaderPayload()` which returns raw payload
- Passes that to `VerifyWithKid()` — so verification succeeds even if header was tampered

## JWS Specification (RFC 7515)

The JWS Signing Input is: `ASCII(BASE64URL(UTF8(JWS Protected Header)) || '.' || BASE64URL(JWS Payload))`

The signature MUST cover the base64url-encoded header AND payload, joined by a dot.

## Fix Strategy

### Option A: Move signing responsibility into `MarshalToken()`
- `MarshalToken()` takes `KeyStore`, builds header, computes `headerB64.payloadB64`, signs it, returns complete token
- Pro: Single responsibility, impossible to forget header in signing input
- Con: Changes `MarshalToken()` API significantly, breaks existing callers and tests

### Option B: Keep `Sign()`/`SignWithKid()` general-purpose, fix call sites
- Callers construct `headerB64 + "." + payloadB64` and pass that as the data to sign
- `MarshalToken()` takes the pre-built `headerB64.payloadB64` string and appends signature
- `rebuildHeaderPayload()` returns `headerB64 + "." + payloadB64` (the first two dot-separated parts)
- Pro: Minimal API change, `Sign()`/`SignWithKid()` remain reusable
- Con: Callers must remember to include header in signing input

### Decision: Option B (modified)
- Refactor `MarshalToken()` to build header internally, compute signing input, and accept a signing function
- Actually, simplest: have `MarshalToken()` accept `KeyStore` (or a signing func) so it controls the full flow
- Even simpler: **Change `MarshalToken()` to sign `headerB64 + "." + payloadB64`** instead of raw payload

Concrete approach:
1. `MarshalToken()` builds header, base64-encodes header and payload, signs `headerB64.payloadB64` via a passed signer, assembles the token
2. `ParseToken()` returns `signingInput` (headerB64.payloadB64) alongside kid, payload, signature
3. Verify handler passes `signingInput` to `VerifyWithKid()` instead of raw payload
4. Remove `rebuildHeaderPayload()` — no longer needed
