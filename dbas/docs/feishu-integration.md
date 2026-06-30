# DBAS Feishu Integration Guide

> Connect DBAS to Feishu (飞书) for kanban notifications and bidirectional chat.

DBAS provides two levels of Feishu integration:

| Feature | Type | Direction | Use Case |
|---------|------|-----------|----------|
| **Kanban Notifier** | Webhook push | DBAS → Feishu | Push branch status cards to a Feishu group chat |
| **Chat Bot** | Event callback | DBAS ↔ Feishu | Chat with DBAS via Feishu bot commands |

---

## 1. Kanban Notifier (One-Way)

Pushes kanban cards to a Feishu group chat via custom webhook.

### Setup

1. Open your Feishu group → Settings → Group Bot → Add Custom Bot
2. Copy the webhook URL
3. Configure in `dbas.toml`:

```toml
[kanban]
enabled = true
notifier = "console"

[kanban.feishu]
webhook_url = "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx"
```

4. Restart DBAS — every kanban poll cycle will push a card to the group

### Format

Cards show:
- Project ID and branch name
- Current FSM state with status icon
- Active / terminal branch counts
- Timestamp

---

## 2. Chat Bot (Bidirectional)

Chat with DBAS directly from Feishu. Supports these commands:

| Command | Description |
|---------|-------------|
| `spawn <task>` | Create a new branch and execute a task |
| `status` / `stats` | View DBAS overall status |
| `skills` | List loaded skills |
| `branches` / `list` | List all branches |
| `help` / `?` | Show help |

### Prerequisites

- A Feishu (self-built) app with bot capability enabled
- Feishu Open API access (`https://open.feishu.cn`)

### Step 1: Create a Feishu App

1. Go to [Feishu Developer Console](https://open.feishu.cn/app)
2. Create a **Self-built App**
3. Under **App Credentials**, note:
   - `App ID`
   - `App Secret` (set as environment variable, never in config)
4. Enable **Bot** capability under Features → Bot
5. Under **Event Subscriptions**, add event: `im.message.receive_v1`

### Step 2: Configure Webhook

In Feishu Developer Console → Event Subscriptions:
- Set **Request URL** to `http://<your-server>:8080/api/feishu/webhook`
- Get the **Verification Token** from the event settings page
- Your server must be publicly accessible (or use a tunnel like ngrok)

### Step 3: Configure DBAS

```toml
[feishu_bot]
enabled = true
domain = "feishu"
app_id = "cli_xxxxxxxxxxxxx"
app_secret_env = "FEISHU_BOT_APP_SECRET"
verification_token = "your_verification_token_here"
mode = "webhook"
webhook_port = 8080
require_mention = true
```

### Step 4: Set Credentials

```bash
# Windows (PowerShell)
$env:FEISHU_BOT_APP_SECRET = "your_app_secret"

# macOS / Linux
export FEISHU_BOT_APP_SECRET="your_app_secret"
```

### Step 5: Start DBAS and Test

```bash
./dbas.exe -config dbas.toml
```

Then in Feishu:
1. Find your bot app in contacts
2. Send a direct message or add to a group
3. If `require_mention = true`, @mention the bot in groups
4. Try: `help` or `spawn 修复登录页面样式`

### How It Works

```
Feishu User → @Bot "status"
       ↓
Feishu Open API → POST /api/feishu/webhook (HTTP callback)
       ↓
DBAS feishubot.Bridge → parse event → verify token → dispatch command
       ↓
DBAS Branch Engine → execute command (spawn / status / etc.)
       ↓
feishubot.Bridge → Feishu Open API → send text reply
       ↓
Feishu User ← "📊 DBAS 状态\n• 总分叉: 5..."
```

### Architecture

The bot is implemented in `dbas/internal/dbas/feishubot/bridge.go`:
- **Self-contained**: no external SDK dependency (uses raw HTTP to Feishu Open API)
- **Token caching**: `tenant_access_token` is obtained and cached until expiry
- **Verification**: supports `verification_token` for webhook security
- **Mention gating**: optional `@mention` requirement for group chats
- **Extensible**: command handlers are a simple switch statement — easy to add new commands

---

## 3. Reference: Config Fields

### Kanban Feishu Notifier

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `kanban.feishu.webhook_url` | string | `""` | Custom bot webhook URL |
| `kanban.feishu.app_id` | string | `""` | App ID (for future interactive cards) |
| `kanban.feishu.app_secret_env` | string | `FEISHU_BOT_APP_SECRET` | Env var for app secret |

### Feishu Chat Bot

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `feishu_bot.enabled` | bool | `false` | Enable the bidirectional chat bot |
| `feishu_bot.domain` | string | `"feishu"` | API domain: `feishu` or `lark` |
| `feishu_bot.app_id` | string | `""` | Feishu app ID |
| `feishu_bot.app_secret_env` | string | `FEISHU_BOT_APP_SECRET` | Env var for app secret |
| `feishu_bot.verification_token` | string | `""` | Webhook verification token |
| `feishu_bot.mode` | string | `"webhook"` | Connection mode (currently only webhook) |
| `feishu_bot.webhook_port` | int | `8080` | Port for webhook HTTP server |
| `feishu_bot.require_mention` | bool | `true` | Require @mention in group chats |

---

## 4. Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Bot not responding | `feishu_bot.enabled = false` | Set to `true` |
| Webhook returns 403 | Wrong `verification_token` | Copy from Feishu dev console |
| "get access token" errors | Wrong `app_id` or `app_secret` | Check both values match the dev console |
| Messages not received in groups | `require_mention = true` | @mention the bot |
| No response after @mention | Bot not added to group | Add the bot app as a group member |
| Event subscription failing | Server not reachable | Use ngrok or deploy with public IP |
