package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FZambia/viper-lite"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
	"github.com/code-tool/docker-fpm-wrapper/pkg/util"
)

func init() {
	pflag.StringP("fpm", "f", "", "path to php-fpm")
	pflag.StringP("fpm-config", "y", "/etc/php/php-fpm.conf", "path to php-fpm config file")

	pflag.StringP("wrapper-socket", "s", "/tmp/fpm-wrapper.sock", "path to socket")

	pflag.String("listen", ":8080", "prometheus statistic addr")
	pflag.String("metrics-path", "/metrics", "prometheus statistic path")
	pflag.Duration("scrape-interval", time.Second, "fpm metrics scrape interval")

	pflag.Uint("line-buffer-size", 16*1024, "Max log line size (in bytes)")
	pflag.Duration("shutdown-delay", 500*time.Millisecond, "Delay before process shutdown")

	pflag.Parse()

	_ = viper.BindPFlags(pflag.CommandLine)
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

func main() {
	if viper.GetString("fpm") == "" {
		fmt.Println("php-fpm path not set")
		os.Exit(1)
	}

	var err error
	errCh := make(chan error, 1)
	stderr := util.NewSyncWriter(os.Stderr)

	dataListener := NewDataListener(
		viper.GetString("wrapper-socket"),
		util.NewReaderPool(viper.GetInt("line-buffer-size")),
		stderr,
		errCh)

	if err = dataListener.Start(); err != nil {
		fmt.Printf("Can't start listen: %v", err)
		os.Exit(1)
	}
	defer dataListener.Stop()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

	fpmProcess := phpfpm.NewProcess(
		viper.GetString("fpm"),
		viper.GetString("fpm-config"),
		os.Stdout,
		stderr,
		viper.GetString("wrapper-socket"),
		viper.GetDuration("shutdown-delay"),
		findFpmArgs()...,
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

	http.Handle(viper.GetString("metrics-path"), promhttp.Handler())
	go func() {
		errCh <- http.ListenAndServe(viper.GetString("listen"), nil)
	}()

	if viper.GetDuration("scrape-interval") > 0 {
		phpfpm.RegisterPrometheus(viper.GetString("fpm-config"), viper.GetDuration("scrape-interval"))
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
