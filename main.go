package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/FZambia/viper-lite"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"

	"github.com/code-tool/docker-fpm-wrapper/pkg/util"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func init() {
	pflag.StringP("fpm", "f", "", "path to php-fpm")
	pflag.StringP("fpm-config", "y", "/etc/php/php-fpm.conf", "path to php-fpm config file")

	pflag.StringP("wrapper-socket", "s", "/tmp/fpm-wrapper.sock", "path to socket")

	pflag.Duration("scrape-interval", time.Second, "fpm statuses update interval")

	pflag.String("listen", ":8080", "prometheus statistic addr")
	pflag.String("metrics-path", "/metrics", "prometheus statistic path")

	pflag.Uint("max-queue-size", 100, "Max pending logs size before trotling")
	pflag.Uint("line-buffer-size", 16*1024, "Max log line size (in bytes)")

	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
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
	dataChan := make(chan []byte, viper.GetInt("max-queue-size"))

	dataListener := NewDataListener(
		viper.GetString("wrapper-socket"),
		util.NewReaderPool(viper.GetInt("line-buffer-size")),
		dataChan,
		errCh)

	if err = dataListener.Start(); err != nil {
		fmt.Printf("Can't start listen: %v", err)
		os.Exit(1)
	}
	defer dataListener.Stop()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	signal.Notify(signalCh, os.Kill)

	cmd := exec.Command(viper.GetString("fpm"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("FPM_WRAPPER_SOCK=unix://%s", viper.GetString("wrapper-socket")))
	cmd.Args = append(cmd.Args, "--nodaemonize")
	cmd.Args = append(cmd.Args, "--fpm-config", viper.GetString("fpm-config"))
	cmd.Args = append(cmd.Args, findFpmArgs()...)

	if err = cmd.Start(); err != nil {
		fmt.Printf("exec.Command: %v", err)
		os.Exit(1)
	}

	go handleSignals(cmd, signalCh)
	procErrCh := make(chan error, 1)

	go func() {
		procErrCh <- cmd.Wait()
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
		case data := <-dataChan:
			os.Stderr.Write(data)
			os.Stderr.Write([]byte{'\n'})
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case err := <-procErrCh:
			if err == nil {
				os.Exit(0)
			}
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			} else {
				panic(err)
			}
		}
	}
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
