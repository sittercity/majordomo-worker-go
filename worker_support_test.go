package majordomo_worker

import (
	"time"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
)

func createWorker(ctx *zmq4.Context, brokerAddress, serviceName string, heartbeat, reconnect, pollInterval, heartbeatLiveness int, action WorkerAction) *mdWorker {
	return newWorker(
		ctx,
		brokerAddress,
		serviceName,
		time.Duration(heartbeat)*time.Millisecond,
		time.Duration(reconnect)*time.Millisecond,
		time.Duration(pollInterval)*time.Millisecond,
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
