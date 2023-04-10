package main

import (
	"strings"
	"time"

	"github.com/FZambia/viper-lite"
	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/pflag"
)

type Config struct {
	LogLevel   int    `mapstructure:"log-level"`
	LogEncoder string `mapstructure:"log-encoder"`

	FpmPath       string `mapstructure:"fpm"`
	FpmConfigPath string `mapstructure:"fpm-config"`

	FpmSlowlogProxyDisabled bool `mapstructure:"fpm-no-slowlog"`

	// Logging proxy section
	WrapperSocket  string `mapstructure:"wrapper-socket"`
	LineBufferSize int    `mapstructure:"line-buffer-size"`

	//
	Listen         string        `mapstructure:"listen"`
	MetricsPath    string        `mapstructure:"metrics-path"`
	ScrapeInterval time.Duration `mapstructure:"scrape-interval"`

	ShutdownDelay time.Duration `mapstructure:"shutdown-delay"`
}

func parseCommandLineFlags() {
	pflag.Int8("log-level", -1, "Log level. -1 debug ")
	pflag.String("log-encoder", "auto", "Internal logging encoder")

	pflag.StringP("fpm", "f", "", "path to php-fpm")
	pflag.StringP("fpm-config", "c", "/etc/php/php-fpm.conf", "path to php-fpm config file")
	pflag.Bool("fpm-no-slowlog", false, "Disable php-fpm slowlog parsing and proxy")

	// Logging proxy section
	pflag.StringP("wrapper-socket", "s", "/tmp/fpm-wrapper.sock", "path to logging socket, set null to disable")
	pflag.Uint("line-buffer-size", 16*1024, "Max log line size (in bytes)")

	// Prom section
	pflag.String("listen", ":8080", "prometheus statistic addr")
	pflag.String("metrics-path", "/metrics", "prometheus statistic path")
	pflag.Duration("scrape-interval", time.Second, "fpm metrics scrape interval")

	pflag.Duration("shutdown-delay", 500*time.Millisecond, "Delay before process shutdown")

	pflag.Parse()
}

func parseAllFlags() error {
	parseCommandLineFlags()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return viper.BindPFlags(pflag.CommandLine)
}

func CreateConfigFromViper(v *viper.Viper) (*Config, error) {
	var conf Config

	err := v.UnmarshalExact(&conf)

	return &conf, err
}

func createConfig() (*Config, error) {
	if err := parseAllFlags(); err != nil {
		return nil, err
	}

	config, err := CreateConfigFromViper(viper.GetViper())
	if err != nil {
		return nil, err
	}

	return config, nil
}
