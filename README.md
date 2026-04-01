# oauth-boss-battle

OAuth/OIDCの失敗系を「ボス」として倒していくブラウザゲーム兼学習ツール。PKCE抜け、state不一致、nonce再利用、JWKSローテーション失敗などの脆弱性を体感で学べます。

## 特徴

- PKCE Missing Attack、State Mismatch (CSRF)、Nonce Replay、JWKS Rotation Failure の4ボスを収録
- 攻撃シナリオの視覚的な再現と、正しい実装による「ボス撃破」判定
- 各攻撃の解説・防御方法・フロー図解をゲーム内で表示

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| バックエンド | Go |
| フロントエンド | TypeScript, React, Vite, Canvas/WebGL |
| IdP | ORY Hydra (Docker) |
| データベース | PostgreSQL |
| インフラ | Docker Compose (dev), GCP Cloud Run (prod) |

## アーキテクチャ

```
[Browser] → [React SPA (Vite)] → [Go API Server] → [ORY Hydra]
                                                  → [PostgreSQL]
```

フロントエンドはCanvas/WebGLでゲームUIを描画し、バックエンドのGo APIがHydra連携とボスロジックを担当します。

## Quick Start

### 前提条件

- Go 1.23+
- Node.js 22+
- Docker & Docker Compose

### 1. Clone & Setup

```bash
git clone https://github.com/akaitigo/oauth-boss-battle.git
cd oauth-boss-battle
```

### 2. Start Dependencies

```bash
docker compose up -d

# 全サービスが起動していることを確認
docker compose ps
```

| Service | Port | Purpose |
|---------|------|---------|
| PostgreSQL | `5432` | データストア |
| ORY Hydra (Public) | `4444` | OAuth2 エンドポイント |
| ORY Hydra (Admin) | `4445` | 管理API |
| Backend | `8080` | Go API サーバー |
| Frontend | `3000` | React SPA |

### 3. Build & Run (ローカル開発)

```bash
# バックエンド
cd backend
go run ./cmd/server

# フロントエンド（別ターミナル）
cd frontend
npm install
npm run dev
```

### 4. Run Tests

```bash
# 全体チェック（format → tidy → lint → test → build）
make check
```

## 開発

```bash
# 個別コマンド
make build     # ビルド
make test      # テスト
make lint      # lint
make format    # フォーマット
make quality   # 品質チェック
```

## License

MIT
