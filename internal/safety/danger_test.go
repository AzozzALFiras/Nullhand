package safety

import "testing"

func TestClassifyCommandFlagsDestructiveCommands(t *testing.T) {
	cases := []string{
		"rm notes.txt",
		"rm -rf ~/projects",
		"/bin/rm -r build",
		"env rm -rf /tmp/x",
		"env -i FOO=bar rm x",
		"kill -9 1234",
		"killall firefox",
		"pkill -f nullhand",
		"chmod -R 777 /home/me",
		"chown --recursive me:me /srv",
		"sed -i s/a/b/ config.ini",
		"sed -i.bak s/a/b/ config.ini",
		"sed --in-place=.bak s/a/b/ f",
		"find . -name *.log -delete",
		"find / -exec cat {} ;",
		"systemctl stop nginx",
		"systemctl --user restart pipewire",
		"systemctl poweroff",
		"apt remove firefox",
		"apt purge -y vim",
		"snap remove code",
		"dpkg -r somepkg",
		"dpkg --purge somepkg",
		"pip uninstall requests",
		"npm rm -g typescript",
		"yarn remove react",
		"git push --force origin main",
		"git push -f",
		"git push origin +main",
		"git push origin :old-branch",
		"git push --delete origin feature",
		"git reset --hard HEAD~3",
		"git clean -fd",
		"git -C /repo clean -xf",
		"git branch -D feature",
		"git branch -d --force feature",
		"git checkout -- .",
		"git checkout .",
		"git restore src/main.go",
		"git restore --staged --worktree f",
		"git stash drop",
		"git stash clear",
	}
	for _, c := range cases {
		reason, dangerous := ClassifyCommand(c)
		if !dangerous {
			t.Errorf("%q should need confirmation", c)
		} else if reason == "" {
			t.Errorf("%q flagged without a reason", c)
		}
	}
}

func TestClassifyCommandAllowsSafeCommands(t *testing.T) {
	cases := []string{
		"",
		"ls -la",
		"cat /etc/hostname",
		"df -h",
		"ps aux",
		"env",
		"env FOO=bar",
		"chmod +x run.sh",
		"chmod 644 notes.txt",
		"sed -n 1,10p file",
		"sed -e s/a/b/ file",
		"find . -name *.go",
		"systemctl status nginx",
		"systemctl list-units",
		"apt list --installed",
		"apt search vim",
		"dpkg -l",
		"pip list",
		"npm run build",
		"npm install",
		"git status",
		"git log --oneline",
		"git push",
		"git push origin main",
		"git pull --rebase",
		"git reset HEAD file",
		"git clean -n",
		"git branch -d merged-feature",
		"git checkout main",
		"git checkout -b feature",
		"git restore --staged file",
		"git stash",
		"git stash list",
		"git -c color.ui=always status",
	}
	for _, c := range cases {
		if reason, dangerous := ClassifyCommand(c); dangerous {
			t.Errorf("%q should run without confirmation, flagged as %q", c, reason)
		}
	}
}

func TestHasFlag(t *testing.T) {
	cases := []struct {
		args  []string
		names []string
		want  bool
	}{
		{[]string{"-rf"}, []string{"f"}, true},
		{[]string{"-D"}, []string{"d"}, false}, // case-sensitive
		{[]string{"--force=true"}, []string{"force"}, true},
		{[]string{"--forced"}, []string{"force"}, false},
		{[]string{"--", "-f"}, []string{"f"}, true}, // tokens after "--" are still scanned
		{[]string{"file-f"}, []string{"f"}, false},  // not a flag
		{[]string{"--f"}, []string{"f"}, false},     // single-letter names only match short flags
	}
	for _, c := range cases {
		if got := hasFlag(c.args, c.names...); got != c.want {
			t.Errorf("hasFlag(%q, %q) = %v, want %v", c.args, c.names, got, c.want)
		}
	}
}
