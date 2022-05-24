package session

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/prometheus/procfs"
)

/*
SAPIS="apache2:apache2 apache2filter:apache2 cgi:php@VERSION@ fpm:php-fpm@VERSION@ cli:php@VERSION@"

# Iterate through all web SAPIs
(
proc_names=""
for version in $(/usr/sbin/phpquery -V); do
    for sapi in ${SAPIS}; do
		conf_dir=${sapi%%:*}
		proc_name=${sapi##*:}
		if [ -e "/etc/php/${version}/${conf_dir}/php.ini" ] && [ -x "/usr/bin/php${version}" ]; then
			# Get all session variables once so we don't need to start PHP to get each config option
			session_config=$(PHP_INI_SCAN_DIR=/etc/php/${version}/${conf_dir}/conf.d/ "/usr/bin/php${version}" -c "/etc/php/${version}/${conf_dir}/php.ini" -d "error_reporting='~E_ALL'" -r 'foreach(ini_get_all("session") as $k => $v) echo "$k=".$v["local_value"]."\n";')
			save_handler=$(echo "$session_config" | sed -ne 's/^session\.save_handler=\(.*\)$/\1/p')
			save_path=$(echo "$session_config" | sed -ne 's/^session\.save_path=\(.*;\)\?\(.*\)$/\2/p')
			gc_maxlifetime=$(($(echo "$session_config" | sed -ne 's/^session\.gc_maxlifetime=\(.*\)$/\1/p')/60))

			if [ "$save_handler" = "files" ] && [ -d "$save_path" ]; then
			proc_names="$proc_names $(echo "$proc_name" | sed -e "s,@VERSION@,$version,")";
			printf "%s:%s\n" "$save_path" "$gc_maxlifetime"
			fi
		fi
    done
done

# first find all open session files and touch them (hope it's not massive amount of files)
for pid in $(pidof $proc_names); do
    find "/proc/$pid/fd" -ignore_readdir_race -lname "$save_path/sess_*" -exec touch -c {} \; 2>/dev/null
done ) | \
    sort -rn -t: -k2,2 | \
    sort -u -t: -k 1,1 | \
    while IFS=: read -r save_path gc_maxlifetime; do
	# find all files older then maxlifetime and delete them
	find -O3 "$save_path/" -ignore_readdir_race -depth -mindepth 1 -name 'sess_*' -type f -cmin "+$gc_maxlifetime" -delete
    done

exit 0
*/

type Cleaner struct {
	cmd *exec.Cmd
}

func NewCleaner() *Cleaner {
	return &Cleaner{}
}

func (c *Cleaner) getPHPVersions() ([]string, error) {
	// /usr/sbin/phpquery -V
	cmd := exec.Command("/usr/sbin/phpquery")

	var stdoutBuff, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuff
	cmd.Stderr = &stderrBuf

	cmd.Args = append(cmd.Args, "-V")

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	result := strings.Split(stdoutBuff.String(), "\n")
	return result, nil
}

func (c *Cleaner) getPhpIniPath(phpVersion string, info sapiInfo) string {
	return fmt.Sprintf("/etc/php/%s/%s/php.ini", phpVersion, info.ConfDir)
}

func (c *Cleaner) getPhpPath(phpVersion string) string {
	return fmt.Sprintf("/usr/bin/php%s", phpVersion)
}

func (c *Cleaner) isPhpCliExists(phpVersion string, info sapiInfo) bool {
	_, err := os.Stat(c.getPhpIniPath(phpVersion, info))
	if err != nil {
		return false
	}

	_, err = os.Stat(c.getPhpPath(phpVersion))

	return err == nil
}

func (c *Cleaner) touchOpenFiles(procName string, savePath string) error {
	pids, err := pidOf(procName)
	if err != nil {
		return err
	}

	prefix := savePath + "/sess_"
	currentTime := time.Now().Local()

	for _, pid := range pids {
		proc, err := procfs.NewProc(pid)
		if err != nil {
			return err
		}

		targets, err := proc.FileDescriptorTargets()
		if err != nil {
			return nil
		}

		for _, target := range targets {
			if !strings.HasPrefix(target, prefix) {
				continue
			}

			stat, err := os.Stat(target)
			if err != nil || stat.IsDir() {
				continue
			}

			if err := os.Chtimes(target, currentTime, currentTime); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cleaner) removeSessionFiles(savePath string, cutoff time.Duration) error {
	fileInfo, err := ioutil.ReadDir(savePath)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, info := range fileInfo {
		if info.IsDir() || !strings.HasPrefix(info.Name(), "sess_") {
			continue
		}

		if diff := now.Sub(info.ModTime()); diff > cutoff {
			continue
		}

		if err := os.Remove(info.Name()); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cleaner) Cleanup() error {
	phpVersions, err := c.getPHPVersions()
	if err != nil {
		return err
	}

	for _, phpVersion := range phpVersions {
		for _, sapi := range sapis {
			if !c.isPhpCliExists(phpVersion, sapi) {
				continue
			}

			sessConfig, err := NewConfig(phpVersion, sapi.ConfDir)
			if err != nil {
				continue
			}

			if !sessConfig.IfSaveHandlerFilesAndPathNonEmpty() {
				continue
			}

			c.touchOpenFiles(sapi.GetProcName(phpVersion), sessConfig.SavePath)

			c.removeSessionFiles(sessConfig.SavePath, sessConfig.GcMaxLifetime)
		}
	}

	return nil
}

func (c *Cleaner) CleanupInfinity(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		return
	case <-ticker.C:
		_ = c.Cleanup()
	}
}
