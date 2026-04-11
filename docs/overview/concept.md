# Concept

## What is Nullhand?

Nullhand is a local macOS agent that lets you control your computer remotely through a Telegram bot conversation. It combines:

- **Telegram** as the communication interface
- **AI** (Claude, OpenAI, Gemini, DeepSeek, Grok) as the brain
- **macOS native tools** (`osascript`, `screencapture`, `open`) as the hands

The name "Nullhand" represents an invisible hand — one that acts on your machine without physically being there.

## The Problem It Solves

You are away from your Mac. It is on, connected to the internet. You need to:

- Open a browser and renew an SSL certificate on your server
- Analyze a codebase in VS Code and get a report
- Run a shell command and see the output
- Take a screenshot to see what is currently on screen
- Open a file, read it, and summarize it

Normally, you would need a VPN, SSH, or a remote desktop tool. Nullhand replaces all of that with a simple Telegram conversation — using natural language or direct commands.

## What Makes Nullhand Different

| Feature | Traditional Remote Desktop | Nullhand |
|---|---|---|
| Interface | VNC/RDP client app | Telegram (any device) |
| Control style | Manual mouse clicks | Natural language + direct commands |
| AI integration | None | Full AI agent with vision |
| Setup complexity | High (firewall, port forwarding) | Zero (Telegram handles connectivity) |
| Dependencies | Heavy clients | Zero external Go dependencies |
| Platform | Cross-platform | macOS-native, optimized |

## Design Philosophy

1. **Zero external Go dependencies** — built entirely on Go stdlib and macOS native tools
2. **Local first** — everything runs on your machine, nothing in the cloud except the AI API call
3. **Minimal attack surface** — only your Telegram user ID can send commands
4. **Transparency** — every action is logged, dangerous actions require confirmation
5. **Hybrid control** — you decide when to let AI think and when to control directly
