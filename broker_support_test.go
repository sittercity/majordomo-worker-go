package majordomo_worker

import (
	"syscall"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
)

func createBroker() testBroker {
	return testBroker{
		performReceive:     make(chan struct{}),
		receivedFromWorker: make(chan [][]byte, 5),
		sendToWorker:       make(chan [][]byte, 1),
		shutdown:           make(chan struct{}),
	}
}

type testBroker struct {
	performReceive     chan struct{}
	receivedFromWorker chan [][]byte
	sendToWorker       chan [][]byte
	shutdown           chan struct{}
}

func (b testBroker) run(ctx *zmq4.Context, brokerAddress string) {
	socket, err := ctx.NewSocket(zmq4.ROUTER)
	if err != nil {
		panic(err)
	}

	socket.SetLinger(0)
	socket.Bind(brokerAddress)

	var workerId []byte

	for {
		select {
		case workerMessage := <-b.sendToWorker:
			messageWithId := append([][]byte{workerId}, workerMessage...)

			for {
				_, err := socket.SendMessage(messageWithId)

				if err != zmq4.Errno(syscall.EINTR) {
					break
				}
			}
		case <-b.performReceive:
			message, err := socket.RecvMessageBytes(0)

			if err != nil {
				panic(err)
			}

			workerId = message[0]
			b.receivedFromWorker <- message
		case <-b.shutdown:
			socket.Close()
			break
		}
	}
}
