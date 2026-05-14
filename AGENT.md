# AGENT.md

このリポジトリでAIエージェントが作業するときのメモです。

## プロジェクト概要

`nsite-cli` は、NIP-5A / nsite 静的Webサイトを作成・ローカル確認・ビルド・Blossomアップロード・Nostr公開するGo製CLIです。

初期ターゲットはVim/Vimmer向けの小さな静的ミニアプリ公開体験です。

## 重要な仕様

- NIP-5A root site: `kind 15128`
- NIP-5A named site: `kind 35128`
- named siteは `d` tag必須
- canonical named-site labelは `<pubkeyB36><dTag>`
- `pubkeyB36` は32-byte pubkeyをbase36化し、50文字に左ゼロ埋めする
- ファイル本体はBlossomへアップロードする
- Nostr eventには `path` tagで `/path -> sha256` を入れる

## 設定ファイル

ユーザー設定は以下です。

```text
~/.config/nsite-cli/config.json
```

秘密鍵 `privatekey` に `nsec` が入るため、ユーザーの実設定ファイルをリポジトリに追加してはいけません。

## コマンド

```bash
make build
go test ./...
./nsite-cli --help
```

MVPコマンド:

- `init`
- `dev`
- `build`
- `publish`
- `inspect`
- `doctor`
- `config`

## コーディング方針

- Goの標準的な小さいパッケージ分割を維持する
- CLIは `urfave/cli/v2`
- Nostr処理は `github.com/nbd-wtf/go-nostr`
- ユーザー向け出力は簡潔にする
- `nsec` や個人設定をログ・README・テストデータに残さない
- `go test ./...` と `make build` が通ることを確認する

## ディレクトリ

```text
cmd/nsite-cli/        CLI entrypoint
internal/config/      config.json読み書き
internal/project/     nsite.jsonとプロジェクト雛形
internal/nip5a/       NIP-5A manifest生成
internal/blossom/     Blossom upload/check
docs/                 設計書・計画
examples/             config例
```

## 注意

- `vim-cheat/` や `vim-cheat2/` はローカル検証用プロジェクトなのでコミットしない
- ビルド済みバイナリ `nsite-cli` はコミットしない
- `.nsite/` は生成物なのでコミットしない
