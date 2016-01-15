package majordomo_worker

import (
	"fmt"
	"strings"
	"time"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
)

type WorkerConfig struct {
	BrokerAddress, ServiceName                            string
	HeartbeatInMillis, ReconnectInMillis, PollingInterval time.Duration
	MaxHeartbeatLiveness                                  int
	Action                                                WorkerAction
}

type mdWorker struct {
	shutdown chan bool

	brokerAddress string
	serviceName   string

	heartbeat        time.Duration
	reconnect        time.Duration
	pollInterval     time.Duration
	maxLivenessCount int
	heartbeatAt      time.Time

	sockets []mdWorkerSocket
	context *zmq4.Context

	workerAction WorkerAction
	logger       Logger
}

func newWorker(context *zmq4.Context, logger Logger, config WorkerConfig) (*mdWorker, error) {
	w := &mdWorker{
		context:          context,
		brokerAddress:    config.BrokerAddress,
		serviceName:      config.ServiceName,
		heartbeat:        config.HeartbeatInMillis,
		reconnect:        config.ReconnectInMillis,
		pollInterval:     config.PollingInterval,
		maxLivenessCount: config.MaxHeartbeatLiveness,
		workerAction:     config.Action,
		shutdown:         make(chan bool),
		logger:           logger,
	}

	err := w.connectToBroker()
	return w, err
}

func (w *mdWorker) Receive() (msg [][]byte, err error) {
	for {
		select {
		case <-w.shutdown:
			w.cleanup()
			return msg, GracefulShutdown("Graceful Shutdown")
		default:
			poller := zmq4.NewPoller()

			for _, workerSocket := range w.sockets {
				poller.Add(workerSocket.socket, zmq4.POLLIN)
			}

			var polledSockets []zmq4.Polled
			polledSockets, err = poller.Poll(w.pollInterval)

			if err != nil {
				logError(w.logger, fmt.Sprintf("Polling failed, error: %s", err.Error()))
				continue
			}

			if len(polledSockets) > 0 {
				for _, polledSocket := range polledSockets {
					msg, _ = polledSocket.Socket.RecvMessageBytes(0)

					if len(msg) < 3 {
						logError(w.logger, fmt.Sprintf("Received invalid message (not enough frames), received %d", len(msg)))
						continue // ignore invalid messages
					}

					polledWorkerSocket := w.findWorkerSocket(polledSocket.Socket)
					polledWorkerSocket.liveness = w.maxLivenessCount

					switch command := string(msg[2]); command {
					case MD_REQUEST:
						logDebug(w.logger, fmt.Sprintf("Received MD_REQUEST from broker with message '%q'", msg[5:]))
						replyTo := msg[3]

						actionResponse := w.workerAction.Call(msg[5:])
						reply := [][]byte{nil}
						reply = append(reply, actionResponse...)

						w.sendToBroker(polledWorkerSocket.socket, MD_REPLY, replyTo, reply)

						msg = actionResponse
						return
					case MD_DISCONNECT:
						logDebug(w.logger, "Received MD_DISCONNECT from broker")
						polledWorkerSocket.connect(w.maxLivenessCount) // Initiate a reconnect
						w.sendToBroker(polledWorkerSocket.socket, MD_READY, []byte(w.serviceName), nil)
					case MD_HEARTBEAT:
						// Do nothing, ANY message coming in acts as a heartbeat so we handle it above
						logDebug(w.logger, "Received MD_HEARTBEAT from broker")
					default:
						// Do nothing, if we received something we don't recognize we'll just ignore it
						logDebug(w.logger, fmt.Sprintf("Received unknown command of %s'", msg[2]))
					}
				}
			} else {
				for _, workerSocket := range w.sockets {
					if workerSocket.liveness--; workerSocket.liveness <= 0 {
						logWarn(w.logger, fmt.Sprintf("Worker at address '%s' has received nothing from the broker for %d polls, sleeping for %s and reconnecting", workerSocket.address, w.maxLivenessCount, w.reconnect))
						time.Sleep(w.reconnect)
						workerSocket.connect(w.maxLivenessCount)
						w.sendToBroker(workerSocket.socket, MD_READY, []byte(w.serviceName), nil)
					}
				}
			}

			for _, workerSocket := range w.sockets {
				if workerSocket.heartbeatAt.Before(time.Now()) {
					w.sendToBroker(workerSocket.socket, MD_HEARTBEAT, nil, nil)
					workerSocket.heartbeatAt = time.Now().Add(w.heartbeat)
				}
			}
		}
	}
}

func (w *mdWorker) Shutdown() {
	logDebug(w.logger, "Worker attempting graceful shutdown...")
	w.shutdown <- true
}

func (w *mdWorker) connectToBroker() (err error) {
	w.closeSockets()

	addresses := strings.Split(w.brokerAddress, ",")

	w.sockets = make([]mdWorkerSocket, 0)

	for _, address := range addresses {
		logDebug(w.logger, fmt.Sprintf("Attempting connection to broker at '%s'", address))

		heartbeatAt := time.Now().Add(w.heartbeat)

		workerSocket, err := createWorkerSocket(address, w.context, w.maxLivenessCount, heartbeatAt, w.logger)
		if err != nil {
			logError(w.logger, fmt.Sprintf("Error connecting to broker address '%s', error: '%s'", address, err.Error()))
			return err
		}

		w.sendToBroker(workerSocket.socket, MD_READY, []byte(w.serviceName), nil)
		logDebug(w.logger, fmt.Sprintf("Connected successfully to broker at '%s'", address))

		w.sockets = append(w.sockets, workerSocket)
	}

	return
}

func (w *mdWorker) sendToBroker(socket *zmq4.Socket, command string, serviceName []byte, msg [][]byte) error {
	workerMessage := [][]byte{[]byte(""), []byte(MD_WORKER), []byte(command)}

	if serviceName != nil {
		workerMessage = append(workerMessage, serviceName)
	}

	if msg != nil {
		workerMessage = append(workerMessage, msg...)
	}

	_, err := socket.SendMessage(workerMessage)

	logDebug(w.logger, fmt.Sprintf("Sent command '%s' to broker with message '%q'", command, msg))

	return err
}

func (w *mdWorker) findWorkerSocket(polledSocket *zmq4.Socket) mdWorkerSocket {
	var foundWorkerSocket mdWorkerSocket

	for _, workerSocket := range w.sockets {
		if workerSocket.socket == polledSocket {
			foundWorkerSocket = workerSocket
			break
		}
	}

	return foundWorkerSocket
}

func (w *mdWorker) closeSockets() {
	for _, workerSocket := range w.sockets {
		workerSocket.close()
	}
}

func (w *mdWorker) cleanup() {
	w.closeSockets()
	w.context.Term()
	logDebug(w.logger, "Worker socket and context closed successfully")
}
