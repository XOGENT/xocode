// Package runner constructs the exec.Cmd invocations for the two backing CLIs
// with the exact flags xocode relies on, and wires them for graceful
// cancellation.
package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// gracePeriod is how long a cancelled child gets to exit after SIGINT before
// it is force-killed.
const gracePeriod = 3 * time.Second

// prepare applies the cancellation and environment policy shared by both CLIs.
func prepare(cmd *exec.Cmd, workdir string) *exec.Cmd {
	cmd.Dir = workdir
	cmd.Env = childEnv()

	// On ctx cancel, send SIGINT first so the CLI can flush its stream, then
	// SIGKILL after the grace period.
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGINT) }
	cmd.WaitDelay = gracePeriod
	return cmd
}

// childEnv returns the environment for child processes, ensuring ~/.local/bin
// (where both `claude` and `cursor-agent` install) is on PATH even if the
// launching shell hadn't picked it up yet.
func childEnv() []string {
	env := os.Environ()
	home, err := os.UserHomeDir()
	if err != nil {
		return env
	}
	localBin := filepath.Join(home, ".local", "bin")
	for i, kv := range env {
		if len(kv) >= 5 && kv[:5] == "PATH=" {
			path := kv[5:]
			if !pathContains(path, localBin) {
				env[i] = "PATH=" + localBin + string(os.PathListSeparator) + path
			}
			return env
		}
	}
	return append(env, "PATH="+localBin)
}

func pathContains(path, dir string) bool {
	for _, p := range filepath.SplitList(path) {
		if p == dir {
			return true
		}
	}
	return false
}
