package majordomo_worker

import (
	"errors"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pebbe/zmq4"
)

const (
	MD_WORKER = "MDPW01"

	MD_READY      = "\x01"
	MD_REQUEST    = "\x02"
	MD_REPLY      = "\x03"
	MD_HEARTBEAT  = "\x04"
	MD_DISCONNECT = "\x05"
)

type WorkerAction interface {
	Call([][]byte) [][]byte
}

type Worker interface {
	Shutdown()
	Receive() ([][]byte, error)
}

func newWorker(context *zmq4.Context, brokerAddress, serviceName string, heartbeatInMillis, reconnectInMillis, pollInterval, heartbeatLiveness int, action WorkerAction) *mdWorker {
	w := &mdWorker{
		brokerAddress: brokerAddress,
		serviceName:   serviceName,
		heartbeat:     time.Duration(heartbeatInMillis) * time.Millisecond,
		reconnect:     time.Duration(reconnectInMillis) * time.Millisecond,
		pollInterval:  time.Duration(pollInterval) * time.Millisecond,
		context:       context,
		liveness:      heartbeatLiveness,
		workerAction:  action,
		shutdown:      make(chan bool),
	}

	w.reconnectToBroker()
	return w
}

type mdWorker struct {
	shutdown chan bool

	brokerAddress string
	serviceName   string

	heartbeat    time.Duration
	reconnect    time.Duration
	pollInterval time.Duration
	liveness     int
	heartbeatAt  time.Time

	socket  *zmq4.Socket
	context *zmq4.Context

	workerAction WorkerAction
}

func (w *mdWorker) reconnectToBroker() (err error) {
	if w.socket != nil {
		w.socket.Close()
	}

	w.socket, _ = w.context.NewSocket(zmq4.DEALER)
	w.socket.SetLinger(0)
	w.socket.Connect(w.brokerAddress)

	w.sendToBroker(MD_READY, []byte(w.serviceName), nil)

	w.heartbeatAt = time.Now().Add(w.heartbeat)

	return
}

func (w *mdWorker) sendToBroker(command string, serviceName []byte, msg [][]byte) error {
	_, err := w.socket.SendMessage("", []byte(MD_WORKER), []byte(command), serviceName, msg)
	return err
}

func (w *mdWorker) Shutdown() {
	logrus.Info("Performing graceful shutdown...")
	w.shutdown <- true
}

func (w *mdWorker) cleanup() {
	if w.socket != nil {
		w.socket.Close()
	}
	w.context.Term()
}

func (w *mdWorker) Receive() (msg [][]byte, err error) {
	for {
		select {
		case <-w.shutdown:
			w.cleanup()
			return msg, errors.New("Graceful shutdown")
		default:
			poll := zmq4.NewPoller()
			poll.Add(w.socket, zmq4.POLLIN)

			var polled []zmq4.Polled
			polled, err = poll.Poll(w.pollInterval)

			if err != nil {
				return
			}

			if len(polled) > 0 {
				msg, _ = w.socket.RecvMessageBytes(0)

				w.liveness = HEARTBEAT_LIVENESS

				if len(msg) < 3 {
					continue // ignore invalid messages
				}

				w.liveness = HEARTBEAT_LIVENESS

				switch command := string(msg[2]); command {
				case MD_REQUEST:
					logrus.Info("Got a request!")
					replyTo := msg[3]

					actionResponse := w.workerAction.Call(msg[5:])
					reply := [][]byte{nil}
					reply = append(reply, actionResponse...)

					w.sendToBroker(MD_REPLY, replyTo, reply)

					msg = actionResponse
					return
				case MD_DISCONNECT:
					logrus.Info("Got a disconnect from the broker")
					w.reconnectToBroker() // Initiate a reconnect, which basically resets the connection
				case MD_HEARTBEAT:
					logrus.Info("Got a heartbeat from the broker")
					// Do nothing, ANY message coming in acts as a heartbeat so we handle it above
				default:
					logrus.Info("Got something unexpected")
					// Do nothing, if we received something we don't recognize we'll just ignore it
				}
			} else if w.liveness--; w.liveness <= 0 {
				time.Sleep(w.reconnect)
				w.reconnectToBroker()
			}

			if w.heartbeatAt.Before(time.Now()) {
				w.sendToBroker(MD_HEARTBEAT, nil, nil)
				w.heartbeatAt = time.Now().Add(w.heartbeat)
			}
		}
	}
}
