package majordomo_worker

import (
	"github.com/pebbe/zmq4"
)

func createWorker(ctx *zmq4.Context, brokerAddress, serviceName string, heartbeat, reconnect, pollInterval, heartbeatLiveness int, action WorkerAction) *mdWorker {
	return newWorker(
		ctx,
		brokerAddress,
		serviceName,
		heartbeat,
		reconnect,
		pollInterval,
		heartbeatLiveness,
		action,
	)
}

func sendWorkerMessage(broker testBroker, command string, parts ...[]byte) {
	data := [][]byte{[]byte(nil), []byte(MD_WORKER), []byte(command)}
	data = append(data, parts...)

	broker.sendToWorker <- data
}

type defaultWorkerAction struct{}

func (a defaultWorkerAction) Call(args [][]byte) [][]byte {
	return args
}

type funcWorkerAction struct {
	call func([][]byte) [][]byte
}

func (f funcWorkerAction) Call(args [][]byte) [][]byte {
	return f.call(args)
}
