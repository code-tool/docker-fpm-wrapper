package phpfpm

import (
	"encoding/json"
	"io"
	"time"

	"github.com/tomasen/fcgi_client"
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

func GetStats(net, addr, statusPath string) (*Status, error) {
	fcgi, err := fcgiclient.DialTimeout(net, addr, time.Second)
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
	if err != nil {
		return nil, err
	}
	defer tryClose(resp.Body)

	s := Status{}
	err = json.NewDecoder(resp.Body).Decode(&s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func tryClose(closer io.ReadCloser) {
	if closer == nil {
		return
	}

	_ = closer.Close()
}
