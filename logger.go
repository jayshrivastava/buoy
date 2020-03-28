package main

import (
	"fmt"
	"github.com/jayshrivastava/buoy/node"
	"os"
	"text/tabwriter"
	"time"
)

func formatTime(t time.Time) string {
	return t.In(time.Local).Format("03:04:05.000000")
}

type logger struct {
	totalNodes int32
	counter    int32
	stream     chan string
}

func CreateLogger(totalNodes int32) node.Logger {
	logger := logger{totalNodes: totalNodes, counter: 0, stream: make(chan string)}
	go logger.listen()
	message := "\t"
	for i := int32(0); i < logger.totalNodes; i++ {
		message = message + fmt.Sprintf("%d\t", i)
	}
	logger.stream <- fmt.Sprintf(message)

	return &logger
}

func (l *logger) Log(id int32, message string) {

	line := fmt.Sprintf("%s\t", formatTime(time.Now()))
	for i := int32(0); i < l.totalNodes; i++ {
		if i == id {
			line = line + fmt.Sprintf("%s", message)
		}
		line = line + "\t"
	}

	l.stream <- line
}

func (l *logger) listen() {
	f, _ := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()

	w := tabwriter.NewWriter(f, 0, 0, 1, ' ', 0)

	for {
		message := <-l.stream

		fmt.Fprintln(w, message)
		l.counter += 1
		if l.counter == BUFFER_COUNT {
			w.Flush()
			l.counter = 0
		}

	}
}

/* CONSTANTS */
const (
	LOG_FILE     = "log.txt"
	BUFFER_COUNT = 100
)
