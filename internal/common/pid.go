package common

import (
	"fmt"
)

type PID uint64

func (p PID) String() string {
	return fmt.Sprintf("%d", p)
}

var currentPID PID

func SetCurrentPID(pid PID) {
	currentPID = pid
}

func GetCurrentPID() PID {
	return currentPID
}
