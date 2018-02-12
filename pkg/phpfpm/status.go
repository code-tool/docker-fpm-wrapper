package phpfpm

import (
	"github.com/tomasen/fcgi_client"
	"time"
	"io"
	"io/ioutil"
	"encoding/json"
	"strings"
)

type Status struct {
	Name               string `json:"pool"`
	ProcessManager     string `json:"process manager"`
	StartTime          int    `json:"start time"`
	StartSince         int    `json:"start since"`
	AcceptedConn       int    `json:"accepted conn"`
	ListenQueue        int    `json:"listen queue"`
	MaxListenQueue     int    `json:"max listen queue"`
	ListenQueueLen     int    `json:"listen queue len"`
	IdleProcesses      int    `json:"idle processes"`
	ActiveProcesses    int    `json:"active processes"`
	TotalProcesses     int    `json:"total processes"`
	MaxActiveProcesses int    `json:"max active processes"`
	MaxChildrenReached int    `json:"max children reached"`
	SlowRequests       int    `json:"slow requests"`
}

func GetStats(listen, statusPath string) (*Status, error) {
	network := "tcp"
	if strings.Contains(listen, "/") && listen[0:1] != "[" {
		network = "unix"
	}

	fcgi, err := fcgiclient.DialTimeout(network, listen, time.Second)
	if err != nil {
		return nil, err
	}
	defer fcgi.Close()

	env := map[string]string{
		"QUERY_STRING":    "json&full",
		"SCRIPT_FILENAME": statusPath,
		"SCRIPT_NAME":     statusPath,
	}
	resp, err := fcgi.Get(env)
	if err != nil && err != io.EOF {
		return nil, err
	}
	defer tryClose(resp.Body)

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		return nil, err
	}
	s := Status{}
	err = json.Unmarshal(content, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func tryClose(closer io.ReadCloser) {
	if closer == nil {
		return
	}
	closer.Close()
}