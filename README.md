# nsite-cli

`nsite-cli` は、[`NIP-5A`](https://github.com/nostr-protocol/nips/blob/master/5A.md)  / nsite 静的Webサイトを **作成・ローカル確認・ビルド・Blossomアップロード・Nostr公開** するための小さなGo製CLIです。

Vimユーザーが `vim` とCLIだけで小さな静的サイト/ミニアプリを作り、ローカルで確認し、そのままNIP-5A形式で公開できる体験を目指しています。

mattnさんの [`algia`](https://github.com/mattn/algia) を参考に、以下の方針で作っています。

- Go製の単一バイナリ
- `~/.config/nsite-cli/config.json` に設定を置く
- JSONでリレー、秘密鍵、Blossom server、nsite hostを設定
- サブコマンド中心のシンプルなCLI
- Nostr / Blossom / NIP-5A の細かい処理をCLIで隠蔽

> [!CAUTION]
> `privatekey` に本番アカウントの `nsec` を直接入れる場合は十分注意してください。テスト中は専用の開発用鍵を使うことを推奨します。

---

## インストール / ビルド

```bash
git clone https://github.com/tami1A84/nsite-cli.git
cd nsite-cli
make build
```

または:

```bash
go build -o nsite-cli ./cmd/nsite-cli
```

---

## 設定ファイル

設定ファイルの場所:

```bash
~/.config/nsite-cli/config.json
```

雛形を作成:

```bash
./nsite-cli config init
```

例:

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
    "servers": [
      "https://blossom.example.com"
    ]
  },
  "nsite": {
    "host": "nsite.example.com"
  }
}
```

簡易形式も利用できます。

```json
{
  "relays": {},
  "privatekey": "nsec...",
  "blossomServers": ["https://blossom.example.com"],
  "nsite": {
    "host": "nsite.example.com"
  }
}
```

---

## configコマンド

設定ファイルのパスを確認:

```bash
./nsite-cli config path
```

nsite hostを設定:

```bash
./nsite-cli config set nsite.host nsite.example.com
```

確認:

```bash
./nsite-cli config get nsite.host
```

削除:

```bash
./nsite-cli config unset nsite.host
```

Blossom serverを設定:

```bash
./nsite-cli config set blossom.servers https://blossom.primal.net
```

複数指定:

```bash
./nsite-cli config set blossom.servers https://blossom.primal.net,https://example.com
```

秘密鍵を設定:

```bash
./nsite-cli config set privatekey nsec1...
```

---

## 基本的な使い方

プロジェクト作成:

```bash
./nsite-cli init vim-cheat --d vimcheat
cd vim-cheat
```

ローカル確認:

```bash
../nsite-cli dev
```

デフォルトでは以下で起動します。

```text
http://localhost:3128
```

NIP-5A manifest previewを生成:

```bash
../nsite-cli build
```

出力:

```text
.nsite/manifest-preview.json
```

公開:

```bash
../nsite-cli publish
```

確認を省略する場合:

```bash
../nsite-cli publish -y
```

---

## コマンド一覧

```text
init      新しいnsiteプロジェクトを作成
dev       public/ をローカルHTTPサーバーで配信
build     ファイルのSHA-256を計算し、NIP-5A manifest previewを生成
publish   ファイルをBlossomへアップロードし、kind 15128 / 35128をNostrリレーへpublish
inspect   event id / nevent からNIP-5Aイベントを確認
doctor    config、プロジェクト、リレー、Blossom取得可否を確認
config    config.jsonの作成・参照・更新
```

---

## NIP-5A対応

`nsite-cli` は以下のNIP-5Aイベントを生成します。

- root site: `kind: 15128`
- named site: `kind: 35128`

named siteでは `d` tagを使います。

```json
["d", "vimcheat"]
```

静的ファイルは `path` tagとしてmanifestに入ります。

```json
["path", "/index.html", "<sha256>"]
```

Blossom serverは `server` tagとして入ります。

```json
["server", "https://blossom.example.com"]
```

---

## canonical URL / pubkeyB36

NIP-5A named siteのcanonical host labelは以下です。

```text
<pubkeyB36><dTag>
```

`pubkeyB36` は、32-byte raw pubkeyをbase36に変換し、50文字にゼロ埋めした値です。

`publish` や `inspect` では以下を表示します。

```text
canonical:
  pubkeyB36: ...
  host label: <pubkeyB36><dTag>
  url: https://<pubkeyB36><dTag>.<nsite.host>/
```

`nsite.host` が未設定の場合は、以下で設定できます。

```bash
./nsite-cli config set nsite.host nsite.example.com
```

---

## inspect

公開済みイベントを確認:

```bash
./nsite-cli inspect <event-id-or-nevent>
```

raw JSONを表示:

```bash
./nsite-cli inspect <event-id-or-nevent> --json
```

一時的にhostを指定:

```bash
./nsite-cli inspect <event-id-or-nevent> --host nsite.example.com
```

---

## doctor

```bash
./nsite-cli doctor
```

確認内容:

- config path
- `privatekey` が有効な `nsec` か
- read/write relay
- Blossom server
- `nsite.json`
- manifest生成
- relay接続
- Blossom上で各blobを取得できるか

ネットワーク確認を飛ばす場合:

```bash
./nsite-cli doctor --online=false
```

---

## プロジェクト構成

```text
nsite-cli/
  cmd/nsite-cli/        CLI entrypoint
  internal/config/      config.json読み書き
  internal/project/     nsite.jsonとプロジェクト雛形
  internal/nip5a/       NIP-5A manifest生成
  internal/blossom/     Blossom upload/check
  docs/                 設計書・計画
  examples/             config例
```

---

## ドキュメント

- [`docs/DESIGN.md`](docs/DESIGN.md)
- [`docs/PLAN.md`](docs/PLAN.md)

---

# English

`nsite-cli` is a small Go CLI for creating, testing, building, and publishing NIP-5A / nsite static websites.

It is intentionally modeled after mattn's [`algia`](https://github.com/mattn/algia): simple JSON config under `~/.config`, small subcommands, and Nostr-first publishing.

## Quick start

```bash
git clone https://github.com/tami1A84/nsite-cli.git
cd nsite-cli
make build
./nsite-cli config init
./nsite-cli init vim-cheat --d vimcheat
cd vim-cheat
../nsite-cli dev
../nsite-cli build
../nsite-cli publish
```

## Config

Create `~/.config/nsite-cli/config.json`:

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
    "servers": [
      "https://blossom.example.com"
    ]
  },
  "nsite": {
    "host": "nsite.example.com"
  }
}
```

Shortcut form is also accepted:

```json
{
  "relays": {},
  "privatekey": "nsec...",
  "blossomServers": ["https://blossom.example.com"],
  "nsite": {
    "host": "nsite.example.com"
  }
}
```

## Commands

- `init`: create a minimal static app project
- `dev`: serve project locally
- `build`: hash files and generate a NIP-5A manifest preview
- `publish`: upload files to Blossom and publish kind `15128` or `35128`
- `inspect`: inspect a NIP-5A event by event id or nevent
- `doctor`: check config, project, relays, and Blossom availability
- `config`: create, read, and update config values

See `docs/DESIGN.md` and `docs/PLAN.md`.
