//+build !test

package majordomo_worker

import (
	"github.com/pebbe/zmq4"
)

func NewWorker(brokerAddress, serviceName string, heartbeatInMillis, reconnectInMillis, pollingInterval, maxHeartbeatLiveness int, action WorkerAction) Worker {
	context, _ := zmq4.NewContext()

	worker := newWorker(
		context,
		brokerAddress,
		serviceName,
		heartbeatInMillis,
		reconnectInMillis,
		pollingInterval,
		maxHeartbeatLiveness,
		action,
	)

	return worker
}
