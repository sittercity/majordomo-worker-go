package majordomo_worker

import (
	"time"
)

type WorkerAction interface {
	Call([][]byte) [][]byte
}

type Worker interface {
	Shutdown()
	Receive() ([][]byte, error)
}

type WorkerConfig struct {
	BrokerAddress, ServiceName                            string
	HeartbeatInMillis, ReconnectInMillis, PollingInterval time.Duration
	MaxHeartbeatLiveness                                  int
	Action                                                WorkerAction
}
