//+build !test

package majordomo_worker

import (
	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
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
