package session

import (
	"regexp"

	"github.com/prometheus/procfs"
)

func pidOf(name string) ([]int, error) {
	re, err := regexp.Compile("(^|/)" + name + "$")
	if err != nil {
		return nil, err
	}

	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, err
	}

	var result []int
	for _, proc := range procs {
		executable, err := proc.Executable()
		if err != nil {
			continue
		}

		if !re.MatchString(executable) {
			continue
		}

		result = append(result, proc.PID)
	}

	return result, err
}
