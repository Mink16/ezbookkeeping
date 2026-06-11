---
name: sync-upstream
description: 本家 (mayswind/ezbookkeeping) の新リリースを custom/main に取り込む手順。ユーザーが「本家に追従したい」「アップデートを取り込みたい」「新バージョンが出た」と言ったときに使用する。
---

# 本家リリースの取り込み手順

本家の新リリースタグを `custom/main` にマージし、独自機能を維持したままバージョンアップする。

## 手順

1. **新タグの確認**

   ```bash
   git fetch upstream --tags
   git tag --sort=-creatordate | head -5
   ```

   取り込むのは**リリースタグ単位**(例: `v1.7.0`)。日次の `upstream/main` を直接マージしない。

2. **作業ツリーの確認**

   `git status` がクリーンであること。未コミットの変更があれば先にコミットまたは stash する。

3. **main ミラーの更新**

   ```bash
   git checkout main && git pull upstream main
   ```

4. **custom/main にタグをマージ**

   ```bash
   git checkout custom/main
   git merge vX.Y.Z
   ```

5. **コンフリクト解消の指針**

   - 独自機能は新規ファイル中心のため、衝突は主に i18n ファイル・ルーティング・既存ファイルへのフック箇所で起きる
   - 原則「本家の変更を受け入れた上で、独自機能のフックを再適用する」方向で解消する
   - 判断に迷う衝突はユーザーに確認する

6. **検証**

   ```bash
   npm run lint && npm test && go test ./...
   ```

7. **再デプロイ**

   ```bash
   docker compose build && docker compose up -d
   ```

   起動後、http://localhost:8081/ で動作確認する。

8. **push**

   ```bash
   git push origin custom/main
   ```
