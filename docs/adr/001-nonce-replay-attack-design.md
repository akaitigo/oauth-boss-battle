# ADR-001: Nonce Replay Attack (Boss 3) — ID Token Replay Prevention Design

## Status

Accepted

## Context

Boss 3 simulates an ID Token replay attack exploiting nonce reuse/absence in OpenID Connect.
The nonce parameter (OIDC Core §3.1.2.1) binds an ID Token to a specific authentication session.
Without proper nonce management, an attacker who obtains a valid ID Token can replay it in a
different session to impersonate the legitimate user.

Key design decisions:

1. **Nonce lifecycle**: How to manage generate → store → validate → consume
2. **ID Token simulation**: Whether to use real JWTs or simplified tokens
3. **Attack surface**: What specific replay scenarios to demonstrate

## Decision

### Nonce Store Design
- Server-side nonce store with in-memory map (`nonce → consumed` boolean)
- Nonces are one-time-use: validated then immediately marked consumed
- Cryptographically random generation (16 bytes, hex-encoded)

### ID Token Simulation
- Use simplified JWT-like tokens (header.payload.signature) with HS256
- Include standard OIDC claims: `iss`, `sub`, `aud`, `exp`, `iat`, `nonce`
- Server signs tokens with a shared secret (acceptable for educational simulation)
- Real JWT parsing/validation on the verification side

### Attack Scenarios
1. **No nonce**: Authorization request without nonce → ID Token without nonce claim → replay succeeds
2. **Nonce reuse**: Same nonce used across sessions → replay of captured ID Token succeeds
3. **Defense**: Unique nonce per session, consumed after validation → replay blocked

### API Design
- `POST /api/boss/3/authorize` — Start OIDC flow (with/without nonce)
- `POST /api/boss/3/token` — Exchange code for ID Token (includes nonce in claims)
- `POST /api/boss/3/replay` — Attempt to replay a captured ID Token
- `POST /api/boss/3/verify` — Boss defeat check

## Consequences

### Positive
- Educational: clearly demonstrates why nonce is required per OIDC Core §11.5
- Simplified JWT avoids external library dependency while teaching token structure
- One-time-use semantics prevent subtle replay bugs

### Negative
- HS256 is not recommended for production (RS256 preferred) — acceptable for education
- In-memory store loses state on restart — acceptable for demo application

### Risks
- Students may copy HS256 pattern to production: mitigated by explanatory text in UI
- Simplified token may not cover all real-world replay vectors: acceptable scope for MVP
