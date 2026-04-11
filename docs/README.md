# Nullhand Documentation

> Your invisible hand on the machine.

Nullhand is a self-hosted macOS remote control agent powered by AI. It runs locally on your Mac and listens for commands via Telegram — giving you full control over your machine from anywhere in the world.

## Documentation Index

### Overview
- [Concept](overview/concept.md) — What Nullhand is and why it exists
- [Architecture](overview/architecture.md) — High-level system design
- [How It Works](overview/how-it-works.md) — Telegram polling, local execution, AI loop

### Features
- [Manual Mode](features/manual-mode.md) — Direct commands (/click, /type, /screenshot)
- [AI Agent Mode](features/ai-agent-mode.md) — Natural language tasks powered by AI
- [Hybrid Mode](features/hybrid-mode.md) — Combining manual and AI control
- [AI Providers](features/providers.md) — Supported AI providers

### Setup
- [Installation](setup/installation.md) — Build and run Nullhand
- [First Run](setup/first-run.md) — Setup wizard walkthrough
- [macOS Permissions](setup/macos-permissions.md) — Granting required system permissions

### Tools
- [Mouse & Keyboard](tools/mouse-keyboard.md) — Simulating input via osascript
- [Screen](tools/screen.md) — Taking screenshots via screencapture
- [Apps](tools/apps.md) — Opening, closing, and focusing applications
- [Shell](tools/shell.md) — Executing whitelisted shell commands
- [Files](tools/files.md) — Reading and writing files

### Security
- [Security Model](security/overview.md) — How Nullhand stays safe
- [User Whitelist](security/user-whitelist.md) — Restricting access to your Telegram ID only
- [Action Confirmation](security/confirmation.md) — Confirming dangerous actions before execution
