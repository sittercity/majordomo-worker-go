package majordomo_worker

import (
	"github.com/pebbe/zmq4"
)

func createWorker(ctx *zmq4.Context, brokerAddress, serviceName string, heartbeat, reconnect int, action WorkerAction) *mdWorker {
	return newWorker(
		ctx,
		brokerAddress,
		serviceName,
		heartbeat,
		reconnect,
		action,
	)
}

func sendWorkerMessage(broker testBroker, command string, parts ...[]byte) {
	data := [][]byte{[]byte(nil), []byte(MD_WORKER), []byte(command)}
	data = append(data, parts...)

	broker.sendToWorker <- data
}

type defaultWorkerAction struct{}

func (a defaultWorkerAction) Call(args []string) []string {
	return args
}

type funcWorkerAction struct {
	call func([]string) []string
}

func (f funcWorkerAction) Call(args []string) []string {
	return f.call(args)
}
