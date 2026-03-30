# Harvest Report: oauth-boss-battle

> 生成日: 2026-03-30
> リポジトリ: akaitigo/oauth-boss-battle

## プロジェクト概要

OAuth 2.0 / OIDC の代表的な攻撃パターンを「ボス戦」として体験し、防御実装を学ぶ教育用 Web アプリケーション。Go バックエンド + React フロントエンド + Ory Hydra 構成。全 5 ボスを実装し v1.0.0 リリース済み。

**ボス一覧:**

| # | ボス名 | Issue | PR | 状態 |
|---|--------|-------|----|------|
| 1 | PKCE Missing Attack | #1 | #6 | CLOSED / MERGED |
| 2 | State Mismatch (CSRF) | #2 | #7 | CLOSED / MERGED |
| 3 | Nonce Replay Attack | #3 | #8 | CLOSED / MERGED |
| 4 | JWKS Rotation Failure | #4 | #9 | CLOSED / MERGED |
| 5 | Logout Hell | #5 | #10 | CLOSED / MERGED |

## メトリクス

| メトリクス | 値 |
|------------|-----|
| Issue 数 | 5 (全 CLOSED) |
| PR 数 | 5 (全 MERGED) |
| Issue→PR 完了率 | 100% (5/5) |
| コミット数 (non-merge) | 7 |
| ADR 数 | 3 |
| Go テスト数 | 91 |
| Frontend テスト数 | 15 (5 files) |
| 総テスト数 | 106 |
| Go カバレッジ (token pkg) | 86.8% |
| CLAUDE.md 行数 | 45 |

## ハーネス適用状況

### Layer-0: リポジトリ衛生

| 項目 | 適用 | 備考 |
|------|------|------|
| CLAUDE.md (50行以下) | YES | 45行。ポインタ型で構造化 |
| .gitignore | YES | |
| .claudeignore | YES | コンテキスト汚染防止 |
| ADR 運用 | YES | 3件 (nonce-replay, jwks-rotation, logout-hell) |
| Makefile | YES | build/test/lint/format/check 統一 |
| docker-compose.yml | YES | Hydra + PostgreSQL + backend + frontend |

### Layer-1: 決定論的ツールによる品質強制

| 項目 | 適用 | 備考 |
|------|------|------|
| settings.json (Hooks) | YES | PreToolUse / PostToolUse / PreCompact / Stop |
| PreToolUse: lint設定変更ブロック | YES | Edit/Write に対するファイル名ガード |
| PreToolUse: 破壊的コマンドブロック | YES | rm -rf, --force, --no-verify 等 |
| PreToolUse: 機密ファイルブロック | YES | .env, credentials, *.pem 等 |
| PostToolUse: 自動lint | YES | post-lint.sh でファイル編集後にlint実行 |
| PreCompact: CLAUDE.md バックアップ | YES | コンパクト前に backup 作成 |
| Stop: 品質ゲート | YES | make check && make quality |
| lefthook.yml | YES | pre-commit: lint + format + test + archgate |
| startup.sh | YES | ツール自動インストール + ヘルスチェック |

### Layer-2: 計画と実行の分離

| 項目 | 適用 | 備考 |
|------|------|------|
| research.md → plan.md ワークフロー | YES | CLAUDE.md に明記、承認ゲート付き |
| Issue → PR 1:1 対応 | YES | 全5ボスが Issue→PR で対応 |
| ラベル運用 (mvp, model:opus/sonnet) | YES | タスク難易度でモデル選択 |
| セッション間状態管理 (git + Issues) | YES | CLAUDE.md に明記 |

### 適用サマリー

| レイヤー | 適用率 | 判定 |
|----------|--------|------|
| Layer-0: リポジトリ衛生 | 6/6 (100%) | FULL |
| Layer-1: 品質強制 | 9/9 (100%) | FULL |
| Layer-2: 計画/実行分離 | 4/4 (100%) | FULL |
| **総合** | **19/19 (100%)** | **FULL** |

## テンプレート改善提案

| # | 提案 | 根拠 | 優先度 |
|---|------|------|--------|
| 1 | hooks-structure.md テンプレートを Harness テンプレートに追加 | CLAUDE.md で参照されているが docs/hooks-structure.md が存在しない。Hooks 設計の文書化テンプレートがあると再利用しやすい | HIGH |
| 2 | ADR テンプレートに「Boss 2 以降で ADR が必要になった理由」パターンを記載 | Boss 1-2 は ADR なし、Boss 3 以降で ADR 追加。判断基準の明文化が有用 | MEDIUM |
| 3 | make quality ターゲットのテンプレート化 | Stop hook で `make quality` を呼んでいるが、テンプレートとして標準化すれば新 PJ での再現性が向上 | MEDIUM |
| 4 | テストカバレッジ閾値の Makefile テンプレート化 | Go 86.8% だが閾値指定がない。`go test -coverprofile` + 閾値チェックをテンプレートに組み込むべき | MEDIUM |
| 5 | startup.sh に `--ci` モードを追加 | CI 環境ではインタラクティブなツールインストールが不要。CI フラグで分岐するテンプレートが望ましい | LOW |
| 6 | CLAUDE.md 行数の自動チェック hook 追加 | 現在 45 行で制限内だが、膨張を防ぐ仕組みがない。PostToolUse で行数チェックを追加するテンプレート | LOW |

## 振り返り

### 良かった点

1. **Issue→PR 完了率 100%**: 5 Issue 全てが対応 PR でマージされ、未完了タスクがゼロ。計画と実行の分離が機能した
2. **ハーネス適用率 100%**: Layer-0/1/2 全項目が適用済み。テンプレートの再現性が高いことを実証
3. **コミット粒度の適切さ**: 7 non-merge コミットで 5 ボス + 初期セットアップ + リリース準備。1 ボス = 1 コミットの明確な粒度
4. **テスト 106 件 / カバレッジ 86.8%**: セキュリティ教育アプリとして十分なテストカバレッジ。攻撃パターンの再現テストが充実
5. **Hooks による品質ガード**: 機密ファイル編集ブロック、破壊的コマンドブロック、自動 lint が全て稼働。人間の介入なしに品質を維持
6. **ラベルによるモデル選択**: `model:sonnet` (Boss 1-2) と `model:opus` (Boss 3-5) で難易度に応じたモデル割り当てが機能

### 改善点

1. **hooks-structure.md が欠損**: CLAUDE.md で参照しているが実ファイルが存在しない。ドキュメントの整合性チェックが必要
2. **ADR の適用タイミングが不統一**: Boss 1-2 は ADR なし、Boss 3 以降で追加。ADR が必要な判断基準を事前に明文化すべき
3. **カバレッジ閾値未設定**: 86.8% は良好だが、閾値がないため将来的な低下を検知できない
4. **Frontend テスト比率が低い**: Go 91 件に対して Frontend 15 件。UI コンポーネントのテストが薄い可能性
5. **E2E テストの欠如**: Boss 戦の攻撃→防御フロー全体を通す E2E テストがない。教育アプリとしてはシナリオテストが有効
