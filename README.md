# studyclaw

`studyclaw` はマルチアカウント接続、Web管理、プッシュ通知、設定管理を備えた学習コンソールです。メインフローは安定性を重視し、以下の2つに収斂しています：

1. 記事学習
2. 音声学習

日々の回答、週末回答、専門回答のコードはプロジェクト内に残っていますが、デフォルトのスコアフローには接続されていません。

## プロジェクト概要

- `記事 + 音声` を唯一の正式学習フローとし、不安定な要素を排除。
- 旧プロジェクトのWeb管理、プッシュ通知、スケジュール、マルチアカウント管理機能を維持。
- 正式エントリポイント `/studyclaw/` を提供、デプロイ後すぐに他ユーザーと共有可能。
- シンプルなWebコンソールと明確な完了通知フォーマットを提供。

## 現在の方針

- 正式フローでは日々の回答を実行しない。
- ログインポイントはメインフローに含まれない。
- 完了通知には総スコア、当日の獲得点数、所要時間、プログラムで学習可能な3項目の進捗が含まれる。
- WebUIはシンプルなスタイルに変更済み。

## クイックスタート

### Docker Compose

```bash
docker compose up -d
```

デフォルト設定：

- `8080` ポートでWebサービスを起動
- `./config` をコンテナ内の `/opt/config` にマウント
- `ghcr.io/legolasljl/studyclaw:latest` を使用

### バイナリ実行

設定の初期化：

```bash
./studyclaw --init
```

プログラムの起動：

```bash
./studyclaw
```

### ソースコードからの実行

```bash
go mod tidy
go build -o studyclaw ./
./studyclaw --init
./studyclaw
```

## Webエントリ

以下の正式エントリを使用してください：

- `/studyclaw/`

ルートパス `/` もサポートしており、自動的に `/studyclaw/` にリダイレクトされます。

管理者ログインの認証情報は `config/config.yml` 内の以下の項目から取得されます：

- `web.account`
- `web.password`

一般ユーザーは `web.common_user` で管理され、マルチアカウントに対応していますが、自分のデータのみ閲覧可能です。

## 設定手順

### 1. 設定の初期化

初回実行時に `./studyclaw --init` を実行するか、コンテナを初めて起動すると `config/config.yml` が生成されます。

### 2. Web管理アカウントの設定

```yaml
web:
  enable: true
  host: 0.0.0.0
  port: 8080
  account: admin
  password: admin
  common_user:
    user1: password1
    user2: password2
```

### 3. 学習リズムの設定

```yaml
study:
  recent_days: 180
  fallback_to_all: true
  article_duration_seconds: 90
  article_duration_jitter_seconds: 20
  audio_duration_seconds: 80
  audio_duration_jitter_seconds: 15

cron: "0 0 * * *"
cron_random_wait: 0
```

### 4. プッシュ通知の設定

`conf/config_default.yml` に安全なサンプルが含まれています。実際のデプロイ時に必要なセクションを `config/config.yml` に記入してください。

現在のビルトインサンプル：

- DingTalk
- PushPlus
- Telegram
- WeChat公式アカウント（テスト用）
- QQ / go-cqhttp
- PushDeer
- JPush

#### Telegram

設定例：

```yaml
tg:
  enable: true
  chat_id: 123456789
  token: "123456789:telegram_bot_token"
  proxy: ""
  custom_api: "https://api.telegram.org"
  white_list:
    - 123456789
```

フィールド説明：

- `enable`：Telegramプッシュ通知とインタラクティブコマンドを有効にするかどうか。
- `chat_id`：通知を受け取るデフォルトの個人またはグループID。
- `token`：`@BotFather` でボットを作成した際に取得したトークン。
- `proxy`：オプション。Telegramに接続できない環境の場合、プロキシURLを指定。
- `custom_api`：オプション。自前のTelegram APIリバースプロキシを使用する場合にURLを置換。
- `white_list`：ボットにコマンドを送信できるchat IDのリスト。最低限自分の `chat_id` を含めることを推奨。

設定手順：

1. Telegramで `@BotFather` を使用してボットを作成し、`token` を取得。
2. ボットに `/start` を送信して会話を開始。
3. ボットまたは他のツールで自分の `chat_id` を確認。
4. `chat_id` を `chat_id` と `white_list` の両方に記入。
5. プログラムを再起動し、`/ping` でボットがオンラインか確認。

#### DingTalk

設定例：

```yaml
push:
  ding:
    enable: true
    access_token: "your_ding_access_token"
    secret: "SECxxxxxxxxxxxxxxxxxxxxxxxx"
```

フィールド説明：

- `enable`：DingTalkグループロボットプッシュを有効にするかどうか。
- `access_token`：グループロボットのwebhook内のトークン。完全なwebhook URLではなくトークン部分のみを記入。
- `secret`：ロボットが「署名」を有効にしている場合、対応する `SEC...` キーを記入。有効にしていない場合は空文字列のまま。

設定手順：

1. DingTalkグループにカスタムロボットを追加。
2. webhook内の `access_token` を記録。
3. セキュリティ設定で「署名」を有効にした場合、`secret` も記入。
4. プログラムを再起動し、起動通知が正常に届くか確認。

#### WeChat公式アカウント（テスト用）

設定例：

```yaml
wechat:
  enable: true
  token: "your_wechat_token"
  secret: "your_wechat_secret"
  app_id: "wx1234567890abcdef"
  login_temp_id: "wechat_login_template_id"
  normal_temp_id: "wechat_normal_template_id"
  push_login_warn: true
  super_open_id: "openid_example"
```

フィールド説明：

- `enable`：WeChat公式アカウントテスト号プッシュを有効にするかどうか。
- `token`：WeChatテスト号の「インターフェース設定」内のトークン。
- `secret`：WeChatテスト号の `appsecret`。
- `app_id`：WeChatテスト号の `appID`。
- `login_temp_id`：ログイン/認証フロー用のテンプレートメッセージID。
- `normal_temp_id`：通常の学習通知用のテンプレートメッセージID。
- `push_login_warn`：cookieが無効になった場合にプッシュ通知を送信するかどうか。
- `super_open_id`：管理者自身のopenid。管理レベルのメッセージを受信するために使用。

設定手順：

1. WeChat公式プラットフォームのテスト号ページでテスト号を作成。
2. テスト号の管理画面で `app_id`、`secret`、`token` を取得。
3. ログイン通知と通常通知のテンプレートを作成し、`login_temp_id`、`normal_temp_id` を記入。
4. WeChatでテスト号をフォローし、自分の `openid` を取得して `super_open_id` に記入。
5. デプロイ先がWeChatプラットフォームからコールバック可能であることを確認し、プログラムを再起動して検証。

### 5. 学習アカウントの接続

1. Webコンソールを開く。
2. 管理者アカウントでログイン。
3. 「ユーザー管理」に移動。
4. 認証QRコードを生成。
5. 学習強国アプリでスキャンして接続。
6. アカウント一覧またはスコア照会で状態を確認。

## 完了通知フォーマット

```text
xxアカウント 学習完了
現在の総学習スコア：
当日獲得スコア：
今回の所要時間：
記事学習： /12
動画学習： /12
日々の回答： /5
プログラム学習可能3項目： /29
```

説明：

- `当日獲得スコア` は学習強国のその日の全獲得項目の合計であり、`29` を超える場合があります。
- プログラムで直接完了できる3項目は `記事学習 12`、`動画学習 12`、`日々の回答 5` で、合計 `29` 点です。
- ログイン、購読、専門などのスコアを他の方法で取得した場合、`当日獲得スコア` は増加しますが、プログラムのこの3項目の達成判定には影響しません。

## Dockerデプロイ

コンテナデプロイ時の主要なポイント：

- `-p 8080:8080` でWebエントリに対応
- `./config:/opt/config` で `config.yml`、`user.db`、ログを保持
- コンテナを再構築する場合、ホストマシンの `config/` ディレクトリを保持してください

## デプロイ移行

旧バージョンからの移行時、以下のデータを保持して新デプロイに移してください：

- `config.yml`
- `user.db`
- `config/logs/`
- `config/dist/scheme.html`
- `config/tg.jpg`

新デプロイでは `/studyclaw/` を統一エントリとして使用することを推奨します。

## よくある質問

### Webログインの認証情報

設定ファイル内の `web.account` / `web.password` です。

### ページが開けない場合

以下の順序で確認してください：

1. プログラムまたはコンテナが実行中かどうか
2. `8080` ポートマッピングが正しいか
3. ホストマシンで `curl http://127.0.0.1:8080/` が成功するか
4. 旧エントリがブラウザにキャッシュされていないか、直接 `/studyclaw/` を開いてみてください

### 日々の回答が表示されない理由

回答コードは残っていますが、正式フローでは無効化されています。これは意図的な安定性重視の調整であり、機能の欠落ではありません。

## 主要ファイル

- `lib/lib.go`：メインワークフロー
- `lib/score.go`：スコア整理と完了通知
- `conf/config_default.yml`：デフォルト設定とプッシュ通知サンプル
- `web/router.go`：Webルーティングと正式エントリ
- `web/studyclaw/src/compents/pages/Help.tsx`：サイト内ヘルプ

## 免責事項

本プロジェクトは学習およびテスト目的のみに使用してください。
