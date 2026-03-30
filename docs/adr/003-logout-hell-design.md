# ADR-003: Logout Hell (Boss 5) — Incomplete Logout and Session Persistence Design

## Status

Accepted

## Context

Boss 5 simulates the "Logout Hell" scenario where logging out of the Identity Provider (IdP)
does not properly invalidate sessions at Relying Parties (RPs). This is a common production
issue with federated identity, where:

- Front-channel logout relies on browser iframes/redirects that silently fail
- Missing back-channel logout leaves RP sessions active after IdP logout
- Users believe they are logged out but their sessions remain accessible

Key design decisions:

1. **Multi-RP simulation**: How to simulate multiple RPs with independent sessions
2. **Logout mechanism comparison**: Front-channel vs back-channel vs RP-initiated
3. **Session management**: How to track and visualize session state across RPs

## Decision

### Multi-RP Session Simulation
- Server-side session store tracking sessions per RP (up to 3 simulated RPs)
- Each RP has independent session state (active/invalidated)
- Session ID (sid) claim in ID Token binds sessions across IdP and RPs
- In-memory session store with sid-based lookup

### Logout Mechanisms
1. **Front-channel logout (vulnerable)**: IdP sends iframe requests to RP logout URLs
   - Simulated failure: iframes silently fail due to browser restrictions
   - Result: RP sessions remain active after IdP logout
2. **RP-Initiated Logout**: RP redirects user to IdP's end_session_endpoint
   - Only logs out the user at IdP, does not notify other RPs
3. **Back-channel logout (defense)**: IdP sends server-to-server POST with logout_token
   - Contains sid claim to identify which sessions to invalidate
   - Reliable: not dependent on browser state

### API Design
- `POST /api/boss/5/login` — Simulate login at IdP, create sessions at RPs
- `POST /api/boss/5/sessions` — Get current session state for all RPs
- `POST /api/boss/5/logout-frontchannel` — Simulate front-channel logout (fails)
- `POST /api/boss/5/logout-backchannel` — Simulate back-channel logout (succeeds)
- `POST /api/boss/5/verify` — Boss defeat check

## Consequences

### Positive
- Clearly demonstrates why front-channel logout is unreliable in practice
- Back-channel logout + sid pattern is directly applicable to production
- Multi-RP visualization makes session persistence visible and intuitive

### Negative
- Simplified simulation doesn't cover all edge cases (clock skew, network partitions)
- No actual iframe rendering for front-channel (text simulation instead)

### Risks
- Users may not realize the iframe failure is intentional: mitigated by clear UI messaging
- Session store is in-memory: acceptable for demo tool
