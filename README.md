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
- 完成通知只保留總積分、今日得分、本次用時、文章學習與視頻學習。
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

`conf/config_default.yml` 已保留安全示例，可直接依照需要填入：

- DingTalk
- PushPlus
- Telegram
- 微信公眾號測試號
- QQ / go-cqhttp
- PushDeer
- 極光推送

例如 Telegram：

```yaml
tg:
  enable: false
  chat_id: 123456789
  token: "123456789:telegram_bot_token"
  proxy: ""
  custom_api: "https://api.telegram.org"
  white_list:
    - 123456789
```

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
```

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
