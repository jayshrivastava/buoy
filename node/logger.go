package node

import (
	"fmt"
	"os"
	"time"
)

type logger struct {
	nodeId int32
	stream chan string
}

func CreateLogger(nodeId int32) Logger {
	logger := logger{nodeId: nodeId, stream: make(chan string)}
	go logger.listen()
	return &logger
}

type Logger interface {
	Log(string)
}

func (l *logger) Log(message string) {
	l.stream <- fmt.Sprintf("%s: (Node %d) %s\n", formatTime(time.Now()), l.nodeId, message)
}

func (l *logger) listen() {
	f, _ := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()

	for {
		message := <-l.stream
		f.WriteString(message)
	}
}

/* CONSTANTS */
const (
	LOG_FILE = "log.txt"
)
