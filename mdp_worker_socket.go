package majordomo_worker

import (
	"time"

	"github.com/pebbe/zmq4"
)

type mdWorkerSocket struct {
	context               *zmq4.Context
	socket                *zmq4.Socket
	address               string
	maxLiveness, liveness int
	heartbeatAt           time.Time
	logger                Logger
}

func createWorkerSocket(address string, context *zmq4.Context, maxLiveness int, heartbeatAt time.Time, logger Logger) (mdWorkerSocket, error) {
	ws := mdWorkerSocket{
		address:     address,
		heartbeatAt: heartbeatAt,
		context:     context,
		logger:      logger,
		maxLiveness: maxLiveness,
	}

	err := ws.connect()
	if err != nil {
		return mdWorkerSocket{}, err
	}

	return ws, nil
}

func (ws *mdWorkerSocket) connect() error {
	socket, _ := ws.context.NewSocket(zmq4.DEALER)
	socket.SetLinger(0)

	err := socket.Connect(ws.address)
	if err != nil {
		return err
	}

	ws.socket = socket
	ws.liveness = ws.maxLiveness

	return nil
}

func (ws *mdWorkerSocket) close() {
	if ws.socket != nil {
		ws.socket.Close()
	}
}
