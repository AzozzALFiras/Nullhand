package safety

import (
	"path"
	"strings"
)

// ClassifyCommand reports whether a shell command line is destructive enough
// to need an explicit /yes before it runs, and why. It sees the command the
// same way shell.Run does — whitespace-split tokens, no quoting or pipes — and
// catches the common ways to delete data, kill processes, change services or
// packages, and rewrite git history. It is a speed bump, not a sandbox:
// allowed interpreters like python3 can still do anything.
func ClassifyCommand(cmdLine string) (reason string, dangerous bool) {
	args := strings.Fields(cmdLine)
	// `env [-i] [NAME=value …] cmd …` runs cmd; judge the wrapped command.
	for len(args) > 0 && path.Base(args[0]) == "env" {
		args = args[1:]
		for len(args) > 0 && (strings.HasPrefix(args[0], "-") || strings.Contains(args[0], "=")) {
			args = args[1:]
		}
	}
	if len(args) == 0 {
		return "", false
	}

	name, rest := path.Base(args[0]), args[1:]
	switch name {
	case "rm", "shred":
		return name + " permanently deletes files", true
	case "dd", "truncate":
		return name + " can overwrite files or disks", true
	case "reboot", "shutdown", "poweroff", "halt":
		return name + " turns the machine off or restarts it", true
	case "kill", "killall", "pkill":
		return name + " terminates running processes", true
	case "chmod", "chown":
		if hasFlag(rest, "R", "recursive") {
			return name + " -R changes a whole directory tree", true
		}
	case "sed":
		if hasFlag(rest, "i", "in-place") {
			return "sed -i rewrites files in place", true
		}
	case "find":
		for _, a := range rest {
			switch a {
			case "-delete", "-exec", "-execdir", "-ok", "-okdir":
				return "find " + a + " can delete files or run other commands", true
			}
		}
	case "systemctl":
		if verb, ok := firstIn(rest, systemctlVerbs); ok {
			return "systemctl " + verb + " changes system services", true
		}
	case "apt", "snap":
		if verb, ok := firstIn(rest, packageVerbs); ok {
			return name + " " + verb + " changes installed packages", true
		}
	case "dpkg":
		if hasFlag(rest, "i", "r", "P", "install", "remove", "purge") {
			return "dpkg changes installed packages", true
		}
	case "pip", "pip3", "npm", "yarn", "gem", "cargo":
		if verb, ok := firstIn(rest, uninstallVerbs); ok {
			return name + " " + verb + " removes installed packages", true
		}
	case "git":
		return classifyGit(rest)
	}
	if strings.HasPrefix(name, "mkfs") {
		return name + " erases a filesystem", true
	}
	return "", false
}

var systemctlVerbs = setOf(
	"start", "stop", "restart", "try-restart", "reload", "reload-or-restart",
	"enable", "disable", "reenable", "mask", "unmask", "kill", "isolate",
	"set-default", "daemon-reload", "reboot", "poweroff", "halt", "suspend",
	"hibernate", "hybrid-sleep", "rescue", "emergency",
)

var packageVerbs = setOf(
	"install", "reinstall", "remove", "purge", "autoremove", "upgrade",
	"full-upgrade", "dist-upgrade", "refresh", "revert", "disable",
)

var uninstallVerbs = setOf("uninstall", "remove", "rm", "un")

// classifyGit flags the subcommands that throw away work or rewrite remote
// history. Read-only and additive commands (status, log, commit, pull…) pass.
func classifyGit(args []string) (string, bool) {
	sub, rest := gitSubcommand(args)
	switch sub {
	case "push":
		if hasFlag(rest, "f", "d", "force", "force-with-lease", "force-if-includes", "delete", "mirror", "prune") {
			return "git push with force/delete rewrites remote history", true
		}
		for _, a := range rest {
			if strings.HasPrefix(a, "+") || strings.HasPrefix(a, ":") {
				return "git push " + a + " force-updates or deletes a remote branch", true
			}
		}
	case "reset":
		if hasFlag(rest, "hard", "merge", "keep") {
			return "git reset --hard discards uncommitted changes", true
		}
	case "clean":
		if hasFlag(rest, "f", "force") {
			return "git clean -f deletes untracked files", true
		}
	case "branch":
		if hasFlag(rest, "D") || (hasFlag(rest, "d", "delete") && hasFlag(rest, "f", "force")) {
			return "git branch -D deletes a branch even if it is unmerged", true
		}
	case "checkout":
		for _, a := range rest {
			if a == "--" || a == "." {
				return "git checkout " + a + " discards local changes", true
			}
		}
		if hasFlag(rest, "f", "force") {
			return "git checkout --force discards local changes", true
		}
	case "restore":
		if !hasFlag(rest, "S", "staged") || hasFlag(rest, "W", "worktree") {
			return "git restore discards local changes", true
		}
	case "stash":
		if len(rest) > 0 && (rest[0] == "drop" || rest[0] == "clear") {
			return "git stash " + rest[0] + " deletes stashed changes", true
		}
	}
	return "", false
}

// gitSubcommand skips git's global options (including the value taken by
// -C and -c) and returns the subcommand plus its arguments.
func gitSubcommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-C" || a == "-c" {
			i++
			continue
		}
		if !strings.HasPrefix(a, "-") {
			return a, args[i+1:]
		}
	}
	return "", nil
}

// hasFlag reports whether args contain any of the given options. Names of
// one character match short flags, including inside bundles ("-rf" has "f");
// longer names match "--name" and "--name=value". Matching is case-sensitive
// because "-D" and "-d" mean different things.
func hasFlag(args []string, names ...string) bool {
	for _, a := range args {
		switch {
		case a == "--" || !strings.HasPrefix(a, "-"):
			continue
		case strings.HasPrefix(a, "--"):
			long, _, _ := strings.Cut(a[2:], "=")
			for _, n := range names {
				if len(n) > 1 && n == long {
					return true
				}
			}
		default:
			for _, n := range names {
				if len(n) == 1 && strings.Contains(a[1:], n) {
					return true
				}
			}
		}
	}
	return false
}

// firstIn returns the first non-flag argument that is in set.
func firstIn(args []string, set map[string]bool) (string, bool) {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && set[a] {
			return a, true
		}
	}
	return "", false
}

func setOf(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}
