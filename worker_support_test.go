package majordomo_worker

import (
	"time"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
)

func createWorker(ctx *zmq4.Context, brokerAddress, serviceName string, heartbeat, reconnect, pollInterval, heartbeatLiveness int, action WorkerAction, logger Logger) *mdWorker {
	config := WorkerConfig{
		BrokerAddress:        brokerAddress,
		ServiceName:          serviceName,
		HeartbeatInMillis:    time.Duration(heartbeat) * time.Millisecond,
		ReconnectInMillis:    time.Duration(reconnect) * time.Millisecond,
		PollingInterval:      time.Duration(pollInterval) * time.Millisecond,
		MaxHeartbeatLiveness: heartbeatLiveness,
		Action:               action,
	}

	w, err := newWorker(ctx, logger, config)
	if err != nil {
		panic(err)
	}

	return w
}

func sendWorkerMessage(broker testBroker, command string, parts ...[]byte) {
	data := [][]byte{[]byte(nil), []byte(MD_WORKER), []byte(command)}
	data = append(data, parts...)

	broker.sendToWorker <- data
}

func readUntilNonHeartbeat(broker testBroker) [][]byte {
	var workerMsg [][]byte
	for {
		broker.performReceive <- struct{}{}
		workerMsg = <-broker.receivedFromWorker

		if string(workerMsg[3]) != MD_HEARTBEAT {
			break
		}
	}

	return workerMsg
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
