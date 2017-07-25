package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/FZambia/viper-lite"
	"github.com/spf13/pflag"
)

func init() {
	pflag.StringP("fpm", "f", "", "path to php-fpm")
	pflag.StringP("socket", "s", "", "path to socket")
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
}

func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	signal.Notify(signalCh, os.Kill)

	cmd := exec.Command(viper.GetString("fpm"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("LOGGER_SOCK_PATH=%s", viper.GetString("socket")))
	cmd.Args = append(cmd.Args, "--nodaemonize")
	cmd.Args = append(cmd.Args, findFpmArgs()...)

	err := cmd.Start()
	if err != nil {
		fmt.Printf("exec.Command: %v", err)
		os.Exit(1)
	}

	go handleSignals(cmd, signalCh)
	procErrCh := make(chan error, 1)
	go func() {
		procErrCh <- cmd.Wait()
	}()

	dataChan := make(chan string, 1)
	errChan := make(chan error, 1)
	dataListener := NewDataListener(viper.GetString("socket"), dataChan, errChan)

	dataListener.Start()
	defer dataListener.Stop()

	for {
		select {
		case data := <-dataChan:
			os.Stderr.WriteString(data)
		case err := <-errChan:
			if err != nil {
				panic(err)
			}
		case err := <-procErrCh:
			if err == nil {
				os.Exit(0)
			}
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			} else {
				panic(err)
			}
		}
	}
}

func findFpmArgs() []string {
	doubleDashIndex := -1

	for i := range os.Args {
		if os.Args[i] == "--" {
			doubleDashIndex = i
			break
		}
	}
	if doubleDashIndex == -1 || doubleDashIndex+1 == len(os.Args) {
		return nil
	}

	return os.Args[doubleDashIndex+1:]
}

func handleSignals(cmd *exec.Cmd, signalCh chan os.Signal) {
	for {
		err := cmd.Process.Signal(<-signalCh)
		if err != nil {
			fmt.Printf("cmd.Process.Signal: %v", err)
			os.Exit(1)
		}
	}
}
