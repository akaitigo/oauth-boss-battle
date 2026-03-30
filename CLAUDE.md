# oauth-boss-battle
<!-- メンテナンス指針: 各行に問う「この行を消したらエージェントは間違えるか？」→ No なら削除 -->

## コマンド
- ビルド: `make build`
- テスト: `make test`
- lint: `make lint`
- フォーマット: `make format`
- 全チェック: `make check`
- 開発環境: `docker compose up -d`

## ワークフロー
1. research.md を作成（調査結果の記録）
2. plan.md を作成（実装計画。人間承認まで実装禁止）
3. 承認後に実装開始。plan.md のtodoを進捗管理に使用

## ルール
- ADR: docs/adr/ 参照。新規決定はADRを書いてから実装
- テスト: 機能追加時は必ずテストを同時に書く
- lint設定の変更禁止（ADR必須）
- critical ruleは本ファイルの先頭に配置（earlier-instruction bias対策）

## 構造
- backend/: Go API サーバー（Hydra連携、ボスロジック）
- frontend/: React + Canvas/WebGL（ゲームUI）
- docker-compose.yml: Hydra + PostgreSQL + backend + frontend

## 禁止事項
- any型(TS) / !!(Kotlin) / unwrap(Rust) → 各言語ルール参照
- console.log / print文のコミット
- TODO コメントのコミット（Issue化すること）
- .env・credentials のコミット
- lint設定の無効化（ルール単位の disable 含む）

## Hooks
- 設定: .claude/hooks/ 参照
- 構造定義: docs/hooks-structure.md 参照

## 状態管理
- git log + GitHub Issues でセッション間の状態を管理
- セッション開始: `bash startup.sh`（ツール自動インストール + ヘルスチェック）

## コンテキスト衛生
- .gitignore / .claudeignore で不要ファイルを除外
- バイナリ、キャッシュ、node_modules等がコンテキストを汚染しないこと
