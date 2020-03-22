package node

import (
	"fmt"
	"time"
)

type logger struct {
	nodeId int32
}

func CreateLogger(nodeId int32) Logger {
	return &logger{nodeId: nodeId}
}

type Logger interface {
	Log(string)
}

func (l *logger) Log(message string) {
	fmt.Printf("%s: (Node %d) %s\n", formatTime(time.Now()), l.nodeId, message)
}