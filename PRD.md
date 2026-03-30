# PRD: oauth-boss-battle

## 概要

OAuth/OIDCの失敗系を「倒す」ブラウザゲーム兼学習ツール。
Hydraや擬似IdPを相手に、PKCE抜け、`state`不一致、`nonce`再利用、JWKSローテーション失敗、logout地獄を再現する。

## 背景と動機

- OAuth/OIDCの仕様は複雑で、テキストベースの学習では理解が浅くなりがち
- 失敗系（セキュリティ脆弱性）を体感的に学べるツールが存在しない
- セキュリティカンファレンスでのデモ用途にも最適

## ユーザーペルソナ

1. **Web開発者**: OAuth実装時の落とし穴を事前に学びたい
2. **セキュリティエンジニア**: 攻撃ベクターを実際に体験したい
3. **カンファレンス参加者**: インタラクティブなデモで理解を深めたい

## 技術スタック

- **Backend**: Go
- **IdP**: ORY/Hydra (Docker)
- **Frontend**: React + Canvas/WebGL
- **Database**: PostgreSQL
- **Infrastructure**: Docker Compose (dev), GCP Cloud Run (prod)

## MVP機能（受け入れ条件）

### Boss 1: PKCE Missing Attack
- [x] 認可コードフローでPKCEなしのリクエストを送信できる
- [x] サーバーがcode_verifier不在を検出し、攻撃成功を表示する
- [x] 正しいPKCE実装を行うと「ボス撃破」判定になる
- [x] 攻撃の解説と防御方法がゲーム内で表示される

### Boss 2: State Mismatch (CSRF)
- [x] state パラメータなしの認可リクエストを再現できる
- [x] CSRF攻撃シナリオが視覚的に表現される
- [x] 正しいstate検証を実装すると「ボス撃破」判定になる
- [x] 攻撃フローの図解が表示される

### Boss 3: Nonce Replay Attack
- [x] nonce再利用によるトークンリプレイ攻撃を再現できる
- [x] ID Tokenのnonce検証の重要性が体験できる
- [x] 正しいnonce検証を実装すると「ボス撃破」判定になる

### Boss 4: JWKS Rotation Failure
- [x] JWKSエンドポイントのキーローテーション失敗を再現できる
- [x] 古い鍵で署名されたトークンの検証失敗が体験できる
- [x] 適切なJWKSキャッシュ戦略を実装すると「ボス撃破」判定になる

### Boss 5: Logout Hell
- [x] フロントチャネルログアウトの不完全実装を再現できる
- [x] セッション残存によるセキュリティリスクが体験できる
- [x] 正しいRP-Initiated Logout + Back-Channel Logoutを実装すると「ボス撃破」判定になる

### 共通要件
- [x] 各ボスに挑戦履歴がブラウザに保存される（localStorage）
- [x] ゲーム画面にボス一覧（進捗付き）が表示される
- [x] Docker Compose一発で開発環境が起動する
- [x] 各ボスの解説ページに RFC/仕様書へのリンクがある

## 非機能要件

- レスポンスタイム: ゲーム操作は200ms以内
- Docker Composeで3分以内に起動完了
- モバイル非対応（デスクトップブラウザ専用）

## スコープ外

- ユーザー認証（ゲーム自体のログイン機能）
- マルチプレイヤー
- スコアのサーバーサイド永続化
- モバイル対応

## 成功指標

- 5つのボスが全て動作する
- Docker Compose起動から最初のボス挑戦まで5分以内
- カンファレンスデモで30秒以内にコンセプトが伝わる
