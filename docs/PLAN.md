# nsite-cli 実装プラン

## Phase 0: プロジェクト作成

- Go module作成
- `urfave/cli/v2` 導入
- `algia` 風 config loader作成
- ドキュメント作成

## Phase 1: ローカル開発

- `init` で `nsite.json` と `public/` を生成
- `dev` でローカル配信
- `/` とディレクトリパスを `index.html` にフォールバック

## Phase 2: manifest preview

- `build` でファイル走査
- sha256計算
- `.nsite/manifest-preview.json` 出力
- NIP-5Aのkind、tagsを検証

## Phase 3: publish

- configからnsec decode
- Blossom upload
- kind `15128` / `35128` event生成
- 書き込みリレーへpublish
- 公開URL候補とevent idを表示

## Phase 4: 改善

- NIP-46 remote signer対応
- BUD-03 user servers event publish対応
- 複数Blossomへのミラー状況表示
- `inspect` コマンド
- null--nostrミニアプリタブとの連携メタデータ出力
