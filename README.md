# Nullhand

> Your invisible hand on the machine.

Nullhand is a self-hosted macOS remote control agent. It runs locally on your Mac and listens for commands via Telegram — giving you full control over your machine from anywhere in the world, powered by AI.

## What It Does

Send a message to your Telegram bot:

```
Open Safari and renew the SSL certificate for example.com
```

Nullhand will open Safari, navigate to your server panel, find the certificate, click renew, and send you a screenshot confirmation — all automatically.

Or take direct control:

```
/screenshot          → get a live screenshot
/click 500 300       → click anywhere on screen
/type Hello World    → type text
/shell git status    → run a command
/open VS Code        → open any app
```

## How It Works

Nullhand runs on your Mac and polls Telegram for messages. When you send a command:

- **Direct commands** (`/click`, `/type`, `/screenshot`) execute instantly with no AI
- **Natural language tasks** go through an AI agent loop that uses vision and tools to complete multi-step tasks

Your Mac never needs a public IP, port forwarding, or VPN. Telegram handles the connection.

## Features

- Zero external Go dependencies — built entirely on stdlib + macOS native tools
- Multiple AI providers: Claude, OpenAI, Gemini, DeepSeek, Grok
- Vision support: AI sees your screen via screenshots
- Hybrid control: mix manual commands and AI tasks freely
- Interactive setup wizard on first run
- Security: whitelist-based access, action confirmation, 0600 config permissions
- Runs as a background service via launchd

## Quick Start

```bash
git clone https://github.com/AzozzALFiras/Nullhand.git
cd Nullhand
go build -o nullhand ./cmd/nullhand
./nullhand
```

Follow the setup wizard, grant macOS permissions, and you are ready.

## Documentation

See the [docs/](docs/) folder for full documentation:

- [Concept & Architecture](docs/overview/concept.md)
- [How It Works](docs/overview/how-it-works.md)
- [Installation](docs/setup/installation.md)
- [First Run](docs/setup/first-run.md)
- [macOS Permissions](docs/setup/macos-permissions.md)
- [Manual Mode Commands](docs/features/manual-mode.md)
- [AI Agent Mode](docs/features/ai-agent-mode.md)
- [Security Model](docs/security/overview.md)

## Requirements

- macOS 12 or later
- Go 1.21 or later
- Telegram account
- API key from Claude, OpenAI, Gemini, DeepSeek, or Grok

## License

MIT
