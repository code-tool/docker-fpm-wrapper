package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	fcgiclient "github.com/tomasen/fcgi_client"
)

const (
	fpmSocket   = "/var/run/php5-fpm.sock"
	phpTestFile = "/tmp/test.php"

	fpmWrapperSocket = "/tmp/fpm-wrapper.sock"
	fpmWrapperStderr = "/tmp/stderr"
)

const (
	testData1 = "en taro adun en taro tassadar en taro zeratul"
	testData2 = "power overwhelming\n"
)

func main() {
	//testFPMExecute()
	testWrapperSocket()
}

func testFPMExecute() {
	log.Println("testFPMExecute:")
	scriptFilename := phpTestFile
	if scriptFilename[0] != '/' {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalln("os.Getwd:", err)
		}
		scriptFilename = fmt.Sprintf("%s/%s", cwd, scriptFilename)
	}

	env := map[string]string{
		"SCRIPT_FILENAME": scriptFilename,
	}
	log.Println(env)

	fcgi, err := fcgiclient.Dial("unix", fpmSocket)
	if err != nil {
		log.Fatalln("fcgiclient.Dial:", err)
	}

	resp, err := fcgi.Get(env)
	if err != nil {
		log.Fatalln("fcgi.Get:", err)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("ioutil.ReadAll:", err)
	}

	if string(content) != testData1 {
		log.Fatalln("content not match:", string(content), "dump:\n", hex.Dump(content))
	}

	log.Println("content is match")
}

func testWrapperSocket() {
	log.Println("testWrapperSocket:")
	conn, err := net.Dial("unix", fpmWrapperSocket)
	defer conn.Close()
	if err != nil {
		log.Fatalln("net.Dial:", err)
	}

	_, err = conn.Write([]byte(testData2))
	if err != nil {
		log.Fatalln("conn.Write:", err)
	}
	err = conn.Close()
	if err != nil {
		log.Fatalln("conn.Close:", err)
	}

	time.Sleep(100 * time.Millisecond)
	stderrData, err := ioutil.ReadFile(fpmWrapperStderr)
	if err != nil {
		log.Fatalln("ioutil.ReadFile:", err)
	}

	if string(stderrData) != testData2 {
		log.Fatalln("content not match:", string(stderrData), "dump:\n", hex.Dump(stderrData))
	}
}
