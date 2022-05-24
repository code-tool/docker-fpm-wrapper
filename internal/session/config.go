package session

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	SavePath      string
	SaveHandler   string
	GcMaxLifetime time.Duration
}

func NewConfig(phpVersion, confDir string) (*Config, error) {
	//session_config=$(
	//	PHP_INI_SCAN_DIR=/etc/php/${version}/${conf_dir}/conf.d/
	//	"/usr/bin/php${version}"
	//	-c "/etc/php/${version}/${conf_dir}/php.ini"
	//	-d "error_reporting='~E_ALL'"
	//	-r 'foreach(ini_get_all("session") as $k => $v) echo "$k=".$v["local_value"]."\n";'
	//)
	cmd := exec.Command(fmt.Sprintf("/usr/bin/php%s", phpVersion))

	var stdoutBuff, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuff
	cmd.Stderr = &stderrBuf

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PHP_INI_SCAN_DIR=/etc/php/%s/%s/conf.d/", phpVersion, confDir))
	cmd.Args = append(cmd.Args, "-c", fmt.Sprintf("/etc/php/%s/%s/php.ini", phpVersion, confDir))
	cmd.Args = append(cmd.Args, "-d", "error_reporting='~E_ALL'")
	cmd.Args = append(cmd.Args, "-r", `foreach(ini_get_all('session') as $k => $v) echo $k, '=', $v['local_value'], PHP_EOL;`)

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	result := &Config{}
	scanner := bufio.NewScanner(&stdoutBuff)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) != 2 {
			continue
		}

		switch parts[0] {
		case "session.save_path":
			result.SavePath = parts[1]
		case "session.save_handler":
			result.SaveHandler = parts[1]
		case "session.gc_maxlifetime":
			if len(parts[1]) == 0 {
				continue
			}

			gcMaxLifetimeSeconds, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, err
			}

			result.GcMaxLifetime = time.Second * time.Duration(gcMaxLifetimeSeconds)
		}
	}

	return result, scanner.Err()
}

func (c *Config) IfSaveHandlerFilesAndPathNonEmpty() bool {
	return c.SaveHandler == "files" && len(c.SavePath) > 0
}
