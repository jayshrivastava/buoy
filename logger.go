package main

import (
	"fmt"
	"github.com/jayshrivastava/buoy/node"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func formatTime(t time.Time) string {
	return t.In(time.Local).Format("03:04:05.000000")
}

type logger struct {
	totalNodes int32
	stream     chan string
	terminated chan bool
}

func CreateLogger(totalNodes int32) node.Logger {
	logger := logger{
		totalNodes: totalNodes,
		stream:     make(chan string),
		terminated: make(chan bool, 1),
	}
	go logger.listen()
	go logger.terminationHandler()
	logger.stream <- fmt.Sprintf("<style>%s</style", CSS)
	logger.stream <- "<html>"
	logger.stream <- "<table>"

	message := "<tr><td></td>"
	for i := int32(0); i < logger.totalNodes; i++ {
		message = message + fmt.Sprintf("<td>%d</td>", i)
	}
	message = message + "</tr>"

	logger.stream <- fmt.Sprintf(message)

	return &logger
}

func (l *logger) terminationHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)

	_ = <-sigs

	l.stream <- TERMINATION_STR

	_ = <-l.terminated

	syscall.Kill(os.Getpid(), syscall.SIGTERM)
}

func (l *logger) Log(id int32, message string) {

	line := fmt.Sprintf("<tr><td>%s</td>", formatTime(time.Now()))
	for i := int32(0); i < l.totalNodes; i++ {
		if i == id {
			line = line + fmt.Sprintf("<td>%s</td>", message)
		} else {
			line = line + "<td></td>"
		}
	}
	line = line + "</tr>"

	l.stream <- line
}

func (l *logger) listen() {
	f, _ := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()

	for {
		message := <-l.stream

		if message == TERMINATION_STR {
			fmt.Fprintln(f, "</table>")
			fmt.Fprintln(f, "</html>")
			l.terminated <- true
			return
		}

		fmt.Fprintln(f, message)

	}
}

/* CONSTANTS */
const (
	LOG_FILE        = "log.html"
	TERMINATION_STR = "TERMINATE"
	CSS             = `
	table{
        border-collapse:collapse;
        border:1px solid #000000;
    }
    table td{
		border:1px solid #000000;
		max-width: 150px;
		padding: 5px;
    }
	`
)
