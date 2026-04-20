package command

import (
	"fmt"
	"strconv"
	"strings"

	cmdmodel "github.com/AzozzALFiras/Nullhand/internal/model/command"
	a11ysvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/accessibility"
	appsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/apps"
	filesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/files"
	kbsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/keyboard"
	mousesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/mouse"
	screensvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/screen"
	shellsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/shell"
	systemsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/system"
	tgfmt "github.com/AzozzALFiras/Nullhand/internal/view/telegram"
)

// Result carries the outcome of executing a manual command.
type Result struct {
	Text      string
	ImageData []byte // non-nil when a screenshot should be sent
}

// ViewModel executes manual slash commands and returns formatted results.
type ViewModel struct{}

// New creates a command ViewModel.
func New() *ViewModel { return &ViewModel{} }

// Execute dispatches the command and returns the result.
func (vm *ViewModel) Execute(cmd *cmdmodel.Command) Result {
	switch cmd.Name {
	case "start", "help":
		return vm.help()
	case "screenshot":
		return vm.screenshot(cmd.Args)
	case "click":
		return vm.mouseAction("click", cmd.Args)
	case "rclick":
		return vm.mouseAction("rclick", cmd.Args)
	case "dclick":
		return vm.mouseAction("dclick", cmd.Args)
	case "move":
		return vm.mouseAction("move", cmd.Args)
	case "drag":
		return vm.drag(cmd.Args)
	case "scroll":
		return vm.scroll(cmd.Args)
	case "type":
		return vm.typeText(cmd.Args)
	case "key":
		return vm.pressKey(cmd.Args)
	case "open":
		return vm.openApp(cmd.Args)
	case "shell":
		return vm.shell(cmd.Args)
	case "read":
		return vm.readFile(cmd.Args)
	case "ls":
		return vm.listDir(cmd.Args)
	case "paste":
		return vm.getClipboard()
	case "copy":
		return vm.setClipboard(cmd.Args)
	case "apps":
		return vm.listApps()
	case "status":
		return vm.status()
	case "diag":
		return vm.diag()
	case "inspect":
		return vm.inspect()
	default:
		return Result{Text: fmt.Sprintf("Unknown command: /%s", cmd.Name)}
	}
}

// diag returns a quick system+app sanity report for debugging.
func (vm *ViewModel) diag() Result {
	var sb strings.Builder
	sb.WriteString("<b>🔍 Diagnostics</b>\n\n")

	// Active app
	if active, err := systemsvc.ActiveApp(); err == nil {
		sb.WriteString(fmt.Sprintf("Frontmost app: %s\n", active))
	} else {
		sb.WriteString(fmt.Sprintf("Frontmost app: (error: %v)\n", err))
	}

	// Screen size
	if w, h, err := screensvc.Size(); err == nil {
		sb.WriteString(fmt.Sprintf("Screen: %d x %d\n", w, h))
	}

	// Running apps (top 10)
	if running, err := appsvc.List(); err == nil {
		max := 10
		if len(running) < max {
			max = len(running)
		}
		sb.WriteString(fmt.Sprintf("Running apps (%d total, showing %d):\n", len(running), max))
		for i := 0; i < max; i++ {
			sb.WriteString("  • ")
			sb.WriteString(running[i])
			sb.WriteString("\n")
		}
	}

	return Result{Text: sb.String()}
}

// inspect dumps the AX tree of the frontmost window for debugging.
func (vm *ViewModel) inspect() Result {
	tree, err := a11ysvc.ListElements(6)
	if err != nil {
		return Result{Text: tgfmt.FailWith("inspect", err)}
	}
	return Result{Text: tgfmt.Code(tree)}
}

// ---- handlers ----

func (vm *ViewModel) screenshot(args []string) Result {
	var data []byte
	var err error
	if len(args) > 0 && strings.ToLower(args[0]) == "active" {
		data, err = screensvc.CaptureActive()
	} else {
		data, err = screensvc.Capture()
	}
	if err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{ImageData: data}
}

func (vm *ViewModel) mouseAction(action string, args []string) Result {
	x, y, err := parseXY(args)
	if err != nil {
		return Result{Text: tgfmt.FailWith(action, err)}
	}
	switch action {
	case "click":
		err = mousesvc.Click(x, y)
	case "rclick":
		err = mousesvc.RightClick(x, y)
	case "dclick":
		err = mousesvc.DoubleClick(x, y)
	case "move":
		err = mousesvc.Move(x, y)
	}
	if err != nil {
		if action == "click" {
			return Result{Text: fmt.Sprintf("❌ Could not click at %d,%d.", x, y)}
		}
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{Text: tgfmt.OK()}
}

func (vm *ViewModel) drag(args []string) Result {
	if len(args) < 4 {
		return Result{Text: "Usage: /drag x1 y1 x2 y2"}
	}
	coords, err := parseInts(args[:4])
	if err != nil {
		return Result{Text: tgfmt.FailWith("drag", err)}
	}
	if err := mousesvc.Drag(coords[0], coords[1], coords[2], coords[3]); err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{Text: tgfmt.OK()}
}

func (vm *ViewModel) scroll(args []string) Result {
	if len(args) < 1 {
		return Result{Text: "Usage: /scroll `up|down|left|right` [steps]"}
	}
	dir := args[0]
	steps := 3
	if len(args) >= 2 {
		if n, err := strconv.Atoi(args[1]); err == nil && n > 0 {
			steps = n
		}
	}
	if err := mousesvc.Scroll(dir, steps); err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{Text: tgfmt.OK()}
}

func (vm *ViewModel) typeText(args []string) Result {
	if len(args) == 0 {
		return Result{Text: "Usage: /type `text`"}
	}
	text := strings.Join(args, " ")
	if err := kbsvc.Type(text); err != nil {
		return Result{Text: "❌ Could not type text. Is a window focused?"}
	}
	return Result{Text: tgfmt.OK()}
}

func (vm *ViewModel) pressKey(args []string) Result {
	if len(args) == 0 {
		return Result{Text: "Usage: /key `shortcut`  e.g. /key cmd+t"}
	}
	if err := kbsvc.PressKey(args[0]); err != nil {
		return Result{Text: fmt.Sprintf("❌ Could not press key %s. Check key name.", args[0])}
	}
	return Result{Text: tgfmt.OK()}
}

func (vm *ViewModel) openApp(args []string) Result {
	if len(args) == 0 {
		return Result{Text: "Usage: /open `AppName`"}
	}
	appName := strings.Join(args, " ")
	if err := appsvc.Open(appName); err != nil {
		return Result{Text: fmt.Sprintf("❌ Could not open %s. Is it installed?", appName)}
	}
	return Result{Text: tgfmt.OKWith(appName + " opened")}
}

func (vm *ViewModel) shell(args []string) Result {
	if len(args) == 0 {
		return Result{Text: "Usage: /shell `command`"}
	}
	cmdLine := strings.Join(args, " ")
	out, err := shellsvc.Run(cmdLine)
	if err != nil {
		if out != "" {
			return Result{Text: "⚠️ Command exited with error:\n" + tgfmt.Code(out)}
		}
		return Result{Text: "⚠️ Command exited with error:\n" + tgfmt.Fail(err)}
	}
	if out == "" {
		return Result{Text: tgfmt.OK()}
	}
	return Result{Text: tgfmt.Code(out)}
}

func (vm *ViewModel) readFile(args []string) Result {
	if len(args) == 0 {
		return Result{Text: "Usage: /read `path`"}
	}
	content, err := filesvc.Read(args[0])
	if err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{Text: tgfmt.Code(content)}
}

func (vm *ViewModel) listDir(args []string) Result {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	entries, err := filesvc.List(path)
	if err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{Text: tgfmt.List(entries)}
}

func (vm *ViewModel) getClipboard() Result {
	text, err := filesvc.GetClipboard()
	if err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	if text == "" {
		return Result{Text: "(clipboard is empty)"}
	}
	return Result{Text: tgfmt.Code(text)}
}

func (vm *ViewModel) setClipboard(args []string) Result {
	if len(args) == 0 {
		return Result{Text: "Usage: /copy `text`"}
	}
	if err := filesvc.SetClipboard(strings.Join(args, " ")); err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{Text: tgfmt.OK()}
}

func (vm *ViewModel) listApps() Result {
	apps, err := appsvc.List()
	if err != nil {
		return Result{Text: tgfmt.Fail(err)}
	}
	return Result{Text: tgfmt.List(apps)}
}

func (vm *ViewModel) status() Result {
	w, h, _ := screensvc.Size()
	screenSize := fmt.Sprintf("%d × %d", w, h)

	stats, err := systemsvc.GetStats()
	if err != nil {
		return Result{Text: tgfmt.StatusReport("—", "—", "—", screenSize)}
	}

	cpu := stats.CPUUsed
	if cpu == "" {
		cpu = "—"
	}
	if stats.CPUIdle != "" {
		cpu += fmt.Sprintf(" (%s idle)", stats.CPUIdle)
	}

	mem := stats.MemUsed
	if mem == "" {
		mem = "—"
	}
	if stats.MemUnused != "" {
		mem += fmt.Sprintf(" used, %s free", stats.MemUnused)
	} else {
		mem += " used"
	}

	activeApp := stats.ActiveApp
	if activeApp == "" {
		activeApp = "—"
	}

	return Result{Text: tgfmt.StatusReport(cpu, mem, activeApp, screenSize)}
}

// help returns the welcome + command list for /start and /help.
func (vm *ViewModel) help() Result {
	return Result{Text: tgfmt.Help()}
}

// ---- helpers ----

func parseXY(args []string) (int, int, error) {
	if len(args) < 2 {
		return 0, 0, fmt.Errorf("expected x y coordinates")
	}
	coords, err := parseInts(args[:2])
	if err != nil {
		return 0, 0, err
	}
	return coords[0], coords[1], nil
}

func parseInts(tokens []string) ([]int, error) {
	result := make([]int, len(tokens))
	for i, t := range tokens {
		n, err := strconv.Atoi(t)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %q", t)
		}
		result[i] = n
	}
	return result, nil
}
