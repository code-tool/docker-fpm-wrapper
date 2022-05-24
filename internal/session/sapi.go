package session

import (
	"strings"
)

type sapiInfo struct {
	ConfDir  string
	procName string
}

var sapis = []sapiInfo{
	{ConfDir: "apache2", procName: "apache2"},
	{ConfDir: "apache2filter", procName: "apache2"},
	{ConfDir: "cgi", procName: "php@VERSION@"},
	{ConfDir: "fpm", procName: "php-fpm@VERSION@"},
	{ConfDir: "cli", procName: "php@VERSION@"},
}

func (si sapiInfo) GetProcName(phpVersion string) string {
	return strings.Replace(si.procName, "@VERSION@", phpVersion, 1)
}
