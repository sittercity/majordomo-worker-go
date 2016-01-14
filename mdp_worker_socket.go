package majordomo_worker

import (
	"time"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
)

type mdWorkerSocket struct {
	context     *zmq4.Context
	socket      *zmq4.Socket
	address     string
	liveness    int
	heartbeatAt time.Time
	logger      Logger
}

func createWorkerSocket(address string, context *zmq4.Context, liveness int, heartbeatAt time.Time, logger Logger) (mdWorkerSocket, error) {
	ws := mdWorkerSocket{
		address:     address,
		heartbeatAt: heartbeatAt,
		context:     context,
		logger:      logger,
	}

	err := ws.connect(liveness)
	if err != nil {
		return mdWorkerSocket{}, err
	}

	return ws, nil
}

func (ws *mdWorkerSocket) connect(liveness int) error {
	socket, _ := ws.context.NewSocket(zmq4.DEALER)
	socket.SetLinger(0)

	err := socket.Connect(ws.address)
	if err != nil {
		return err
	}

	ws.socket = socket
	ws.liveness = liveness

	return nil
}

func (ws *mdWorkerSocket) close() {
	if ws.socket != nil {
		ws.socket.Close()
	}
}
