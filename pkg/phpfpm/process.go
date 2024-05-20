package phpfpm

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type Process struct {
	log *zap.Logger

	cmd           *exec.Cmd
	shutdownDelay time.Duration
}

func NewProcess(
	log *zap.Logger,
	fpmPath string, fpmConfigPath string,
	stdout io.Writer, stderr io.Writer,
	shutdownDelay time.Duration,
	env []string, extraArgs ...string,
) *Process {
	cmd := exec.Command(fpmPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	cmd.Env = env

	cmd.Args = append(cmd.Args, extraArgs...)
	cmd.Args = append(cmd.Args, "--nodaemonize")
	cmd.Args = append(cmd.Args, "--fpm-config", fpmConfigPath)

	return &Process{log: log, cmd: cmd, shutdownDelay: shutdownDelay}
}

func (p *Process) Start() error {
	return p.cmd.Start()
}

func (p *Process) HandleSignal(signalCh chan os.Signal) {
	for {
		sig := <-signalCh

		// k8s graceful shutdown impl
		if sig == syscall.SIGTERM {
			if p.shutdownDelay > 0 {
				<-time.After(p.shutdownDelay)
			}

			sig = syscall.SIGQUIT
		}

		if err := p.cmd.Process.Signal(sig); err != nil {
			p.log.Error("Failed to send signal to process", zap.Stringer("signal", sig), zap.Error(err))
		}
	}
}

func (p *Process) Wait(errCh chan<- error) int {
	err := p.cmd.Wait()
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	ok := errors.As(err, &exitErr)
	if !ok {
		errCh <- err
		return -1
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		errCh <- err
		return -1
	}

	if status.Exited() {
		return status.ExitStatus()
	}

	if status.Signaled() {
		return 128 + int(status.Signal())
	}

	return -1
}
