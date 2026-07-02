package stream

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

const (
	// scanBufInitial / scanBufMax size the line scanner. A single stream-json
	// line (a full assistant message or the result payload) can far exceed the
	// 64 KB bufio default, which would otherwise silently truncate it.
	scanBufInitial = 1 << 20 // 1 MB
	scanBufMax     = 8 << 20 // 8 MB
)

// Runner executes a subprocess and streams its stdout as normalized events.
type Runner struct {
	cmd     *exec.Cmd
	adapter Adapter
}

// NewRunner pairs a prepared *exec.Cmd (created with exec.CommandContext) with
// the adapter for its output format.
func NewRunner(cmd *exec.Cmd, a Adapter) *Runner {
	return &Runner{cmd: cmd, adapter: a}
}

// Start launches the process and returns a channel of normalized events. The
// channel is closed when the process exits or ctx is cancelled. A nonzero exit
// is surfaced as a trailing KindError event (with captured stderr) before close.
func (r *Runner) Start(ctx context.Context) (<-chan StreamEvent, error) {
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	var stderr bytes.Buffer
	r.cmd.Stderr = &stderr

	if err := r.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", r.name(), err)
	}

	out := make(chan StreamEvent, 64)
	go func() {
		defer close(out)

		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, scanBufInitial), scanBufMax)
		for sc.Scan() {
			line := bytes.TrimSpace(sc.Bytes())
			if len(line) == 0 {
				continue
			}
			ev, ok := r.adapter.Parse(line)
			if !ok {
				continue
			}
			// Copy the raw slice: the scanner reuses its buffer on the next Scan.
			ev.Raw = append([]byte(nil), ev.Raw...)
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}

		werr := r.cmd.Wait()
		if werr != nil && ctx.Err() == nil {
			out <- StreamEvent{
				Kind: KindError,
				Text: fmt.Sprintf("%s exited: %v\n%s", r.name(), werr, stderr.String()),
			}
		}
	}()

	return out, nil
}

func (r *Runner) name() string {
	if len(r.cmd.Args) > 0 {
		return r.cmd.Args[0]
	}
	return r.cmd.Path
}
