//+build !test

package majordomo_worker

import (
	"github.com/pebbe/zmq4"
)

func NewWorker(brokerAddress, serviceName string, heartbeatInMillis, reconnectInMillis int, action WorkerAction) Worker {
	context, _ := zmq4.NewContext()

	worker := newWorker(context, brokerAddress, serviceName, heartbeatInMillis, reconnectInMillis, action)

	return worker
}
