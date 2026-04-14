# Nullhand

> Your invisible hand on the machine.

Nullhand is a **personal-use**, self-hosted macOS remote control agent. It runs locally on your Mac and listens for commands via Telegram — giving you full control over your machine from anywhere in the world, powered by AI.

**This project is designed for personal use only.** You control your own machine, with your own bot, from your own Telegram account.

## 🚀 Looking for a Server Edition? Meet AXGhost

While **Nullhand** is your ultimate command center for macOS, we've built a specialized, enterprise-grade tool specifically for servers.

**[AXGhost](https://aevonx.app/marketplace/axghost)** is a **zero-cloud, zero-AI Go daemon** that transforms Telegram into a highly secure, natural-language administration console for your server infrastructure.

✨ **Why AXGhost?**
- 🌍 **Bilingual Brilliance:** Seamlessly execute commands using natural language in both **Arabic and English**.
- 🛡️ **Ironclad Security:** Strict **whitelisted execution** ensures only explicitly approved, safe operations can run.
- 🔐 **100% Private (Zero-Cloud / Zero-AI):** A completely self-contained daemon. No third-party APIs, no AI token limits, and absolutely no data leakage.
- 📜 **Full Audit Trail:** Comprehensive logging provides total visibility over every system action you take.

👉 **[Discover AXGhost on the AevonX Marketplace](https://aevonx.app/marketplace/axghost)**

---

## What It Does

Send a message to your Telegram bot:

```
Open Safari and search for Nullhand on GitHub
```

Nullhand will open Safari, focus the URL bar, type the search query, and press Enter — all automatically.

Or take direct control:

```
/screenshot          → get a live screenshot
/click 500 300       → click anywhere on screen
/type Hello World    → type text
/shell git status    → run a command
/open Terminal       → open any app
```

Or control Terminal, browsers, and VS Code remotely:

```
Run "npm install" in VS Code terminal
Open Safari and go to github.com
Open Terminal and run "git pull"
```

## How It Works

Nullhand runs on your Mac and polls Telegram for messages. When you send a command:

- **Direct commands** (`/click`, `/type`, `/screenshot`) execute instantly with no AI
- **Natural language tasks** go through an AI agent loop that uses vision and tools to complete multi-step tasks

Your Mac never needs a public IP, port forwarding, or VPN. Telegram handles the connection.

## Privacy & Security

### Local AI (Recommended)

Nullhand supports **fully local AI** via [Ollama](https://ollama.com). When using a local model:

- **100% private** — your data never leaves your machine
- **No API keys** — no accounts, no subscriptions, no costs
- **No internet required** for AI processing (only Telegram needs internet)
- **Screenshots stay local** — analyzed on your own GPU, never uploaded

```bash
# Install Ollama and a vision model
brew install ollama
brew services start ollama
ollama pull qwen3-vl:8b
```

Then set `ai_provider` to `"ollama"` in your config. That's it.

### Cloud AI Providers

Nullhand also supports Claude, OpenAI, Gemini, DeepSeek, and Grok. However:

> **Warning:** When using cloud AI providers, your commands and screenshots are sent to their servers for processing. This means a third party can see your screen content. **For maximum privacy, use a local AI model instead.**

| | Local AI (Ollama) | Cloud AI (Claude, GPT, etc.) |
|---|---|---|
| Privacy | 100% local | Data sent to provider servers |
| Cost | Free | Requires paid API key |
| Speed | Depends on hardware | Generally faster |
| Vision | Supported (qwen3-vl) | Supported |
| Internet | Only for Telegram | Required for AI + Telegram |

### Security Features

- **Whitelist access** — only your Telegram user ID can control the bot
- **Config permissions** — config file stored with `0600` (owner-only read/write)
- **Shell whitelist** — only approved commands can run via `/shell`
- **No public IP** — your Mac is never directly exposed to the internet

## Features

- Zero external Go dependencies — built entirely on stdlib + macOS native tools
- **Local AI support** via Ollama (fully private, free, no API key)
- Cloud AI providers: Claude, OpenAI, Gemini, DeepSeek, Grok
- Vision support: AI sees your screen and navigates intelligently
- **40+ automation recipes** for Terminal, browsers, VS Code, WhatsApp, Slack, Discord
- Terminal control: run commands, handle sudo, Ctrl+C, clear screen
- Browser control: open URLs, Google search, tab management, find in page
- VS Code control: integrated terminal, Claude chat, command palette
- Hybrid control: mix manual commands and AI tasks freely
- Supports Arabic, English, and all languages (Unicode-safe typing)
- Interactive setup wizard on first run
- Runs as a background service via launchd

## Quick Start

```bash
git clone https://github.com/AzozzALFiras/Nullhand.git
cd Nullhand
go build -o nullhand ./cmd/nullhand
./nullhand
```

Follow the setup wizard, grant macOS permissions, and you are ready.

### Local AI Setup (Recommended)

```bash
# 1. Install Ollama
brew install ollama
brew services start ollama

# 2. Download a vision model (supports vision + tool calling)
ollama pull qwen3-vl:8b    # 6.1 GB, needs ~8 GB RAM

# 3. Configure Nullhand to use Ollama
# During setup wizard, select "ollama" as AI provider
# Or edit ~/.nullhand/config.json:
# {
#   "ai_provider": "ollama",
#   "ai_model": "qwen3-vl:8b"
# }
```

**RAM Requirements:**

| Model | Size | RAM | Quality |
|---|---|---|---|
| `qwen3-vl:2b` | 1.9 GB | ~3-4 GB | Good |
| `qwen3-vl:8b` | 6.1 GB | ~8 GB | Excellent |

Apple Silicon Macs (M1/M2/M3/M4) run these models efficiently using Metal GPU acceleration.

## Documentation

See the [docs/](docs/) folder for full documentation:

- [Concept & Architecture](docs/overview/concept.md)
- [How It Works](docs/overview/how-it-works.md)
- [Privacy & Local AI](docs/privacy.md)
- [Manual Mode Commands](docs/features/manual-mode.md)
- [AI Agent Mode](docs/features/ai-agent-mode.md)
- [Hybrid Mode](docs/features/hybrid-mode.md)

## Requirements

- macOS 12 or later
- Go 1.21 or later
- Telegram account
- One of:
  - **Ollama** (free, local, private) — recommended
  - API key from Claude, OpenAI, Gemini, DeepSeek, or Grok

## License

MIT
