# ADR-002: JWKS Rotation Failure (Boss 4) — Key Rotation and Cache Strategy Design

## Status

Accepted

## Context

Boss 4 simulates JWKS key rotation failures that cause token verification outages.
In production, JWK Set endpoints (`/.well-known/jwks.json`) serve public keys for
token signature verification. Key rotation is critical for security hygiene, but
misconfigured caching or missing `kid` handling causes service disruptions.

Key design decisions:

1. **JWKS endpoint simulation**: How to simulate key rotation in a controlled way
2. **Cache strategy**: What caching behavior to demonstrate (TTL, kid-based refresh)
3. **Attack/failure scenarios**: Which rotation failures to reproduce

## Decision

### JWKS Endpoint Simulation
- In-memory key store supporting multiple active keys with unique `kid` values
- RS256 key pairs generated at runtime (educational; no pre-shared keys)
- Admin endpoints to trigger rotation, revoke keys, and control timing
- Each key has a `kid`, `use`, and `alg` metadata per RFC 7517

### Cache Strategy Demonstration
- Default cache: aggressive TTL, no kid-based refresh (vulnerable)
- Fixed cache: TTL + kid mismatch triggers immediate JWKS re-fetch
- Cache states exposed via API for visualization

### Failure Scenarios
1. **Stale cache**: Old key cached, new key used to sign → verification fails
2. **Premature revocation**: Old key revoked before all tokens expire → outage
3. **Defense**: kid-based cache refresh + overlap period for old/new keys

### API Design
- `POST /api/boss/4/jwks` — Get current JWKS (simulated endpoint)
- `POST /api/boss/4/rotate` — Trigger key rotation
- `POST /api/boss/4/sign` — Sign a token with current key
- `POST /api/boss/4/verify` — Verify token (with cache simulation)
- `POST /api/boss/4/configure-cache` — Set cache strategy (stale/smart)

## Consequences

### Positive
- Demonstrates real-world JWKS rotation pitfalls that cause production outages
- kid-based cache refresh pattern is directly applicable to production systems
- Timeline visualization makes the rotation overlap period intuitive

### Negative
- Simplified RSA implementation (smaller key size for speed) — acceptable for demo
- In-memory keys lost on restart — acceptable for educational tool
- Single-server simulation doesn't cover distributed cache invalidation

### Risks
- Key generation overhead: mitigated by pre-generating at startup
- Race conditions in rotation: mitigated by mutex-protected key store
