# How It Works

## Telegram as a Bridge

Nullhand does not require a public IP, port forwarding, or VPN.

**Your Mac connects to Telegram. Telegram does not connect to your Mac.**

```
Your Phone ──▶ Telegram Servers ◀── Your Mac (Nullhand polling)
```

Nullhand runs on your Mac and continuously asks Telegram: "Any new messages for me?" This is called **long-polling**.

## Long-Polling

```go
offset := 0
for {
    updates := getUpdates(offset, timeout=30)
    for _, update := range updates {
        offset = update.ID + 1
        if isAllowedUser(update) {
            handleMessage(update)
        }
    }
}
```

- `timeout=30` means Telegram holds the connection open for 30 seconds waiting for a message
- If a message arrives, it returns immediately
- If nothing arrives, it returns empty and Nullhand polls again
- Near real-time response with minimal CPU usage

## Two Modes of Operation

### 1. Manual Mode

Direct commands starting with `/`:

```
/screenshot
/click 500 300
/type Hello World
/open Safari
/shell ls ~/Desktop
```

Executed immediately — no AI, no API cost.

### 2. AI Agent Mode

Natural language tasks:

```
Take a screenshot and tell me what you see
Open VS Code, analyze ~/projects/myapp, list all TODO comments
Open Safari, go to my server panel, renew SSL for example.com
```

Runs an **agent loop**:

```
1. Send task + available tools to AI
2. AI responds with a tool call
3. Execute the tool on your Mac
4. Send result back to AI
5. AI decides: done? or call another tool?
6. Repeat until done
7. Send final response to Telegram
```

### The Vision Loop

When a task involves the screen:

1. Take a screenshot
2. Send it to the AI as an image
3. AI sees the screen and decides what to do
4. AI calls a tool (click, type, etc.)
5. Take another screenshot
6. Repeat

The AI behaves like a human looking at the screen and reacting to what it sees.

## Message Flow

```
Message arrives from Telegram
        │
        ▼
Is sender == ALLOWED_USER_ID?
       NO ──▶ Ignore silently
        │
       YES
        ▼
Starts with "/"?
       YES ──▶ Execute tool directly ──▶ Send result
        │
       NO ──▶ AI agent loop ──▶ Send result
```
