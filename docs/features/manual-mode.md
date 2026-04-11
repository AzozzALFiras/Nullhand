# Manual Mode

Direct, instant control over your Mac without involving any AI.

## Commands

### Screenshot
```
/screenshot           — full screen
/screenshot active    — active window only
```

### Mouse
```
/click 500 300        — left click at x,y
/rclick 500 300       — right click
/dclick 500 300       — double click
/move 500 300         — move cursor
/drag 100 200 400 200 — drag from → to
/scroll down 5        — scroll down 5 steps
/scroll up 3
```

### Keyboard
```
/type Hello World     — type text
/key cmd+t            — keyboard shortcut
/key enter
/key escape
/key cmd+shift+5
```

Modifiers: `cmd`, `shift`, `ctrl`, `alt`
Special keys: `enter`, `escape`, `tab`, `space`, `delete`, `up`, `down`, `left`, `right`, `f1`–`f12`

### Apps
```
/open Safari
/open "Visual Studio Code"
/open Terminal
```

### Shell
```
/shell ls ~/Desktop
/shell git status
/shell go build ./...
```
Only whitelisted commands allowed. See [Shell Tool](../tools/shell.md).

### Files
```
/read ~/projects/myapp/main.go
/ls ~/Desktop
/ls ~/projects/myapp
```

### Clipboard
```
/paste                — get clipboard content
/copy Hello World     — copy to clipboard
```

### System
```
/status               — CPU, memory, active app, screen size
/apps                 — list running applications
/stop                 — stop current AI task
```

## Notes

- Manual commands cost zero API credits
- Responses are instant
- Use `/screenshot` often to see current screen state
