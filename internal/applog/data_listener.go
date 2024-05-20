package applog

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/code-tool/docker-fpm-wrapper/internal/breader"
	"github.com/code-tool/docker-fpm-wrapper/pkg/line"
)

type SockDataListener struct {
	socketPath string
	listener   net.Listener
	rPool      *breader.Pool

	writer    io.Writer
	errorChan chan error
}

func NewSockDataListener(sockPath string, rPool *breader.Pool, writer io.Writer, errorChan chan error) *SockDataListener {
	return &SockDataListener{socketPath: sockPath, rPool: rPool, writer: writer, errorChan: errorChan}
}

func (l *SockDataListener) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := l.rPool.Get(conn)
	defer l.rPool.Put(reader)

	for {
		buf, err := line.ReadOne(reader, true)
		if len(buf) > 0 {
			_, _ = l.writer.Write(normalizeLine(buf))
		}

		if err == nil {
			continue
		}

		if err != io.EOF {
			l.errorChan <- err
		}

		break
	}
}

func (l *SockDataListener) initSocket() error {
	var err error
	var c net.Conn

	if _, err = os.Stat(l.socketPath); !os.IsNotExist(err) {
		// socket exists
		c, err = net.Dial("unix", l.socketPath)
		if err == nil {
			_ = c.Close()
			// socket exists and listening
			return errors.New(fmt.Sprintf("Socket %s already exists and listening", l.socketPath))
		}

		err = os.Remove(l.socketPath)
		if err != nil {
			return err
		}
	}

	l.listener, err = net.Listen("unix", l.socketPath)
	if err != nil {
		return err
	}

	return os.Chmod(l.socketPath, 0777)
}

func (l *SockDataListener) acceptConnections() {
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			l.errorChan <- err
			return
		}

		go l.handleConnection(conn)
	}
}

func (l *SockDataListener) Start() error {
	err := l.initSocket()

	if err == nil {
		go l.acceptConnections()
	}

	return err
}

func (l *SockDataListener) Stop() {
	_ = l.listener.Close()
}
