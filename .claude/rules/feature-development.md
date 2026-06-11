# 独自機能開発の規約 (fork 運用)

このリポジトリは mayswind/ezbookkeeping (本家) の fork で、本家のアップデートを追従しながら独自機能を載せて運用する。以下の規約に必ず従うこと。

## ブランチ運用

- `main` は本家 (upstream) の完全ミラー。**独自コミットは絶対に入れない**。更新は `git pull upstream main` のみ
  - この禁止は PreToolUse hook (`.claude/hooks/block-main-commit.sh`) でも強制されており、main 上での `git commit` / `merge` / `cherry-pick` / `revert` / `rebase` は自動的に deny される
- 運用ブランチは `custom/main`(本家リリースタグ + 独自機能の統合ブランチ)
- 機能追加は `custom/main` 起点で `feature/xxx` ブランチを切り、完成後 `custom/main` にマージする
- 本家の新リリース取り込みは `/sync-upstream` スキルの手順に従う(リリースタグ単位でマージ。日次の main 追従はしない)

## リモート構成

- `origin` = git@github.com:Mink16/ezbookkeeping.git(自分の fork。push 先)
- `upstream` = git@github.com:mayswind/ezbookkeeping.git(本家。fetch 専用、push 禁止)
- `gh` のデフォルトリポジトリは本家(`gh issue` / `gh pr` は本家を対象に動く)

## コミット規約

- 本家流に合わせる: 英語・小文字始まり・簡潔な現在形(例: `add transaction picture upload resolution setting`)

## 実装方針

- できるだけ**新規ファイル追加**で実装し、既存ファイルへの変更は最小限にする(将来の本家マージのコンフリクト面を最小化するため)
- i18n キーを追加する場合は `en` と `ja` のみに追加する(本家は全ロケール同時追加だが、独自機能では追加しない)
- 一般に有用な機能は `main` 起点のブランチに切り出して本家へ PR することを検討する(本家ルールは README の Contributing 節: fork + PR のみ)

## 品質チェック (custom/main へのマージ前に必須)

```bash
npm run lint    # vue-tsc + ESLint
npm test        # Vitest
go test ./...
```

## デプロイ反映

独自機能は自前イメージの再ビルドで反映する(本家イメージ `mayswind/ezbookkeeping` への差し替えでは独自機能が消える):

```bash
docker compose build && docker compose up -d
```
