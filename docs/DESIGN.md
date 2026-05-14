# nsite-cli 設計書

## 目的

`nsite-cli` は NIP-5A Static Websites / nsites の公開体験を Vim/CLI ユーザー向けに単純化するツールである。

対象は null--nostr のミニアプリタブに掲載・起動される小さな静的Webアプリで、ストア機能は本CLIでは扱わない。

## 参考思想

mattn の `algia` を参考にする。

- Go製の単一バイナリ
- `urfave/cli/v2` によるサブコマンド型CLI
- `~/.config/<app>/config.json` による最小構成
- `relays` と `privatekey` をJSONで管理
- Nostrの細かい機能を隠し、CLI体験を優先する

## スコープ

### やること

- 静的サイトプロジェクト生成
- ローカルHTTPサーバーでの確認
- ファイルのsha256計算
- NIP-5A manifest preview生成
- Blossomへのファイルアップロード
- NIP-5A manifest eventの署名とリレー投稿

### やらないこと

- アプリストア/Discover機能
- null--nostr側のUI実装
- サーバーサイド実行環境
- 課金・通知・ランキング

## 設定ファイル

場所:

```text
~/.config/nsite-cli/config.json
```

最小例:

```json
{
  "relays": {
    "wss://relay-jp.nostr.wirednet.jp": {
      "read": true,
      "write": true,
      "search": false
    }
  },
  "privatekey": "nsecXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "blossom": {
    "servers": ["https://blossom.example.com"]
  }
}
```

`algia` と互換性のある `relays` / `privatekey` 形式を採用し、NIP-5A用に `blossom.servers` を追加する。

簡易形式として top-level の `blossomServers` も読み込める。

## プロジェクト設定

各プロジェクトには `nsite.json` を置く。

```json
{
  "title": "Vim Cheatsheet",
  "description": "A tiny Vim command reference",
  "type": "named",
  "d": "vimcheat",
  "root": "public",
  "source": "https://github.com/example/vim-cheat"
}
```

- `type`: `root` または `named`
- `d`: named site用。NIP-5A canonical URLに合わせ `^[a-z0-9-]{1,13}$`、末尾 `-` 不可
- `root`: 配信対象ディレクトリ

## NIP-5A manifest生成

- root site: kind `15128`、`d`タグなし
- named site: kind `35128`、`["d", d]` 必須
- 全ファイルを走査し、`["path", "/absolute/path", "sha256"]` を生成
- `index.html` は必須
- `favicon.ico` と `404.html` は推奨
- configの `blossom.servers` を `server` タグに入れる
- `title`, `description`, `source` タグを付与する

## Blossomアップロード

各ファイルをsha256で識別し、設定されたBlossom serverへアップロードする。

初期実装では一般的なBUD-01/BUD-02系実装で使われる形式として、`PUT /upload` にファイル本体を送り、`Authorization: Nostr <base64(event-json)>` を付与する。

認証イベント:

- kind: `24242`
- content: `Upload <sha256>`
- tags:
  - `["t", "upload"]`
  - `["x", sha256]`
  - `["expiration", unix]`

サーバー差異に備え、失敗時はエラーを表示する。

## セキュリティ

- `privatekey` は nsec のみ受け付ける
- configファイルの権限は `0600` を推奨
- `doctor` で秘密鍵、リレー、Blossom、`d`タグ、`index.html` を検査する
- 公開前に manifest preview を表示する

## CLI

```text
nsite-cli init NAME [--d dtag]
nsite-cli dev [--addr :3128]
nsite-cli build
nsite-cli publish [--yes]
nsite-cli doctor
nsite-cli config path
```
