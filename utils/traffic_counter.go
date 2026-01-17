package utils

import (
	"time"
)

type TrafficCounter struct {
	upBytes   unit64
	downBytes unit64
	startTime time.Time
}
