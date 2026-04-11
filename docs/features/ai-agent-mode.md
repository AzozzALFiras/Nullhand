# AI Agent Mode

Send a task in plain language. The AI figures out the steps, uses tools, observes results, and keeps going until complete.

## Usage

Any message without a `/` prefix goes to the AI:

```
Open Safari and go to github.com
Analyze the Go project in ~/projects/myapp and list any issues
Open my server panel, log in, find the SSL for example.com, and renew it
Take a screenshot and describe everything on screen
```

## The Agent Loop

```
Task: "Open Safari and take a screenshot"

Step 1 — AI: "I need to open Safari first"
         calls: open_app("Safari")
         result: "Safari opened"

Step 2 — AI: "Now take a screenshot to confirm"
         calls: take_screenshot()
         result: [image of Safari]

Step 3 — AI sees Safari is open
         responds: "Done. Safari is open."

Bot sends: text + screenshot
```

## Vision

When the AI calls `take_screenshot()`, the image is sent to the AI model. The AI can:

- Read text on screen
- Identify buttons, fields, menus
- Decide where to click based on what it sees
- Detect if a step succeeded or failed

## Available Tools

| Tool | What It Does |
|---|---|
| `take_screenshot` | Captures screen, returns image |
| `open_app` | Opens an application |
| `click` | Clicks at coordinates |
| `right_click` | Right-clicks at coordinates |
| `double_click` | Double-clicks |
| `move_mouse` | Moves cursor |
| `type_text` | Types a string |
| `press_key` | Presses a key or shortcut |
| `run_shell` | Runs a whitelisted shell command |
| `read_file` | Reads a file |
| `list_directory` | Lists directory contents |
| `get_clipboard` | Returns clipboard content |
| `set_clipboard` | Copies to clipboard |
| `get_screen_size` | Returns screen resolution |

## Provider Vision Support

| Provider | Text | Vision | Tools |
|---|---|---|---|
| Claude | ✅ | ✅ | ✅ |
| OpenAI GPT-4o | ✅ | ✅ | ✅ |
| Gemini | ✅ | ✅ | ✅ |
| DeepSeek | ✅ | ⚠️ | ✅ |
| Grok | ✅ | ✅ | ✅ |

## Controls

```
/stop     — cancel the current task immediately
/yes      — confirm a pending dangerous action
/no       — cancel a pending dangerous action
```

## Progress Updates

For long tasks, the bot sends updates as each step completes:

```
Bot: Starting task...
Bot: Opened Safari ✓
Bot: Navigating to server panel...
Bot: [screenshot]
Bot: Logged in ✓
Bot: Found SSL for example.com — expires in 3 days
Bot: Renewing...
Bot: Done. New expiry: 2027-04-11 ✓
```
