package majordomo_worker

import (
	"time"

	"github.com/pebbe/zmq4"
)

const (
	MD_WORKER = "MDPW01"

	MD_READY      = "\x01"
	MD_REQUEST    = "\x02"
	MD_REPLY      = "\x03"
	MD_HEARTBEAT  = "\x04"
	MD_DISCONNECT = "\x05"

	HEARTBEAT_LIVENESS = 3
)

type WorkerAction interface {
	Call([]string) []string
}

type Worker interface {
	Close()
	Receive() ([][]byte, error)
}

func newWorker(context *zmq4.Context, brokerAddress, serviceName string, heartbeatInMillis, reconnectInMillis int, action WorkerAction) *mdWorker {
	w := &mdWorker{
		brokerAddress: brokerAddress,
		serviceName:   serviceName,
		heartbeat:     time.Duration(heartbeatInMillis) * time.Millisecond,
		reconnect:     time.Duration(reconnectInMillis) * time.Millisecond,
		context:       context,
		liveness:      0,
		workerAction:  action,
	}

	w.reconnectToBroker()
	return w
}

type mdWorker struct {
	brokerAddress string
	serviceName   string

	heartbeat   time.Duration
	reconnect   time.Duration
	liveness    int
	heartbeatAt time.Time

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

	w.liveness = HEARTBEAT_LIVENESS
	w.heartbeatAt = time.Now().Add(w.heartbeat)

	return
}

func (w *mdWorker) sendToBroker(command string, serviceName []byte, msg [][]byte) error {
	_, err := w.socket.SendMessage("", []byte(MD_WORKER), []byte(command), serviceName, msg)
	return err
}

func (w *mdWorker) Close() {
	if w.socket != nil {
		w.socket.Close()
	}
	w.context.Term()
}

func (w *mdWorker) Receive() (msg [][]byte, err error) {
	for {
		poll := zmq4.NewPoller()
		poll.Add(w.socket, zmq4.POLLIN)

		var polled []zmq4.Polled
		polled, err = poll.Poll(10 * time.Millisecond)

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
				replyTo := msg[3]
				actionResponse := w.processRequest(msg[5:])
				w.sendToBroker(MD_REPLY, replyTo, actionResponse)

				msg = actionResponse
				return
			case MD_DISCONNECT:
				w.reconnectToBroker() // Initiate a reconnect, which basically resets the connection
			case MD_HEARTBEAT:
				// Do nothing, ANY message coming in acts as a heartbeat so we handle it above
			default:
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

func (w *mdWorker) processRequest(requestInput [][]byte) [][]byte {
	actionInput := make([]string, 0)
	for i := range requestInput {
		s := string(requestInput[i][:])
		actionInput = append(actionInput, s)
	}

	actionOutput := w.workerAction.Call(actionInput)

	reply := make([][]byte, len(actionOutput))
	for i := range actionOutput {
		reply = append(reply, []byte(actionOutput[i]))
	}

	return reply
}
