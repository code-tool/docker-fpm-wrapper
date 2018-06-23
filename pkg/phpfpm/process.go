package phpfpm

import (
	"os"
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"io"
)

type Process struct {
	cmd           *exec.Cmd
	shutdownDelay time.Duration
}

func NewProcess(
	fpmPath string,
	fpmConfigPath string,
	stdout io.Writer,
	stderr io.Writer,
	wrapperSocket string,
	shutdownDelay time.Duration,
	extraArgs ...string,
) *Process {
	cmd := exec.Command(fpmPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("FPM_WRAPPER_SOCK=unix://%s", wrapperSocket))

	cmd.Args = append(cmd.Args, extraArgs...)
	cmd.Args = append(cmd.Args, "--nodaemonize")
	cmd.Args = append(cmd.Args, "--fpm-config", fpmConfigPath)

	return &Process{cmd: cmd, shutdownDelay: shutdownDelay}
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

		p.cmd.Process.Signal(sig)
	}
}

func (p *Process) Wait(errCh chan<- error) int {
	err := p.cmd.Wait()
	if err == nil {
		return 0
	}

	exitErr, ok := err.(*exec.ExitError)
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
