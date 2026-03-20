package journal

import (
	"bufio"
	"context"
	"os/exec"
)

type Follower struct {
	cancel context.CancelFunc
	Lines  <-chan string
}

func NewFollower(serviceName string) *Follower {
	ctx, cancel := context.WithCancel(context.Background())
	lines := make(chan string, 256)

	cmd := exec.CommandContext(ctx, "journalctl",
		"-u", serviceName,
		"-f",
		"--output=short-iso",
		"-n", "200",
		"--no-pager",
	)

	f := &Follower{
		cancel: cancel,
		Lines:  lines,
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		close(lines)
		return f
	}

	if err := cmd.Start(); err != nil {
		cancel()
		close(lines)
		return f
	}

	go func() {
		defer func() {
			close(lines)
			cmd.Wait()
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case lines <- scanner.Text():
			}
		}
	}()

	return f
}

func (f *Follower) Stop() {
	f.cancel()
}
