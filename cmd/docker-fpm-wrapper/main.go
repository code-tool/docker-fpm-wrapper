package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/code-tool/docker-fpm-wrapper/internal/applog"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
	"github.com/code-tool/docker-fpm-wrapper/pkg/util"
)

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

func main() {
	cfg, err := createConfig()
	if err != nil {
		fmt.Println("Can't create app config: %w", err)
		os.Exit(1)
	}

	if cfg.FpmPath == "" {
		fmt.Println("php-fpm path not set")
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	stderr := util.NewSyncWriter(os.Stderr)

	env := os.Environ()

	if cfg.WrapperSocket != "null" {
		env = append(env, fmt.Sprintf("FPM_WRAPPER_SOCK=unix://%s", cfg.WrapperSocket))

		dataListener := applog.NewDataListener(cfg.WrapperSocket, util.NewReaderPool(cfg.LineBufferSize), stderr, errCh)

		if err = dataListener.Start(); err != nil {
			fmt.Printf("Can't start listen: %v", err)
			os.Exit(1)
		}

		defer dataListener.Stop()
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

	fpmProcess := phpfpm.NewProcess(
		cfg.FpmPath, cfg.FpmConfigPath,
		os.Stdout, stderr,
		cfg.ShutdownDelay,
		env, findFpmArgs()...,
	)

	if err = fpmProcess.Start(); err != nil {
		fmt.Printf("exec.Command: %v", err)
		os.Exit(1)
	}

	go fpmProcess.HandleSignal(signalCh)

	fpmExitCodeCh := make(chan int, 1)
	go func() {
		fpmExitCodeCh <- fpmProcess.Wait(errCh)
	}()

	http.Handle(cfg.MetricsPath, promhttp.Handler())
	go func() {
		errCh <- http.ListenAndServe(cfg.Listen, nil)
	}()

	if cfg.ScrapeInterval > 0 {
		err = phpfpm.RegisterPrometheus(cfg.FpmConfigPath, cfg.ScrapeInterval)
		if err != nil {
			fmt.Printf("Can't init prometheus collectior: %v", err)
			os.Exit(1)
		}
	}

	for {
		select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case exitCode := <-fpmExitCodeCh:
			os.Exit(exitCode)
		}
	}
}
