# Changelog

## v1.0.0 (2026-03-30)

MVP release - OAuth/OIDC Boss Battle learning game with 5 bosses.

### Features

- **Boss 1: PKCE Missing Attack** - 認可コードフローでPKCE不在の脆弱性を体験・撃破 (#6)
- **Boss 2: State Mismatch (CSRF)** - stateパラメータ不在によるCSRF攻撃を再現・防御 (#7)
- **Boss 3: Nonce Replay Attack** - nonce再利用によるトークンリプレイ攻撃を体験・防御 (#8)
- **Boss 4: JWKS Rotation Failure** - JWKSキーローテーション失敗を再現・適切なキャッシュ戦略で撃破 (#9)
- **Boss 5: Logout Hell** - フロントチャネルログアウトの不完全実装を体験・Back-Channel Logoutで撃破 (#10)

### Infrastructure

- Go backend + React frontend + Docker Compose 開発環境
- ORY/Hydra連携による本格的なOAuth/OIDCフロー
- 全ボスにテストカバレッジ付き

### Commits

- dfaa01f feat: implement Boss 5 Logout Hell with session persistence simulation (#10)
- 40a0f2a feat: implement Boss 4 JWKS Rotation Failure with cache strategies (#9)
- ae6ba06 feat: implement Boss 3 Nonce Replay Attack with ID Token simulation (#8)
- cb8c0b2 feat: implement Boss 2 State Mismatch CSRF attack and defense (#7)
- 9d7fc3c feat: implement Boss 1 PKCE Missing Attack with full-stack foundation (#6)
- 697ee0a chore: initial project setup with Harness Engineering templates
