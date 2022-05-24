package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/code-tool/docker-fpm-wrapper/pkg/util"
)

type DataListener struct {
	socketPath string
	listener   net.Listener
	rPool      *util.ReaderPool

	writer    io.Writer
	errorChan chan error
}

func NewDataListener(socketPath string, rPool *util.ReaderPool, writer io.Writer, errorChan chan error) *DataListener {
	return &DataListener{socketPath: socketPath, rPool: rPool, writer: writer, errorChan: errorChan}
}

func (l *DataListener) handleConnection(conn net.Conn) {
	reader := l.rPool.Get(conn)

	for {
		line, err := util.ReadLine(reader)

		if line != nil && len(line) > 0 {
			l.writer.Write(line)
		}

		if err != nil {
			if err != io.EOF {
				l.errorChan <- err
			}

			break
		}
	}

	l.rPool.Put(reader)
	_ = conn.Close()
}

func (l *DataListener) initSocket() error {
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

func (l *DataListener) acceptConnections() {
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			l.errorChan <- err
			return
		}

		go l.handleConnection(conn)
	}
}

func (l *DataListener) Start() error {
	err := l.initSocket()

	if err == nil {
		go l.acceptConnections()
	}

	return err
}

func (l *DataListener) Stop() {
	_ = l.listener.Close()
}
