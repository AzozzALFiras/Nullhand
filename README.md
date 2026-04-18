# Nullhand Linux

Control your Linux desktop from Telegram. Send a message, get a screenshot back. Type text, click coordinates, open apps, transfer files, schedule tasks — all through your phone.

---

## What is this?

Nullhand is a Telegram bot that runs as a background process on your Linux machine and gives you full desktop control over chat. You send natural language or slash commands; the bot acts on your screen in real time.

It ships with an OTP session gate, a structured audit log, a built-in scheduler, bidirectional file transfer, and a pluggable AI backend (local rule-based parser included — no API key required to get started).

---

## Features

- **Natural language control** — "open Firefox", "take a screenshot", "type Hello World"
- **Slash commands** — explicit commands with arguments for scripting workflows
- **Inline quick-action menu** — one tap for the most common actions
- **Screenshot & OCR** — capture the screen or extract visible text with Tesseract
- **Mouse & keyboard automation** — click, double-click, right-click, drag, scroll, type, key shortcuts
- **App launcher** — open GNOME/GTK applications by name
- **File transfer (bidirectional)** — send files from Linux to Telegram; receive files from Telegram to Linux
- **Scheduled daily tasks** — set recurring screenshots, shell commands, or system info reports
- **Audit log** — every action appended to `~/.nullhand/audit.log`
- **OTP session lock** — cryptographically random 6-digit code, auto-rotates every 2 minutes
- **Multiple AI backends** — Claude, OpenAI, Gemini, DeepSeek, Grok, Ollama, or offline local mode
- **Interactive file browser** — browse directories with inline keyboard navigation

---

## How it works

```
You (Telegram)
│
▼
OTP Gate ──── locked? → ignore message
│
▼
Message Router (in order)
│
├── File received? ──────────────→ Destination picker → Save to disk
│
├── OCR trigger? ────────────────→ scrot → tesseract → reply text
│
├── Schedule command? ───────────→ Create/list/cancel task
│
├── File send request? ──────────→ Read file → zip if needed → send
│
├── Slash command? ──────────────→ Execute directly → reply
│
└── Everything else ─────────────→ AI Agent Loop
                                        │
                                   Take screenshot
                                        │
                                   Send to AI model
                                        │
                                   AI picks a tool
                                   (click/type/shell/open)
                                        │
                                   Execute on desktop
                                        │
                                   Take new screenshot
                                        │
                                   Done? → reply result
                                   Not done? → repeat
```

Every action is logged to `~/.nullhand/audit.log` with timestamp and user ID.

---

## Requirements

### System

- Linux with an **X11 session** (Wayland is not supported in this version)
- Log in with **"Ubuntu on Xorg"** (or equivalent) at the display manager
- `$DISPLAY` must be set; `$WAYLAND_DISPLAY` must be unset
- GNOME desktop recommended (some launcher entries are GNOME-specific)

### Dependencies

Install all required tools in one command:

```bash
sudo apt install \
  xdotool \
  scrot \
  wmctrl \
  xclip \
  imagemagick \
  python3 \
  x11-xserver-utils \
  libgtk-3-bin \
  python3-pyatspi \
  at-spi2-core
```

For OCR support (optional but recommended):

```bash
sudo apt install tesseract-ocr
```

| Tool | Package | Purpose |
|---|---|---|
| `xdotool` | xdotool | Key presses, active window query |
| `scrot` | scrot | Screenshots |
| `wmctrl` | wmctrl | App listing and window focus |
| `xclip` | xclip | Clipboard read/write |
| `convert` | imagemagick | Screenshot resizing for HiDPI |
| `python3` | python3 | Accessibility scripting |
| `xrandr` | x11-xserver-utils | Screen resolution detection |
| `gtk-launch` | libgtk-3-bin | App launcher via .desktop files |
| `python3-pyatspi` | python3-pyatspi | AT-SPI accessibility tree access |
| `at-spi2-core` | at-spi2-core | AT-SPI2 daemon |
| `tesseract` | tesseract-ocr | OCR — read text from screen |

### Go version

```
go 1.21 or later
```

### Telegram setup

1. Open Telegram and message `@BotFather`
2. Send `/newbot` and follow the prompts
3. Copy the bot token (format: `123456789:ABCdef...`)
4. Get your Telegram user ID — message `@userinfobot` to find it
5. Start a private chat with your new bot before running Nullhand

---

## Installation

```bash
# 1. Clone the repository
git clone https://github.com/AzozzALFiras/nullhand
cd nullhand

# 2. Install system dependencies (see Requirements above)
sudo apt install xdotool scrot wmctrl xclip imagemagick python3 \
  x11-xserver-utils libgtk-3-bin python3-pyatspi at-spi2-core

# 3. Build for Linux
GOOS=linux go build -o nullhand ./cmd/nullhand

# 4. Run
./nullhand
```

On first run, a setup wizard will prompt you for your Telegram bot token, your Telegram user ID, and your preferred AI provider. Configuration is saved to `~/.nullhand/config.json`.

---

## First Run & OTP

When Nullhand starts (and on every restart), it prints a one-time password to the terminal:

```
╔══════════════════════════════╗
║  OTP CODE: 482917          ║
║  Expires in 2 minutes        ║
╚══════════════════════════════╝

Enter this code in Telegram to unlock the bot.
```

**You must send this exact 6-digit code to the bot in Telegram before any command is accepted.** The code:

- Is generated with `crypto/rand` — cryptographically random
- Expires after **2 minutes** and is automatically replaced with a new one (printed to terminal again)
- Once entered correctly, the session stays unlocked until you restart or use the **Lock Bot** button in `/menu`
- Is stored in memory only — never written to disk

To re-lock the session manually, tap **Lock Bot** in `/menu` or press the `menu:lock` inline button.

---

## Commands & Usage

### Natural Language Examples

Just send a message in plain English:

```
take a screenshot
```
```
what's my CPU usage
```
```
open Firefox
```
```
type Hello World into the terminal
```
```
click at 960 540
```
```
press ctrl+t
```
```
send me /home/user/report.pdf
```
```
read the screen
```
```
run git status in terminal
```
```
schedule a screenshot every day at 9am
```
```
remind me to run sysinfo every day at 14:00
```

### Slash Commands (table)

| Command | Arguments | Description |
|---|---|---|
| `/start` | — | Welcome message and command list |
| `/help` | — | Show all available commands |
| `/screenshot` | — | Capture the full screen and send as photo |
| `/status` | — | CPU, memory, and active application info |
| `/apps` | — | List currently open windows |
| `/open` | `<app name>` | Open an application by name |
| `/ls` | `[path]` | List directory contents |
| `/read` | `<path>` | Read a file and return its contents |
| `/shell` | `<command>` | Run a whitelisted shell command |
| `/click` | `<x> <y>` | Click at the given screen coordinates |
| `/type` | `<text>` | Type text into the active window |
| `/key` | `<shortcut>` | Press a key or modifier combination |
| `/paste` | — | Get current clipboard contents |
| `/stop` | — | Cancel the currently running AI task |
| `/diag` | — | Show diagnostic info (frontmost app, screen size) |
| `/inspect` | — | Dump accessibility tree of the frontmost window |
| `/ocr` | — | Extract visible text from the screen |
| `/schedule` | `list` \| `cancel <id>` \| `clear` | Manage scheduled tasks |
| `/menu` | — | Open the inline quick-action toolbar |

**Keyboard shortcut examples for `/key`:**

```
/key enter
/key ctrl+t
/key ctrl+shift+5
/key escape
/key f5
/key super
```

Modifier aliases: `cmd` and `command` map to `ctrl`; `option` maps to `alt`.

### Inline Menu (/menu)

Send `/menu` to get the quick-action toolbar with inline keyboard buttons:

| Button | Action |
|---|---|
| 📸 Screenshot | Capture and send the current screen |
| 💻 System Info | Show CPU, memory, active app |
| 📋 Clipboard | Read and return clipboard contents |
| 🐚 Run Command | Prompt for a shell command, execute it |
| 📤 Send File | Prompt for a file path, upload to Telegram |
| 📥 Downloads | List `~/Downloads` directory |
| 🔍 Read Screen | OCR — extract text from the current screen |
| 🔒 Lock Bot | Lock the session; new OTP printed to terminal |
| ❓ Help | Show natural language usage examples |

---

## File Transfer

### Sending a file from Linux to Telegram

**Natural language:**
```
send me /home/user/documents/report.pdf
```

**Upload keyword with path:**
```
upload /var/log/syslog
```

**Slash command via menu button:**
Tap **Send File** in `/menu`, then enter the path when prompted.

**How it works:**
- Files under 50 MB are sent directly
- Files over 50 MB and entire directories are automatically zipped before sending
- The file type determines the Telegram method: images use `sendPhoto`, everything else uses `sendDocument`
- Temporary zip files are always cleaned up after sending

### Receiving a file from Telegram

Simply send or forward any file (document, photo, video, audio) to the bot. You will be asked where to save it:

```
📥 Where should I save "report.pdf"?
[ 🏠 Home ]  [ 🖥️ Desktop ]
[ 📥 Downloads ]  [ ✏️ Custom path ]
```

Tap a button to save to that location, or tap **Custom path** and type a full directory path (e.g. `/home/user/projects/`).

If a file with the same name already exists, a timestamp is appended automatically (`report_20260417_153012.pdf`).

---

## OCR

Nullhand can read text visible on screen using Tesseract OCR.

**Requires:**
```bash
sudo apt install tesseract-ocr
```

**Trigger via natural language:**
```
read the screen
what does the screen say
read text on screen
ocr
extract text from screen
what's written on screen
```

**Trigger via slash command:**
```
/ocr
```

**Trigger via menu button:** tap **Read Screen** in `/menu`.

**How it works:**
1. Full screenshot is captured via `scrot`
2. Screenshot is written to a temp file
3. `tesseract <file> stdout -l eng` is executed
4. Output is trimmed and truncated to 4096 characters (Telegram message limit)
5. Temp file is deleted immediately after

If Tesseract is not installed, the bot responds with the install command rather than crashing.

---

## Scheduled Tasks

Schedule recurring daily tasks using natural language or slash commands.

### Creating a task (natural language)

The bot detects schedule intent when your message contains phrases like "every day at", "schedule", or "remind me to".

```
schedule a screenshot every day at 9am
```
```
remind me to run sysinfo every day at 8:30am
```
```
run git status every day at 14:00
```
```
send me /home/user/backup.tar.gz every day at 2am
```
```
read screen every day at 9pm
```

**Supported time formats:** `8am`, `8:30am`, `14:00`, `9pm`

**Supported actions:**

| Phrase contains | Scheduled action |
|---|---|
| `screenshot` | Capture screen, send as photo |
| `sysinfo`, `cpu`, `status`, `system info` | Send system status report |
| `read screen`, `ocr` | Run OCR and send text |
| `run <cmd>` or `shell <cmd>` | Run shell command, send output |
| `send` + a `/path` | Send file to Telegram |

### Managing tasks

```
/schedule list
```
```
/schedule cancel task_001
```
```
/schedule clear
```

**Example output of `/schedule list`:**
```
📋 Active scheduled tasks:
🆔 task_001 — screenshot — every day at 09:00
🆔 task_002 — sysinfo — every day at 14:00

Use /schedule cancel <id> to remove a task.
```

**Implementation detail:** the scheduler aligns to the next whole minute on start, then checks every minute. Panics in task callbacks are recovered and logged.

---

## Audit Log

Every action is appended to `~/.nullhand/audit.log`.

**Log format:**
```
[2026-04-17 09:31:05] user=123456789 action=screenshot
[2026-04-17 09:32:11] user=123456789 action=shell cmd="git status"
[2026-04-17 09:33:00] user=123456789 action=file_send path="/home/user/report.pdf"
[2026-04-17 09:34:45] user=123456789 action=otp_unlock
[2026-04-17 09:35:00] user=123456789 action=schedule_create id="task_001"
[2026-04-17 09:40:00] user=123456789 action=scheduled_task id="task_001"
[2026-04-17 09:41:10] user=123456789 action=natural_language input="open Firefox and go to..."
```

**Actions logged:**

| Action | Triggered by |
|---|---|
| `otp_unlock` | Successful OTP entry |
| `otp_lock` | Lock Bot button |
| `screenshot` | `/screenshot`, menu button, or AI tool |
| `shell` | `/shell`, menu button, or AI tool |
| `app_open` | `/open` command |
| `clipboard` | `/paste`, menu button |
| `sysinfo` | `/status`, menu button |
| `ocr` | `/ocr`, natural language, menu button |
| `file_send` | File send trigger |
| `file_receive` | File received from Telegram |
| `downloads` | Downloads menu button |
| `natural_language` | Free-form AI task (first 80 chars logged) |
| `schedule_create` | New scheduled task |
| `schedule_cancel` | Task cancelled |
| `scheduled_task` | Scheduled task fired |

The log directory (`~/.nullhand/`) is created with mode `0700`. The log file has mode `0600`. Logging failures are silently swallowed so a disk error never crashes the bot.

Read the log:
```bash
cat ~/.nullhand/audit.log
```

Tail it live:
```bash
tail -f ~/.nullhand/audit.log
```

---

## Security

**Single-user only.** The bot accepts messages from exactly one Telegram user ID (set during first-run setup). Messages from any other account are silently dropped.

**OTP session gate.** Before any command is processed, the session must be unlocked with the current OTP code. The code is:
- Generated with Go's `crypto/rand`
- A 6-digit number in the range 100000–999999
- Stored in memory only, never written to disk or logged
- Automatically replaced every 2 minutes (new code printed to terminal)
- Invalidated on successful entry (cannot be reused within the same session)

**X11-only.** The startup check rejects runs under Wayland (`$WAYLAND_DISPLAY` set) and headless SSH sessions (`$DISPLAY` unset).

**Capability checks.** Before starting, Nullhand verifies that `scrot` can actually take a screenshot and that `xdotool` can query the active window. If either check fails, the process exits with a clear message.

**No inbound network ports.** Nullhand uses Telegram long-polling outbound only — there is no listening server or open port.

---

## AI Providers

Configure the provider during first-run setup or edit `~/.nullhand/config.json`.

| Provider | `ai_provider` value | Requires API key | Vision | Notes |
|---|---|---|---|---|
| Anthropic Claude | `claude` | Yes | Yes | Set `ai_api_key` |
| OpenAI | `openai` | Yes | Yes | Set `ai_api_key`; optional `ai_base_url` for proxies |
| Google Gemini | `gemini` | Yes | Yes | Set `ai_api_key` |
| DeepSeek | `deepseek` | Yes | No | Set `ai_api_key` |
| Grok (xAI) | `grok` | Yes | No | Set `ai_api_key` |
| Ollama (local LLM) | `ollama` | No | Model-dependent | Set `ai_base_url` and `ai_model`; use a vision model for screenshot analysis |
| Built-in rule-based | `local` | No | No | Zero cost, zero external dependency |

> **Privacy note:** Cloud providers (Claude, OpenAI, Gemini, DeepSeek, Grok) receive your commands and screenshots when the AI agent calls `analyze_screenshot`. If privacy matters, use Ollama or `local`.

| | Local AI (Ollama) | Cloud AI (Claude, GPT, etc.) |
|---|---|---|
| Privacy | 100% local | Data sent to provider servers |
| Cost | Free | Requires paid API key |
| Vision | Supported (vision models) | Supported |
| Internet | Only for Telegram | Required for AI + Telegram |

---

## Local AI Setup

### Option 1 — Built-in rule-based parser (zero dependencies)

The `local` provider requires no API key, no network, and no external process. Use it to get started immediately or in air-gapped environments.

```json
{
  "ai_provider": "local"
}
```

### Option 2 — Ollama (recommended for full AI capability)

Ollama runs open-source LLMs locally. For full screenshot analysis support, use a **vision model**.

```bash
# 1. Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. Pull a vision model (recommended — supports analyze_screenshot tool)
ollama pull qwen3-vl:8b    # 6.1 GB download, needs ~8 GB RAM

# Or a smaller vision model if RAM is limited
ollama pull qwen3-vl:2b    # 1.9 GB download, needs ~3-4 GB RAM

# 3. Start Ollama (if not already running as a service)
ollama serve
```

**RAM requirements:**

| Model | Download size | RAM needed | Quality |
|---|---|---|---|
| `qwen3-vl:2b` | 1.9 GB | ~3–4 GB | Good |
| `qwen3-vl:8b` | 6.1 GB | ~8 GB | Excellent |

Configure Nullhand to use Ollama:

```json
{
  "ai_provider": "ollama",
  "ai_model": "qwen3-vl:8b",
  "ai_base_url": "http://localhost:11434"
}
```

If you don't need screenshot analysis and want a lighter model:

```bash
ollama pull llama3
```

```json
{
  "ai_provider": "ollama",
  "ai_model": "llama3",
  "ai_base_url": "http://localhost:11434"
}
```

---

## Troubleshooting

### Bot can't connect to Telegram
**Symptom:** `dial tcp: lookup api.telegram.org: server misbehaving` or connection timeout errors in terminal.

**Fix:** Your DNS may not be resolving correctly. Run:
```bash
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```
Then restart the bot.

If on a VM, also try:
```bash
sudo systemctl restart NetworkManager
```

---

### Screenshot not working / scrot fails silently
**Symptom:** `/screenshot` returns nothing, or `scrot` produces an empty file.

**Fix 1 — Set DISPLAY variable:**
```bash
export DISPLAY=:0
```
Add this to your `~/.bashrc` to make it permanent:
```bash
echo 'export DISPLAY=:0' >> ~/.bashrc && source ~/.bashrc
```

**Fix 2 — Allow local X11 connections:**
```bash
xhost +local:
```
Run this after every login, or add it to your startup applications.

**Fix 3 — Verify scrot works manually:**
```bash
DISPLAY=:0 scrot /tmp/test.png && echo "works" && ls -la /tmp/test.png
```
If the file is 0 bytes, your X11 session may not be properly initialized.

---

### Must use X11, not Wayland
**Symptom:** `$DISPLAY not set` error on startup, or xdotool/scrot failing completely.

**Cause:** Nullhand requires an X11 session. Wayland is not supported in v1.

**Fix:** At the login screen, click the gear icon ⚙️ and select **"Ubuntu on Xorg"**
(or your distro's equivalent X11 session) before logging in.

To verify you're on X11:
```bash
echo $XDG_SESSION_TYPE
```
Should output `x11`. If it outputs `wayland`, log out and select Xorg session.

---

### xdotool not working / click and type commands fail
**Symptom:** `/click`, `/type`, `/key` return ✓ but nothing happens on screen.

**Fix:** Ensure DISPLAY is set and xdotool can reach the display:
```bash
export DISPLAY=:0
xdotool getactivewindow
```
If `getactivewindow` returns a window ID, xdotool is working correctly.
If it errors, your X11 session needs the local connection fix:
```bash
xhost +local:
```

---

### OCR returns empty or garbled text
**Symptom:** `/ocr` returns no text or random characters.

**Cause:** Tesseract may not be installed, or the screen content is purely graphical.

**Fix:**
```bash
sudo apt install tesseract-ocr
tesseract --version
```

Note: OCR works best on text-heavy screens. Purely graphical content (icons, images)
will return little or no text — this is expected behavior.

---

### Clipboard commands not working (/paste returns empty)
**Symptom:** `/paste` returns empty or fails silently.

**Fix:** Ensure xclip is installed and DISPLAY is set:
```bash
sudo apt install xclip
export DISPLAY=:0
xclip -selection clipboard -o
```
If xclip errors with "Can't open display", run `xhost +local:` first.

---

### AI agent not responding / "empty choices" error
**Symptom:** Natural language commands return `AI call failed: empty choices` or similar.

**Cause:** Your AI provider's API is unavailable or the API key has no credits.

**Fix options:**
1. Switch to the built-in local provider (no API key needed):
   Edit `~/.nullhand/config.json` and set `"ai_provider": "local"`
2. Check your API key has credits at your provider's dashboard
3. Try a different AI provider

Note: The `local` provider handles simple commands (open app, screenshot, status)
but does not support vision or complex multi-step tasks.

---

### OTP code not showing / bot not starting
**Symptom:** Bot starts but no OTP box appears, or bot exits immediately.

**Fix:** Check the terminal output for error messages. Common causes:
- Missing dependencies → run the full `apt install` command from the Requirements section
- Wrong display session → ensure you're on X11 not Wayland
- Config file corrupted → delete `~/.nullhand/config.json` and run setup again:
```bash
rm ~/.nullhand/config.json && ./nullhand
```

---

### Running in a VirtualBox VM

**Clipboard sharing between host and VM:**
```bash
sudo apt install virtualbox-guest-x11
sudo reboot
```
Then in VirtualBox menu: **Devices → Shared Clipboard → Bidirectional**

**No internet in VM:**
1. In VirtualBox Settings → Network → change to **Bridged Adapter**
2. Select your active network adapter (WiFi or Ethernet) from the Name dropdown
3. Start VM and run:
```bash
sudo systemctl restart NetworkManager
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
ping google.com
```

**Slow performance:**
Allocate more resources in VirtualBox Settings:
- RAM: 4096 MB recommended
- CPUs: 2 minimum
- Video Memory: 128 MB (Display settings)

---

### Scheduled tasks not firing
**Symptom:** Scheduled tasks were set but never executed.

**Cause:** Tasks are stored in memory only. They are lost when the bot restarts.

**Fix:** Re-create your scheduled tasks after each bot restart. Persistent scheduling
across restarts is planned for a future version.

---

### General dependency check
Run this to verify all required tools are installed:
```bash
which git go xdotool scrot wmctrl xclip convert tesseract && echo "✅ All dependencies found"
```

If any are missing:
```bash
sudo apt install -y git golang xdotool scrot wmctrl xclip imagemagick python3-pyatspi at-spi2-core desktop-file-utils tesseract-ocr
```

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-change`
3. Make your changes
4. Build and verify: `GOOS=linux go build ./...`
5. Open a pull request

Linux-specific code lives under `internal/service/linux/` and uses the `//go:build linux` build tag. Keep platform-specific code behind that tag.

---

## License

See [LICENSE](LICENSE) in the repository root.
