# studyclaw

`studyclaw` 是一個保留多帳號接入、Web 管理、推送通知與配置體系的學習控制台，並把正式主流程收斂成更穩定的兩段：

1. 文章學習
2. 音頻學習

目前每日答題、每周答題、專項答題的代碼仍保留在專案中，但不接入預設積分流程。

## 專案定位

- 以 `文章 + 音頻` 作為唯一正式學習流程，降低不穩定環節。
- 保留原專案常用的 Web、推送、定時與多帳號管理能力。
- 提供正式入口 `/studyclaw/`，部署後可直接分享給其他使用者。
- 提供簡潔的 Web 控制台與清楚的完成通知格式。

## 目前收斂方向

- 正式流程不再執行每日答題。
- 登入積分不列入主流程。
- 完成通知會包含總積分、今日得分、本次用時，以及程式可學三項的完成情況。
- Web 介面已改為簡潔風格，並收斂過時文案。

## 快速開始

### Docker Compose

```bash
docker compose up -d
```

預設會：

- 啟動 `8080` Web 服務
- 掛載 `./config` 到容器內 `/opt/config`
- 使用 `ghcr.io/legolasljl/studyclaw:latest`

### 可執行檔

初始化配置：

```bash
./studyclaw --init
```

啟動程式：

```bash
./studyclaw
```

### 原始碼執行

```bash
go mod tidy
go build -o studyclaw ./
./studyclaw --init
./studyclaw
```

## Web 入口

建議使用以下正式入口：

- `/studyclaw/`

另外也保留根路徑 `/`，會自動導向 `/studyclaw/`。

管理員登入帳密來自 `config/config.yml` 內的：

- `web.account`
- `web.password`

普通用戶則由 `web.common_user` 管理，支援多帳號，但只能查看自己的資料。

## 配置流程

### 1. 初始化配置

首次執行 `./studyclaw --init` 或第一次啟動容器後，會生成 `config/config.yml`。

### 2. 設定 Web 與管理帳號

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

### 3. 設定學習節奏

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

### 4. 設定推送

`conf/config_default.yml` 已保留安全示例；實際部署時請把需要的段落填進 `config/config.yml`。

目前內建示例包括：

- DingTalk
- PushPlus
- Telegram
- 微信公眾號測試號
- QQ / go-cqhttp
- PushDeer
- 極光推送

#### Telegram

可直接填值的範例如下：

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

欄位說明：

- `enable`：是否啟用 Telegram 推送與互動指令。
- `chat_id`：預設接收通知的個人或群組 ID。
- `token`：`@BotFather` 建立機器人後取得的 bot token。
- `proxy`：可選；若你的環境連不到 Telegram，可填代理 URL。
- `custom_api`：可選；若你有自架 Telegram API 反代，可替換這個網址。
- `white_list`：允許向 bot 發送命令的 chat id 清單；建議至少填入自己的 `chat_id`。

設定步驟：

1. 在 Telegram 用 `@BotFather` 建立機器人並取得 `token`。
2. 先對 bot 發一次 `/start`，讓 Telegram 建立對話。
3. 用 bot 或其他工具查出你的 `chat_id`。
4. 把 `chat_id` 同時填進 `chat_id` 與 `white_list`。
5. 重啟程式後，先發 `/ping` 檢查 bot 是否在線。

#### DingTalk

可直接填值的範例如下：

```yaml
push:
  ding:
    enable: true
    access_token: "your_ding_access_token"
    secret: "SECxxxxxxxxxxxxxxxxxxxxxxxx"
```

欄位說明：

- `enable`：是否啟用釘釘群機器人推送。
- `access_token`：群機器人 webhook 裡的 token 本體，不要貼整條 webhook URL。
- `secret`：若機器人開啟「加簽」，填入對應的 `SEC...` 密鑰；若未開啟加簽，這個值保持空字串。

設定步驟：

1. 在釘釘群新增自訂機器人。
2. 記下 webhook 中的 `access_token`。
3. 若啟用了安全設定中的「加簽」，再把 `secret` 一起填入。
4. 重啟程式後觀察啟動通知是否正常送達。

#### 微信公眾號測試號

可直接填值的範例如下：

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

欄位說明：

- `enable`：是否啟用微信公眾號測試號推送。
- `token`：微信測試號後台「接口配置信息」中的 token。
- `secret`：微信測試號的 `appsecret`。
- `app_id`：微信測試號的 `appID`。
- `login_temp_id`：登入或授權流程用的模板消息 ID。
- `normal_temp_id`：一般學習通知用的模板消息 ID。
- `push_login_warn`：cookie 失效時是否主動推送提醒。
- `super_open_id`：管理員自己的 openid，用來接收管理級消息。

設定步驟：

1. 到微信公眾平台測試號頁面建立測試號。
2. 在測試號後台取得 `app_id`、`secret`、`token`。
3. 建立登入通知與一般通知模板，填入 `login_temp_id`、`normal_temp_id`。
4. 用微信掃碼關注測試號，取得自己的 `openid` 後填入 `super_open_id`。
5. 確保部署入口可被微信平台回調，再重啟程式驗證推送。

### 5. 接入學習帳戶

1. 打開 Web 控制台。
2. 以管理員帳號登入。
3. 進入「用戶管理」。
4. 產生授權二維碼。
5. 使用學習強國 App 掃碼接入。
6. 在帳戶列表或積分查詢中確認帳號狀態。

## 完成通知格式

```text
xx帳號 已學習完成
當前學習總積分：
今日得分：
本次用時：
文章學習： /12
視頻學習： /12
每日答題： /5
程式可學三項： /29
```

說明：

- `今日得分` 是學習強國當天所有已得分項目的總和，可能高於 `29`。
- 程式內可直接完成的三項只有 `文章學習 12`、`視頻學習 12`、`每日答題 5`，合計 `29` 分。
- 若你已透過其他方式拿到登入、訂閱、專項等分數，`今日得分` 仍會增加，但不影響程式判斷這三項是否學滿。

## Docker 部署說明

若你使用容器部署，最常見的重點如下：

- `-p 8080:8080` 對應 Web 入口
- `./config:/opt/config` 用來保留 `config.yml`、`user.db` 與日誌
- 若重建容器，請保留宿主機的 `config/` 目錄

## 部署遷移

若你原本已經有舊版部署，可直接保留以下資料搬到新部署：

- `config.yml`
- `user.db`
- `config/logs/`
- `config/dist/scheme.html`
- `config/tg.jpg`

新部署建議一律對外使用 `/studyclaw/`。

## 常見問題

### Web 登入帳密是什麼

配置檔中的 `web.account` / `web.password`。

### 頁面打不開怎麼辦

先依序檢查：

1. 程式或容器是否正在運行
2. `8080` 埠映射是否正確
3. 宿主機是否能 `curl http://127.0.0.1:8080/`
4. 舊入口是否仍被瀏覽器快取，可嘗試直接打開 `/studyclaw/`

### 為什麼看不到每日答題

答題代碼仍在，但正式流程已停用。這是刻意的穩定性取向調整，不是功能遺失。

## 主要檔案

- `lib/lib.go`：主工作流
- `lib/score.go`：積分整理與完成通知
- `conf/config_default.yml`：預設配置與推送示例
- `web/router.go`：Web 路由與正式入口
- `web/studyclaw/src/compents/pages/Help.tsx`：站內使用說明

## 聲明

本專案僅供學習與測試用途。
