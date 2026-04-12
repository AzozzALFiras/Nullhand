# Privacy & Local AI

> Nullhand is designed for **personal use**. You control your own machine, with your own bot, from your own Telegram account.

## Privacy Architecture

```
You (Telegram)  ←──internet──→  Telegram Servers  ←──internet──→  Your Mac
                                                                      │
                                                                  Nullhand
                                                                      │
                                                              ┌───────┴───────┐
                                                              │  AI Provider  │
                                                              └───────────────┘
                                                              Local or Cloud?
```

The only external connection Nullhand makes is to **Telegram** (to receive your commands and send replies). The AI processing can be either **fully local** or **cloud-based** — your choice.

## Local AI (Ollama) — Recommended

When using a local AI model via Ollama:

| What | Where it stays |
|---|---|
| Your commands | Your Mac only |
| Screenshots | Your Mac + your Telegram chat |
| AI processing | Your Mac GPU (Metal) |
| Screen analysis | Your Mac only |
| Passwords you send | Your Telegram → your Mac → typed into app |

**Nothing is sent to any third-party AI server.** The model runs entirely on your hardware.

### How it works

1. You send a command via Telegram
2. Telegram delivers it to Nullhand on your Mac
3. Nullhand processes it using a local AI model (running on your GPU)
4. Results are sent back to you via Telegram

The AI model is a file on your disk. It does not phone home, does not send telemetry, and does not require internet after download.

### Setup

```bash
# Install Ollama (model runtime)
brew install ollama
brew services start ollama

# Download a vision model
ollama pull qwen3-vl:8b
```

Configure `~/.nullhand/config.json`:

```json
{
  "telegram_token": "YOUR_BOT_TOKEN",
  "allowed_user_id": YOUR_TELEGRAM_ID,
  "ai_provider": "ollama",
  "ai_api_key": "",
  "ai_model": "qwen3-vl:8b"
}
```

### Available Local Models

| Model | Download | RAM | Vision | Tool Calling | Quality |
|---|---|---|---|---|---|
| `qwen3-vl:2b` | 1.9 GB | ~3-4 GB | Yes | Yes | Good |
| `qwen3-vl:8b` | 6.1 GB | ~8 GB | Yes | Yes | Excellent |

**Why `qwen3-vl`?** It's the only local model that supports **both vision** (seeing screenshots) **and tool calling** (executing actions) — both are required for Nullhand's AI agent to work.

Apple Silicon Macs (M1/M2/M3/M4) run these models efficiently using Metal GPU acceleration. A 16 GB Mac can comfortably run the 8B model while using other apps.

### After Download

Once downloaded, the model works **completely offline**. You can disconnect from the internet and the AI will still process your commands. Only Telegram needs internet to deliver messages between you and your Mac.

## Cloud AI Providers

Nullhand also supports cloud providers:

| Provider | Config name | Vision | Notes |
|---|---|---|---|
| Anthropic Claude | `claude` | Yes | Best quality, paid API |
| OpenAI GPT-4o | `openai` | Yes | Good quality, paid API |
| Google Gemini | `gemini` | Yes | Good quality, paid API |
| xAI Grok | `grok` | Yes | Vision model, paid API |
| DeepSeek | `deepseek` | No | Text only, no vision |

### Privacy Warning

> When using cloud AI providers, **your commands, conversation history, and screenshots are sent to their servers** for processing. This means:
>
> - The AI company can see your screen content
> - Your data is processed on their infrastructure
> - Their privacy policies apply to your data
> - Screenshots containing sensitive information will be uploaded
>
> **For maximum privacy, use Ollama with a local model.**

### When to use cloud providers

- You need the highest quality AI reasoning (Claude, GPT-4o)
- Your Mac has limited RAM (< 8 GB)
- You don't mind sharing data with AI providers
- You need specific model capabilities

## Telegram Security

Telegram is the communication channel between you and your Mac. Important notes:

- **Whitelist**: Only your Telegram user ID can send commands. All other messages are silently ignored.
- **Encryption**: Telegram uses encryption in transit (MTProto). For extra security, consider using Secret Chats.
- **Bot Token**: Your bot token is stored in `~/.nullhand/config.json` with `0600` permissions (owner-only).
- **No public IP**: Your Mac never exposes any ports. Nullhand polls Telegram — the connection is always outbound.

## Config File Security

```
~/.nullhand/              drwx------ (700)
~/.nullhand/config.json   -rw------- (600)
```

The config directory and file are readable only by your user account. No other user on the system can access your tokens or API keys.

## Shell Command Safety

The `/shell` and `run_shell` tool only execute **whitelisted commands**. Commands like `sudo`, `bash`, and `sh` are not in the whitelist to prevent arbitrary code execution.

For sudo operations, the AI uses the Terminal app approach: it opens Terminal, types the command, and asks you for the password via Telegram — the password is typed into Terminal via clipboard paste and the clipboard is restored immediately after.

## Summary

| Aspect | Local AI (Ollama) | Cloud AI |
|---|---|---|
| AI processing | Your Mac | Provider servers |
| Screenshots | Never leave your Mac | Uploaded to provider |
| Cost | Free | Paid API |
| Internet for AI | Not needed | Required |
| Privacy | 100% private | Provider can see data |
| Quality | Good (8B model) | Excellent (large models) |
| Speed | Depends on hardware | Generally fast |

**Our recommendation: Use Ollama for privacy. Switch to cloud providers only when you need higher quality reasoning and accept the privacy trade-off.**
