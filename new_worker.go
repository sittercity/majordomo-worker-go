//+build !test

package majordomo_worker

import (
	"github.com/pebbe/zmq4"
)

const (
	POLLING_INTERVAL   = 100
	HEARTBEAT_LIVENESS = 50
)

func NewWorker(brokerAddress, serviceName string, heartbeatInMillis, reconnectInMillis int, action WorkerAction) Worker {
	context, _ := zmq4.NewContext()

	worker := newWorker(
		context,
		brokerAddress,
		serviceName,
		heartbeatInMillis,
		reconnectInMillis,
		POLLING_INTERVAL,
		HEARTBEAT_LIVENESS,
		action,
	)

	return worker
}
