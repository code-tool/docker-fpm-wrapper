package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
)

type DataListener struct {
	socketPath string
	listener   net.Listener
	dataChan   chan string
	errorChan  chan error
}

func NewDataListener(socketPath string, dataChan chan string, errorChan chan error) *DataListener {
	return &DataListener{socketPath, nil, dataChan, errorChan}
}

func (l *DataListener) handleConnection(conn net.Conn) {
	buf := bytes.NewBuffer(make([]byte, 0, bytes.MinRead))

	_, err := buf.ReadFrom(conn)
	conn.Close()

	if err != nil {
		l.errorChan <- err
		return
	}

	l.dataChan <- buf.String()
}

func (l *DataListener) initSocket() error {
	var err error
	var c net.Conn

	if _, err = os.Stat(l.socketPath); !os.IsNotExist(err) {
		// socket exists
		c, err = net.Dial("unix", l.socketPath)
		if err == nil {
			c.Close()
			// socket exists and listening
			return errors.New(fmt.Sprintf("Socket %s already exists and listening", l.socketPath))
		}

		err = os.Remove(l.socketPath)
		if err != nil {
			return err
		}
	}

	l.listener, err = net.Listen("unix", l.socketPath)

	return err
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
	l.listener.Close()
}
