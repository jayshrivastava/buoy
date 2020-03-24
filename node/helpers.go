package node

import (
	"time"
)

func formatTime(t time.Time) string {
	return t.In(time.Local).Format("03:04:05.000000")
}
